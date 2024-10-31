//go:build !apitest

package api

import (
	"context"
	"net/http"
	"os"
	"strings"

	"github.com/rocky2015aaa/tokenswap-server/internal/api/handlers"
	"github.com/rocky2015aaa/tokenswap-server/internal/config"
	"github.com/rocky2015aaa/tokenswap-server/internal/pkg/database"
	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type TokenMonitor struct {
	TargetAddress string
	Monitor       func(ctx context.Context, db *mongo.Client, tokenswapAddress string, sig chan os.Signal)
}

func NewApp(sig chan os.Signal) *http.Server {
	ctx := context.Background()
	db, err := database.NewMongoDB(os.Getenv(config.EnvStsvrMongodbUri))
	if err != nil {
		log.Fatalln(err)
	}
	orderCommonInfo := database.OrderCommonInfo{}
	options := options.FindOneOptions{}
	collection := db.Database(database.tokenswapDatabase).Collection(database.OrderCommonInfoCollection)
	err = collection.FindOne(context.TODO(), bson.D{{}}, &options).Decode(&orderCommonInfo)
	if err != nil {
		log.Fatalln(err)
	}

	tokenList := map[string]*TokenMonitor{}
	for _, tokenPair := range orderCommonInfo.Pairs {
		availabletokens := strings.Split(tokenPair, "/")
		for _, token := range availabletokens {
			if _, exists := tokenList[token]; !exists {
				// if token == "XEL" {
				// 	tokenList[token] = &TokenMonitor{
				// 		TargetAddress: orderCommonInfo.XelisWalletAddress,
				// 		Monitor:       monitorXeltokenswapTranscations,
				// 	}
				// } else
				if token == "USDT" {
					tokenList[token] = &TokenMonitor{
						TargetAddress: orderCommonInfo.UsdtWalletAddress,
						Monitor:       monitorUSDTtokenswapTransactions,
					}
				}

			}
		}
	}
	for _, monitorToken := range tokenList {
		go func(monitorTransactions *TokenMonitor) {
			monitorTransactions.Monitor(ctx, db, monitorTransactions.TargetAddress, sig)
		}(monitorToken)
	}

	return &http.Server{
		Addr:    ":" + os.Getenv(config.EnvStsvrPort),
		Handler: NewRouter(handlers.NewHandler(db)),
	}
}
