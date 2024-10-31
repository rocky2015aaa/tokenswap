package handlers

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/rocky2015aaa/tokenswap-server/internal/pkg/database"
	"github.com/rocky2015aaa/tokenswap-server/internal/pkg/utils"
	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/crypto/bcrypt"
)

func createUser(currentTime time.Time, email, password string) (*database.User, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	return &database.User{
		UUID:                         uuid.New().String(),
		Email:                        email,
		Password:                     string(hashedPassword),
		TokenExpirationTimeInSeconds: jwtAccessTokenExpiration,
		RegistrationDateTime:         currentTime.Format(database.TimeFormat),
		UpdateDateTime:               currentTime.Format(database.TimeFormat),
	}, nil
}

func generateTokens(uuid string, generateRefreshToken bool, currentTime time.Time, accessTokenExpirationTime, refreshTokenExpirationTime int) (string, string, error) {
	// Access Token
	if accessTokenExpirationTime == 0 {
		accessTokenExpirationTime = jwtAccessTokenExpiration
	}
	accessTokenExpirationDateTime := currentTime.Add(time.Duration(accessTokenExpirationTime) * time.Second)
	accessClaims := &Claims{
		Username: uuid,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: accessTokenExpirationDateTime.Unix(),
		},
	}
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessTokenString, err := accessToken.SignedString(jwt_secret)
	if err != nil {
		return "", "", err
	}
	if !generateRefreshToken {
		return accessTokenString, "", nil
	}
	// Refresh Token
	if refreshTokenExpirationTime == 0 {
		refreshTokenExpirationTime = jwtRefreshTokenExpiration
	}
	refreshExpirationDateTime := currentTime.Add(time.Duration(refreshTokenExpirationTime) * time.Second)
	refreshClaims := &Claims{
		Username: uuid,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: refreshExpirationDateTime.Unix(),
		},
	}
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshTokenString, err := refreshToken.SignedString(jwt_secret)
	if err != nil {
		return "", "", err
	}
	return accessTokenString, refreshTokenString, nil
}

func getUUIDStrFromCtx(ctx *gin.Context) (string, error) {
	userUUID, ok := ctx.Get("uuid")
	if !ok {
		return "", fmt.Errorf("missing required field: uuid")
	}
	// Validate uuid format
	uuidStr := userUUID.(string)
	_, err := uuid.Parse(uuidStr)
	if err != nil {
		return "", fmt.Errorf(fmt.Sprintf("%s. Not a valid uuid format: %s", err.Error(), uuidStr))
	}
	return uuidStr, nil
}

func (h *Handler) getUserByFilter(filter primitive.M) (*database.User, error) {
	user := database.User{}
	options := options.FindOneOptions{}
	collection := h.Database.Database(database.tokenswapDatabase).Collection(database.UserCollection)
	err := collection.FindOne(context.TODO(), filter, &options).Decode(&user)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (h *Handler) getFilteredUserWithPassword(filter primitive.M, password string) (*database.User, error) {
	user, err := h.getUserByFilter(filter)
	if err != nil {
		return nil, err
	}
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		return nil, ErrIncorrectUserPassword
	}
	return user, nil
}

func (h *Handler) renewTokensAndUpdateExpirationTime(user *database.User, AccessTokenExpirationTimeInSeconds int, generateRefreshToken bool) (string, string, *mongo.UpdateResult, error) {
	currentTime := time.Now()
	accessToken, refreshToken, err := generateTokens(user.UUID, generateRefreshToken, currentTime, AccessTokenExpirationTimeInSeconds, 2*AccessTokenExpirationTimeInSeconds)
	if err != nil {
		return "", "", nil, fmt.Errorf("failed to generate tokens. %s", err.Error())
	}
	// Update user token expiration in DB
	updateData := bson.M{
		"$set": bson.M{
			"token_expiration_time_in_seconds": AccessTokenExpirationTimeInSeconds,
			"update_date_time":                 currentTime.Format(database.TimeFormat),
		},
	}
	updateResult, err := h.Database.Database(database.tokenswapDatabase).Collection(database.UserCollection).UpdateOne(context.TODO(), bson.M{"uuid": user.UUID}, updateData)
	if err != nil {
		return "", "", nil, fmt.Errorf("failed to Update user data with tokens failed. %s", err.Error())
	}
	return accessToken, refreshToken, updateResult, nil
}

