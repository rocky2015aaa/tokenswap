package handlers

import (
	"fmt"
	"net/http"
	"os"

	"github.com/asaskevich/govalidator"
	"github.com/rocky2015aaa/tokenswap-server/internal/config"
	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

var (
	jwt_secret = []byte(os.Getenv(config.EnvStsvrJwtSecret))
)

const (
	// Seconds
	jwtAccessTokenExpiration  = 1500
	jwtRefreshTokenExpiration = 3000
	minimumExpirationTime     = 60
)

func (h *Handler) RenewTokens(ctx *gin.Context) {
	req := struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}{}
	err := ctx.BindJSON(&req)
	if err != nil {
		log.Error(err)
		ctx.JSON(http.StatusBadRequest,
			getResponse(false, nil, err.Error(), "Binding data has failed"))
		return
	}
	if !govalidator.IsEmail(req.Email) {
		ctx.JSON(http.StatusBadRequest,
			getResponse(false, nil, ErrInvalidUserEmailFormat.Error(), fmt.Sprintf("%s: %s", ErrInvalidUserEmailFormat.Error(), req.Email)))
		return
	}
	// Check if the user exists with a correct password
	filter := bson.M{"email": req.Email}
	user, err := h.getFilteredUserWithPassword(filter, req.Password)
	if err != nil {
		log.Error(err)
		if err == mongo.ErrNoDocuments {
			ctx.JSON(http.StatusNotFound, getResponse(false, nil, ErrUserNotFound.Error(), ErrUserNotFound.Error()))
			return
		} else if err == ErrIncorrectUserPassword {
			ctx.JSON(http.StatusBadRequest, getResponse(false, nil, err.Error(), err.Error()))
			return
		}
		ctx.JSON(http.StatusInternalServerError, getResponse(false, nil, err.Error(), "Failed to get the user"))
		return
	}
	accessToken, refreshToken, updateResult, err := h.renewTokensAndUpdateExpirationTime(user, user.TokenExpirationTimeInSeconds, true)
	if err != nil {
		log.Error(err)
		ctx.JSON(http.StatusInternalServerError,
			getResponse(false, nil, err.Error(), "Failed to generate tokens"))
		return
	}
	// Check if any document was modified
	if updateResult.MatchedCount == 0 {
		ctx.JSON(http.StatusNotFound,
			getResponse(false, nil, "Update was not applied in the condition", "Update has failed"))
		return
	}
	data := struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}
	ctx.JSON(http.StatusOK, getResponse(true, &data, "", "Update access_token has succeeded"))
}

func (h *Handler) RenewTokensWithCustomExpiration(ctx *gin.Context) {
	uuidStr, err := getUUIDStrFromCtx(ctx)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, getResponse(false, nil, err.Error(), err.Error()))
		return
	}
	req := struct {
		Password                           string `json:"password"`
		AccessTokenExpirationTimeInSeconds int    `json:"access_token_expiration_time_in_seconds,omitempty"`
	}{}
	err = ctx.BindJSON(&req)
	if err != nil {
		log.Error(err)
		ctx.JSON(http.StatusBadRequest,
			getResponse(false, nil, err.Error(), "Binding data has failed"))
		return
	}
	// if a valid user request access_token_expiration_time_in_seconds is less than 60s
	if req.AccessTokenExpirationTimeInSeconds <= minimumExpirationTime {
		accessTokenExpirationTimeInSecondsValidationError := fmt.Errorf("access_token_expiration_time_in_seconds must be more than %ds", minimumExpirationTime)
		log.Error(accessTokenExpirationTimeInSecondsValidationError)
		ctx.JSON(http.StatusBadRequest, getResponse(false, nil, accessTokenExpirationTimeInSecondsValidationError.Error(), accessTokenExpirationTimeInSecondsValidationError.Error()))
		return
	}
	// Check if the user exists with a correct password
	filter := bson.M{"uuid": uuidStr}
	user, err := h.getFilteredUserWithPassword(filter, req.Password)
	if err != nil {
		log.Error(err)
		if err == mongo.ErrNoDocuments {
			ctx.JSON(http.StatusNotFound, getResponse(false, nil, ErrUserNotFound.Error(), ErrUserNotFound.Error()))
			return
		} else if err == ErrIncorrectUserPassword {
			ctx.JSON(http.StatusBadRequest, getResponse(false, nil, err.Error(), err.Error()))
			return
		}
		ctx.JSON(http.StatusInternalServerError, getResponse(false, nil, err.Error(), "Failed to get the user"))
		return
	}
	accessToken, refreshToken, updateResult, err := h.renewTokensAndUpdateExpirationTime(user, req.AccessTokenExpirationTimeInSeconds, true)
	if err != nil {
		log.Error(err)
		ctx.JSON(http.StatusInternalServerError,
			getResponse(false, nil, err.Error(), "Failed to generate tokens"))
		return
	}
	// Check if any document was modified
	if updateResult.MatchedCount == 0 {
		ctx.JSON(http.StatusNotFound,
			getResponse(false, nil, "Update was not applied in the condition", "Update has failed"))
		return
	}
	data := struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}
	ctx.JSON(http.StatusOK, getResponse(true, &data, "", "Update access_token has succeeded"))
}

