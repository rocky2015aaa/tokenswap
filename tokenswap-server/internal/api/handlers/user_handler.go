package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/asaskevich/govalidator"
	"github.com/rocky2015aaa/tokenswap-server/internal/pkg/database"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrUserNotFound           = errors.New("user not found")
	ErrInvalidUserEmailFormat = errors.New("not a valid email format")
	ErrIncorrectUserPassword  = errors.New("not a correct user password")
)

const ()

func (h *Handler) GetUserInfo(ctx *gin.Context) {
	uuidStr, err := getUUIDStrFromCtx(ctx)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, getResponse(false, nil, err.Error(), err.Error()))
		return
	}
	TokenExpirationDateTime, ok := ctx.Get("token_expiration_date_time")
	if !ok {
		err := fmt.Errorf("missing required field in ctx: token_expiration_date_time")
		log.Error(err)
		ctx.JSON(http.StatusBadRequest, getResponse(false, nil, err.Error(), err.Error()))
		return
	}
	filter := bson.M{"uuid": uuidStr}
	user, err := h.getUserByFilter(filter)
	if err != nil {
		log.Error(err)
		if err == mongo.ErrNoDocuments {
			ctx.JSON(http.StatusNotFound, getResponse(false, nil, ErrUserNotFound.Error(), ErrUserNotFound.Error()))
			return
		}
		ctx.JSON(http.StatusInternalServerError, getResponse(false, nil, err.Error(), "Failed to get the user"))
		return
	}
	response := UserConfigInformation{
		UUID:                         user.UUID,
		TokenExpirationTimeInSeconds: user.TokenExpirationTimeInSeconds,
		TokenExpirationDateTime:      TokenExpirationDateTime.(string),
		RegistrationDateTime:         user.RegistrationDateTime,
	}
	ctx.JSON(http.StatusOK, getResponse(true, &response, "", "Getting the user data has succeeded"))
}

func (h *Handler) Verification(ctx *gin.Context) {
	uuidStr, err := getUUIDStrFromCtx(ctx)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, getResponse(false, nil, err.Error(), err.Error()))
		return
	}
	req := struct {
		Password string `json:"password"`
	}{}
	err = ctx.BindJSON(&req)
	if err != nil {
		log.Error(err)
		ctx.JSON(http.StatusBadRequest,
			getResponse(false, nil, err.Error(), "Binding data has failed"))
		return
	}
	// Check if the user exists with a correct password
	filter := bson.M{"uuid": uuidStr}
	_, err = h.getFilteredUserWithPassword(filter, req.Password)
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
	ctx.JSON(http.StatusOK, getResponse(true, nil, "", "User verification has succeeded"))
}

func (h *Handler) Register(ctx *gin.Context) {
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
	// Check if the user was already registered
	filter := bson.M{"email": req.Email}
	existingUser, err := h.getUserByFilter(filter)
	if err != nil && err != mongo.ErrNoDocuments {
		log.Error(err)
		ctx.JSON(http.StatusInternalServerError, getResponse(false, nil, err.Error(), "Failed to check for existing user"))
		return
	}
	if existingUser != nil {
		ctx.JSON(http.StatusBadRequest, getResponse(false, nil, "The user already exists", "The user already exists"))
		return
	}
	// Create user with tokens
	currentTime := time.Now()
	user, err := createUser(currentTime, req.Email, req.Password)
	if err != nil {
		log.Error(err)
		ctx.JSON(http.StatusInternalServerError, getResponse(false, nil, err.Error(), "Failed to create user"))
		return
	}
	accessToken, refreshToken, err := generateTokens(user.UUID, true, currentTime, 0, 0)
	if err != nil {
		log.Error(err)
		ctx.JSON(http.StatusInternalServerError,
			getResponse(false, nil, err.Error(), "Failed to generate a token"))
		return
	}
	_, err = h.Database.Database(database.tokenswapDatabase).Collection(database.UserCollection).InsertOne(context.TODO(), user)
	if err != nil {
		log.Error(err)
		ctx.JSON(http.StatusInternalServerError,
			getResponse(false, nil, err.Error(), "Registration has failed"))
		return
	}
	data := struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}
	ctx.JSON(http.StatusOK, getResponse(true, &data, "", "Registration has succeeded"))
}

func (h *Handler) UpdatePassword(ctx *gin.Context) {
	uuidStr, ok := ctx.Get("uuid")
	if !ok {
		err := fmt.Errorf("missing required field in ctx: uuid")
		log.Error(err)
		ctx.JSON(http.StatusBadRequest, getResponse(false, nil, err.Error(), err.Error()))
		return
	}
	req := struct {
		Password    string `json:"password"`
		NewPassword string `json:"new_password"`
	}{}
	err := ctx.BindJSON(&req)
	if err != nil {
		log.Error(err)
		ctx.JSON(http.StatusBadRequest,
			getResponse(false, nil, err.Error(), "Binding data has failed"))
		return
	}
	if req.Password == "" || req.NewPassword == "" {
		ctx.JSON(http.StatusBadRequest, getResponse(false, nil, "Password cannot be empty", "Password cannot be empty"))
		return
	}
	// Check if the user exists with a correct password
	filter := bson.M{"uuid": uuidStr}
	_, err = h.getFilteredUserWithPassword(filter, req.Password)
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
	// Create a new hashed password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		log.Error(err)
		ctx.JSON(http.StatusInternalServerError,
			getResponse(false, nil, err.Error(), "Failed to hash the user's password"))
		return
	}
	// Update user new password in DB
	currentTime := time.Now()
	updateData := bson.M{
		"$set": bson.M{
			"password":         string(hashedPassword),
			"update_date_time": currentTime.Format(database.TimeFormat),
		},
	}
	updateResult, err := h.Database.Database(database.tokenswapDatabase).Collection(database.UserCollection).UpdateOne(context.TODO(), filter, updateData)
	if err != nil {
		log.Error(err)
		ctx.JSON(http.StatusInternalServerError,
			getResponse(false, nil, err.Error(), "Failed to Update user data with tokens failed"))
		return
	}
	// Check if any document was modified
	if updateResult.MatchedCount == 0 {
		log.Error(database.ErrNonUpdated)
		ctx.JSON(http.StatusNotFound,
			getResponse(false, nil, database.ErrNonUpdated.Error(), database.ErrNonUpdated.Error()))
		return
	}
	ctx.JSON(http.StatusOK, getResponse(true, nil, "", "Updating user's password has succeeded"))
}