func (h *Handler) getOrderCommonInfo() (*database.OrderCommonInfo, error) {
	orderCommonInfo := database.OrderCommonInfo{}
	options := options.FindOneOptions{}
	collection := h.Database.Database(database.tokenswapDatabase).Collection(database.OrderCommonInfoCollection)
	err := collection.FindOne(context.TODO(), bson.D{{}}, &options).Decode(&orderCommonInfo)
	if err != nil {
		return nil, err
	}
	return &orderCommonInfo, nil
}

func checkOrderTimeout(h *Handler, orderID, status string) {
	go func(h *Handler, orderID, status string) {
		timeoutCh := time.After(orderTimeout * time.Second)
		for {
			select {
			case <-timeoutCh:
				orderData := database.OrderData{}
				filter := bson.M{"id": orderID, "order.status": status}
				options := options.FindOneOptions{}
				collection := h.Database.Database(database.tokenswapDatabase).Collection(database.OrderCollection)
				err := collection.FindOne(context.TODO(), filter, &options).Decode(&orderData)
				if err != nil {
					log.Error(err)
					return
				}
				filter = bson.M{"id": orderData.ID}
				err = database.UpdateOrderStatus(h.Database, filter, database.OrderStatusType4)
				if err != nil {
					log.Error(err)
					return
				}
				log.Infof("Order %s has timed out.", orderData.ID)
				return
			}
		}
	}(h, orderID, status)
}

func (h *Handler) orderRequestValidator(req *OrderRequest) error {
	orderCommonInfo, err := h.getOrderCommonInfo()
	if err != nil {
		return err
	}
	validPair := false
	for _, pair := range orderCommonInfo.Pairs {
		if req.Pair == pair {
			validPair = true
			break
		}
	}
	if !validPair {
		return fmt.Errorf("the pair value in the request is invalid")
	}
	validChain := false
	for _, chain := range orderCommonInfo.Chains {
		if req.Chain == chain {
			validChain = true
			break
		}
	}
	if !validChain {
		return fmt.Errorf("the chain value in the request is invalid")
	}
	if _, ok := orderTypes[req.Type]; !ok {
		return fmt.Errorf("the type value in the request is invalid")
	}
	if _, ok := orderNetworkTypes[req.Network]; !ok {
		return fmt.Errorf("the network value in the request is invalid")
	}
	if _, ok := orderfeePayerTypes[req.FeePayerType]; !ok {
		return fmt.Errorf("the fee_payer_type value in the request is invalid")
	}
	if _, ok := orderVisibilityTypes[req.Visibility]; !ok {
		return fmt.Errorf("the visibility value in the request is invalid")
	}
	if _, ok := orderStatusTypes[req.Status]; !ok {
		return fmt.Errorf("the status value in the request is invalid")
	}
	if !(utils.ValidateDecimal1or2Places(req.Price) && req.Price > 0) {
		return fmt.Errorf("the price value in the request is invalid")
	}
	if !(utils.ValidateDecimal1or2Places(req.Amount) && req.Amount >= minimumOrderAmount) {
		return fmt.Errorf("the amount value in the request is invalid")
	}
	tokens := strings.Split(req.Pair, "/")
	tokenName := ""
	if req.Type == orderType1 {
		tokenName = tokens[1]
	} else if req.Type == orderType2 {
		tokenName = tokens[0]
	}
	if !utils.ValidateTokenAddress(tokenName, req.OrdererWalletAddress) {
		return fmt.Errorf("the token address the request is invalid")
	}
	// TODO: add referral validation
	return nil
}
