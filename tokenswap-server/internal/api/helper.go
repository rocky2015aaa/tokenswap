package api

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/rocky2015aaa/tokenswap-server/internal/pkg/database"
	log "github.com/sirupsen/logrus"
	"github.com/xelis-project/xelis-go-sdk/wallet"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	ErrWrongAmount = errors.New("The amount is different")
)

const (
	EnvStsvrXelisWalletRPC      = "STSVR_XELIS_WALLET_RPC"
	EnvStsvrXelisWalletID       = "STSVR_XELIS_WALLET_ID"
	EnvStsvrXelisWalletPassword = "STSVR_XELIS_WALLET_PASSWORD"

	depositCheckTermSeconds = 10

	infuraURL   = "https://rpc.sepolia.org/"
	usdtAddress = "0xAA0d26EF9bCFD7536604017D5796109B1A12f844"
	usdcAddress = "0xb2619b4cDB731d32997f052BB432E46339e5e1C9"
)

func monitorXeltokenswapTranscations(ctx context.Context, db *mongo.Client, tokenswapAddress string, sig chan os.Signal) {
	ticker := time.NewTicker(depositCheckTermSeconds * time.Second)
	// TODO: config rpc uri and username/password
	xelisWallet, err := wallet.NewRPC(ctx, os.Getenv(EnvStsvrXelisWalletRPC), os.Getenv(EnvStsvrXelisWalletID), os.Getenv(EnvStsvrXelisWalletPassword))
	if err != nil {
		log.Fatal(err)
	}
	for {
		select {
		case <-ticker.C:
			txs, err := xelisWallet.ListTransactions(wallet.ListTransactionsParams{
				AcceptOutgoing: false,
				AcceptIncoming: true,
				AcceptCoinbase: false,
				AcceptBurn:     false,
			})
			fmt.Println("------- Xelis Transactions ------- ")
			if err != nil {
				log.Errorf("error while finding the order wallet transactions: %s", err.Error())
				continue
			}
			// TODO: Error handling(including timeout)
			for _, tx := range txs {
				orderData, err := updateOrderStatus(db, (*tx.Incoming).From, float64((*tx.Incoming).Transfers[0].Amount))
				if err != nil {
					if err == mongo.ErrNoDocuments {
						log.Infof("no the order wallet transactions to update: %s", err.Error())
					} else if err == ErrWrongAmount {
						log.Infof("not a correct order to update: %s", err.Error())
					} else {
						log.Errorf("error while finding the order wallet to update: %s", err.Error())
					}
					continue
				}
				log.Printf("TX Hash:%s, IncomingInfo: %+v, From:%s, Amount:%f, User Pair:%s, User Amount:%f", tx.Hash, (*tx.Incoming), (*tx.Incoming).From, float64((*tx.Incoming).Transfers[0].Amount)/10e7, orderData.Pair, orderData.Amount)
				log.Printf("order %s status has updated: %s", orderData.ID, orderData.Status)
			}
		case <-sig:
			log.Printf("Stopping XEL deposit checking.")
			return
		}
	}
}

func monitorUSDTtokenswapTransactions(ctx context.Context, db *mongo.Client, tokenswapAddress string, sig chan os.Signal) {
	// TODO: config rpc uri and username/password
	// ticker := time.NewTicker(depositCheckTermSeconds * time.Second)
	// client, err := ethclient.Dial(infuraURL)
	// if err != nil {
	// 	log.Fatalf("Failed to connect to the Ethereum client: %v", err)
	// }

	// usdtContractAddress := common.HexToAddress(usdtAddress)

	// //address := common.HexToAddress(tokenswapAddress)

	// header, err := client.HeaderByNumber(context.Background(), nil)
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// for {
	// 	select {
	// 	case <-ticker.C:
	// 		fmt.Println("------- USDT Transactions ------- ")
	// 		for i := header.Number; i.BitLen() >= 0; i.Sub(i, big.NewInt(1)) {
	// 			block, err := client.BlockByNumber(context.Background(), i)
	// 			if err != nil {
	// 				log.Fatal(err)
	// 			}

	// 			for _, tx := range block.Transactions() {
	// 				fmt.Printf("Transaction hash: %s\n", tx.Hash().Hex())
	// 				if tx.To() != nil && *tx.To() == tokenAddress {
	// 					fmt.Printf("Transaction hash: %s\n", tx.Hash().Hex())
	// 					fmt.Printf("To: %s\n", tx.To().Hex())
	// 					fmt.Printf("Value: %s\n", tx.Value().String())
	// 				}
	// 			}
	// 		}
	// 	case <-sig:
	// 		log.Printf("Stopping XEL deposit checking.")
	// 		return
	// 	}
	// }
}

func updateOrderStatus(db *mongo.Client, fromAddress string, amount float64) (*database.OrderData, error) {
	orderWallets := []*database.OrdererParticipantWallet{}
	filter := bson.M{"order_participant_wallet_address": fromAddress}
	options := options.FindOneOptions{}
	// TODO: define the action when the same user send same amount(now just findOne)
	collection := db.Database(database.tokenswapDatabase).Collection(database.OrderParticipantWalletCollection)
	cursor, err := collection.Find(context.TODO(), filter)
	if err != nil {
		return nil, err
	}
	if err = cursor.All(context.TODO(), &orderWallets); err != nil {
		return nil, err
	}
	if len(orderWallets) == 0 {
		return nil, mongo.ErrNoDocuments
	}
	found := false
	orderData := database.OrderData{}
	// TODO: one from wallet must create one order(create/take) per 10 mins
	for _, orderWallet := range orderWallets {
		// order create type sell
		filter = bson.M{"id": orderWallet.OrderID, "order.status": database.OrderStatusType1}
		collection = db.Database(database.tokenswapDatabase).Collection(database.OrderCollection)
		err = collection.FindOne(context.TODO(), filter, &options).Decode(&orderData)
		if err == nil {
			found = true
			break
		} else if err == mongo.ErrNoDocuments {
			// order take type buy
			filter = bson.M{"id": orderWallet.OrderID, "order.status": database.OrderStatusType3}
			collection = db.Database(database.tokenswapDatabase).Collection(database.OrderCollection)
			err = collection.FindOne(context.TODO(), filter, &options).Decode(&orderData)
			if err == nil {
				found = true
				break
			}
		}

	}
	if !found {
		return nil, mongo.ErrNoDocuments
	}
	if orderData.Amount != amount/10e7 {
		return nil, ErrWrongAmount
	}
	orderStatus := ""
	if orderData.Status == database.OrderStatusType1 {
		orderStatus = database.OrderStatusType2
	} else if orderData.Status == database.OrderStatusType3 {
		orderStatus = database.OrderStatusType6
	}
	filter = bson.M{"id": orderData.ID}
	err = database.UpdateOrderStatus(db, filter, orderStatus)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	orderData.Status = orderStatus
	return &orderData, nil
}
