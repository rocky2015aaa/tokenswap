package handlers

import (
	"github.com/dgrijalva/jwt-go"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/rocky2015aaa/tokenswap-server/internal/pkg/database"
)

type Claims struct {
	Username string `json:"username"`
	jwt.StandardClaims
}

type UserConfigInformation struct {
	UUID                         string `json:"uuid"`
	TokenExpirationTimeInSeconds int    `json:"token_expiration_time_in_seconds"`
	TokenExpirationDateTime      string `json:"token_expiration_date_time"`
	RegistrationDateTime         string `json:"registration_date_time"`
}

type RenewTokensParam struct {
	GenerateRefreshToken               bool
	Filter                             primitive.M
	ReqPassword                        string
	AccessTokenExpirationTimeInSeconds int
}

type OrderRequest struct {
	*database.Order
	OrdererWalletAddress string `json:"orderer_wallet_address"`
	Password             string `json:"password"`
}
