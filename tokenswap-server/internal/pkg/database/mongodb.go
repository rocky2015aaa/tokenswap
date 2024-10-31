package database

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	OrderIDdepositSig = make(chan string)
	ErrNonUpdated     = errors.New("update was not applied in the condition")
)

const (
	tokenswapDatabase = "tokenswap"

	OrderCollection                  = "orders"
	OrderCommonInfoCollection        = "order_common_info"
	OrderParticipantWalletCollection = "order_participant_wallets"
	ConfigCollection                 = "config"
	UserCollection                   = "users"

	OrderStatusType1 = "waitingForDeposit"
	OrderStatusType2 = "active"
	OrderStatusType3 = "takeInProgress"
	OrderStatusType4 = "timeout_cancelled"
	OrderStatusType5 = "cancelled"
	OrderStatusType6 = "completed"

	TimeFormat = "2006-01-02 15:04:05 MST"
)

func NewMongoDB(uri string) (*mongo.Client, error) {
	credential := options.Credential{
		Username: "tokenswap",
		Password: "1q2w3e4r",
	}

	clientOptions := options.Client().ApplyURI(uri).SetAuth(credential)
	client, err := mongo.Connect(context.Background(), clientOptions)
	if err != nil {
		return nil, err
	}
	err = client.Ping(context.Background(), nil)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func UpdateOrderStatus(db *mongo.Client, filter primitive.M, status string) error {
	updateData := bson.M{
		"$set": bson.M{
			"order.status":     status,
			"update_date_time": time.Now().Format(TimeFormat),
		},
	}
	updateResult, err := db.Database(tokenswapDatabase).Collection(OrderCollection).UpdateOne(context.TODO(), filter, updateData)
	if err != nil {
		return err
	}
	// Check if any document was modified
	if updateResult.MatchedCount == 0 {
		return ErrNonUpdated
	}
	return nil
}