func (h *Handler) Refresh(ctx *gin.Context) {
	req := struct {
		Password     string `json:"password"`
		RefreshToken string `json:"refresh_token"`
	}{}
	err := ctx.BindJSON(&req)
	if err != nil {
		log.Error(err)
		ctx.JSON(http.StatusBadRequest,
			getResponse(false, nil, err.Error(), "Binding data has failed"))
		return
	}
	if req.RefreshToken == "" {
		ctx.JSON(http.StatusBadRequest, getResponse(false, nil, "Missing refresh token", "Missing refresh token"))
		return
	}
	// Parse the user refresh token and check the token validity
	token, err := jwt.Parse(req.RefreshToken, func(token *jwt.Token) (interface{}, error) {
		return []byte(os.Getenv(config.EnvStsvrJwtSecret)), nil
	})
	if err != nil {
		log.Error(err)
		ctx.JSON(http.StatusUnauthorized, getResponse(false, nil, err.Error(), err.Error()))
		return
	}
	var uuid string
	if !token.Valid {
		ctx.JSON(http.StatusUnauthorized, getResponse(false, nil, "The refresh token is invalid", "The refresh token is invalid"))
		return
	} else {
		var ok bool
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			err := fmt.Errorf("invalid claims format")
			log.Error(err)
			ctx.JSON(http.StatusUnauthorized, getResponse(false, nil, err.Error(), err.Error()))
			return
		}
		uuid, ok = claims["username"].(string)
		if !ok {
			err := fmt.Errorf("missing required field in claims: username")
			log.Error(err)
			ctx.JSON(http.StatusUnauthorized, getResponse(false, nil, err.Error(), err.Error()))
			return
		}
	}
	// Check if the user exists with a correct password
	filter := bson.M{"uuid": uuid}
	user, err := h.getFilteredUserWithPassword(filter, req.Password)
	if err != nil {
		log.Error(err)
		if err == mongo.ErrNoDocuments {
			ctx.JSON(http.StatusNotFound, getResponse(false, nil, ErrUserNotFound.Error(), ErrUserNotFound.Error()))
			return
		} else if err == ErrIncorrectUserPassword {
			ctx.JSON(http.StatusBadRequest, getResponse(false, nil, err.Error(), err.Error()))
			return
		}
		ctx.JSON(http.StatusInternalServerError, getResponse(false, nil, err.Error(), "Failed to get the user"))
		return
	}
	accessToken, _, updateResult, err := h.renewTokensAndUpdateExpirationTime(user, user.TokenExpirationTimeInSeconds, false)
	if err != nil {
		log.Error(err)
		ctx.JSON(http.StatusInternalServerError,
			getResponse(false, nil, err.Error(), "Failed to generate tokens"))
		return
	}
	// Check if any document was modified
	if updateResult.MatchedCount == 0 {
		ctx.JSON(http.StatusNotFound,
			getResponse(false, nil, "Update was not applied in the condition", "Update has failed"))
		return
	}
	data := struct {
		AccessToken string `json:"access_token"`
	}{
		AccessToken: accessToken,
	}
	ctx.JSON(http.StatusOK, getResponse(true, &data, "", "Update access_token has succeeded"))
}
