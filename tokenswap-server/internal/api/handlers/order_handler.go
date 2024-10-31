package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/rocky2015aaa/tokenswap-server/internal/pkg/database"
	"github.com/rocky2015aaa/tokenswap-server/internal/pkg/utils"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

var (
	ErrOrderNotFound = errors.New("order not found")

	orderTypes         = map[string]struct{}{orderType1: {}, orderType2: {}}
	orderNetworkTypes  = map[string]struct{}{orderNetwork1: {}, orderNetwork2: {}}
	orderfeePayerTypes = map[string]struct{}{orderfeePayerType1: {}, orderfeePayerType2: {}, orderfeePayerType3: {}}
	orderStatusTypes   = map[string]struct{}{database.OrderStatusType1: {}, database.OrderStatusType2: {},
		database.OrderStatusType3: {}, database.OrderStatusType4: {}, database.OrderStatusType5: {}, database.OrderStatusType6: {}}
	orderVisibilityTypes = map[string]struct{}{orderVisibilityTypes1: {}, orderVisibilityTypes2: {}}
)

const (
	orderType1            = "buy"
	orderType2            = "sell"
	orderNetwork1         = "mainnet"
	orderNetwork2         = "testnet"
	orderfeePayerType1    = "split"
	orderfeePayerType2    = "buyer"
	orderfeePayerType3    = "seller"
	orderVisibilityTypes1 = "public"
	orderVisibilityTypes2 = "private"

	minimumOrderAmount = 10
	orderTimeout       = 120
)

func (h *Handler) GetOrderCommonInfo(ctx *gin.Context) {
	// TODO: how to create order_common_info collection
	uuidStr, err := getUUIDStrFromCtx(ctx)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, getResponse(false, nil, err.Error(), err.Error()))
		return
	}
	filter := bson.M{"uuid": uuidStr}
	_, err = h.getUserByFilter(filter)
	if err != nil {
		log.Error(err)
		if err == mongo.ErrNoDocuments {
			ctx.JSON(http.StatusNotFound, getResponse(false, nil, ErrUserNotFound.Error(), ErrUserNotFound.Error()))
			return
		}
		ctx.JSON(http.StatusInternalServerError, getResponse(false, nil, err.Error(), "Failed to get the user"))
		return
	}
	orderCommonInfo, err := h.getOrderCommonInfo()
	if err != nil {
		log.Error(err)
		ctx.JSON(http.StatusInternalServerError,
			getResponse(false, nil, err.Error(), "Getting the order common data has failed"))
		return
	}
	orderCommonInfo.Types = orderTypes
	orderCommonInfo.Networks = orderNetworkTypes
	orderCommonInfo.FeePayerTypes = orderfeePayerTypes
	orderCommonInfo.Statuses = orderStatusTypes
	orderCommonInfo.Visibility = orderVisibilityTypes
	ctx.JSON(http.StatusOK, getResponse(true, &orderCommonInfo, "", "Getting the order common data has succeeded"))
}

func (h *Handler) GetOrderList(ctx *gin.Context) {
	// TODO: pagination
	visibility := ctx.Query("visibility")
	if len(visibility) == 0 {
		visibility = " public"
	}
	email := ctx.Query("email")
	if len(email) == 0 && visibility == "private" {
		ctx.JSON(http.StatusBadRequest, getResponse(false, nil, "only public order is allowed", "only public order is allowed"))
		return
	}
	chain := ctx.Query("chain")
	if len(chain) > 0 {
		orderCommonInfo, err := h.getOrderCommonInfo()
		if err != nil {
			log.Error(err)
			ctx.JSON(http.StatusInternalServerError,
				getResponse(false, nil, err.Error(), "Getting the order common data has failed"))
			return
		}
		validChain := false
		for _, orderCommonInfoChain := range orderCommonInfo.Chains {
			if orderCommonInfoChain == chain {
				validChain = true
				break
			}
		}
		if !validChain {
			err := fmt.Errorf("the chain value in the request is invalid")
			ctx.JSON(http.StatusBadRequest, getResponse(false, nil, err.Error(), err.Error()))
			return
		}
	}
	pair := ctx.Query("pair")
	if len(pair) > 0 {
		orderCommonInfo, err := h.getOrderCommonInfo()
		if err != nil {
			log.Error(err)
			ctx.JSON(http.StatusInternalServerError,
				getResponse(false, nil, err.Error(), "Getting the order common data has failed"))
			return
		}
		validPair := false
		for _, orderCommonInfoPair := range orderCommonInfo.Pairs {
			if orderCommonInfoPair == pair {
				validPair = true
				break
			}
		}
		if !validPair {
			err := fmt.Errorf("the pair value in the request is invalid")
			ctx.JSON(http.StatusBadRequest, getResponse(false, nil, err.Error(), err.Error()))
			return
		}
	}
	network := ctx.Query("network")
	if len(network) > 0 {
		if _, ok := orderNetworkTypes[network]; !ok {
			err := fmt.Errorf("the network value in the request is invalid")
			log.Error(err)
			ctx.JSON(http.StatusBadRequest, getResponse(false, nil, err.Error(), err.Error()))
			return
		}
	} else {
		network = "mainnet"
	}
	feePayerType := ctx.Query("fee_payer_type")
	if len(feePayerType) > 0 {
		if _, ok := orderfeePayerTypes[feePayerType]; !ok {
			err := fmt.Errorf("the fee_payer_type value in the request is invalid")
			log.Error(err)
			ctx.JSON(http.StatusBadRequest, getResponse(false, nil, err.Error(), err.Error()))
			return
		}
	}
	orderType := ctx.Query("type")
	if len(orderType) > 0 {
		if _, ok := orderTypes[orderType]; !ok {
			err := fmt.Errorf("the order type value in the request is invalid")
			log.Error(err)
			ctx.JSON(http.StatusBadRequest, getResponse(false, nil, err.Error(), err.Error()))
			return
		}
	}
	status := ctx.Query("status")
	if len(status) > 0 {
		if _, ok := orderStatusTypes[status]; !ok {
			err := fmt.Errorf("the status value in the request is invalid")
			log.Error(err)
			ctx.JSON(http.StatusBadRequest, getResponse(false, nil, err.Error(), err.Error()))
			return
		}
	}
	orderId := ctx.Query("order_id")
	uuidStr, err := getUUIDStrFromCtx(ctx)
	if err != nil {
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
	filter = bson.M{"order.visibility": "public", "order.network": network}
	if len(orderId) > 0 {
		filter["id"] = orderId
	}
	if len(email) > 0 {
		if user.Email != email {
			ctx.JSON(http.StatusBadRequest, getResponse(false, nil, "Different user email", "Different user email"))
			return
		}
		filter["user_uuid"] = uuidStr
		filter["order.visibility"] = visibility
	}
	if len(chain) > 0 {
		filter["order.chain"] = chain
	}
	if len(pair) > 0 {
		filter["order.pair"] = pair
	}
	if len(orderType) > 0 {
		filter["order.type"] = orderType
	}
	if len(feePayerType) > 0 {
		filter["order.fee_payer_type"] = feePayerType
	}
	if len(status) > 0 {
		filter["order.status"] = status
	}
	orders := []*database.OrderData{}
	collection := h.Database.Database(database.tokenswapDatabase).Collection(database.OrderCollection)
	cursor, err := collection.Find(context.TODO(), filter)
	if err != nil {
		log.Error(err)
		if err == mongo.ErrNoDocuments {
			ctx.JSON(http.StatusNotFound, getResponse(false, nil, ErrOrderNotFound.Error(), ErrOrderNotFound.Error()))
			return
		}
		ctx.JSON(http.StatusInternalServerError,
			getResponse(false, nil, err.Error(), "Getting the order list data has failed"))
		return
	}
	if err = cursor.All(context.TODO(), &orders); err != nil {
		log.Error(err)
		ctx.JSON(http.StatusInternalServerError,
			getResponse(false, nil, err.Error(), "Getting the order list data has failed"))
	}
	ctx.JSON(http.StatusOK, getResponse(true, &orders, "", "Getting the order list data has succeeded"))
}

func (h *Handler) CreateOrder(ctx *gin.Context) {
	uuidStr, err := getUUIDStrFromCtx(ctx)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, getResponse(false, nil, err.Error(), err.Error()))
		return
	}
	req := OrderRequest{}
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
	err = h.orderRequestValidator(&req)
	if err != nil {
		log.Error(err)
		ctx.JSON(http.StatusBadRequest, getResponse(false, nil, err.Error(), err.Error()))
		return
	}

	// TODO: Create Order
	currentTime := time.Now()
	orderID, err := utils.GenerateID()
	if err != nil {
		log.Error(err)
		ctx.JSON(http.StatusInternalServerError,
			getResponse(false, nil, err.Error(), "Order creation has failed"))
		return
	}
	orderData := database.OrderData{
		ID:               orderID,
		UserUUID:         uuidStr,
		Order:            req.Order,
		CreationDateTime: currentTime.Format(database.TimeFormat),
		UpdateDateTime:   currentTime.Format(database.TimeFormat),
	}
	orderWallet := database.OrdererParticipantWallet{
		OrderID:                         orderID,
		OrdererParticipantWalletAddress: req.OrdererWalletAddress,
	}
	session, err := h.Database.StartSession()
	if err != nil {
		log.Error(err)
		ctx.JSON(http.StatusInternalServerError,
			getResponse(false, nil, err.Error(), "Order creation has failed"))
		return
	}
	err = session.StartTransaction()
	if err != nil {
		log.Error(err)
		ctx.JSON(http.StatusInternalServerError,
			getResponse(false, nil, err.Error(), "Order creation has failed"))
		return
	}
	err = mongo.WithSession(ctx, session, func(sc mongo.SessionContext) error {
		_, err := h.Database.Database(database.tokenswapDatabase).Collection(database.OrderCollection).InsertOne(context.TODO(), orderData)
		if err != nil {
			return err
		}

		_, err = h.Database.Database(database.tokenswapDatabase).Collection(database.OrderParticipantWalletCollection).InsertOne(context.TODO(), orderWallet)
		if err != nil {
			return err
		}

		err = session.CommitTransaction(sc)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		log.Error(err)
		ctx.JSON(http.StatusInternalServerError,
			getResponse(false, nil, err.Error(), "Order creation has failed"))
		return
	}
	session.EndSession(ctx)
	checkOrderTimeout(h, orderID, database.OrderStatusType1)
	ctx.JSON(http.StatusOK, getResponse(true, &orderData, "", "Creating an order has succeeded"))
}

func (h *Handler) TakeOrder(ctx *gin.Context) {
	uuidStr, err := getUUIDStrFromCtx(ctx)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, getResponse(false, nil, err.Error(), err.Error()))
		return
	}
	req := struct {
		OrderID           string `json:"order_id"`
		OrderTakerAddress string `json:"ordertaker_address"`
		Password          string `json:"password"`
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
	orderTakerWallet := database.OrdererParticipantWallet{
		OrderID:                         req.OrderID,
		OrdererParticipantWalletAddress: req.OrderTakerAddress,
	}
	session, err := h.Database.StartSession()
	if err != nil {
		log.Error(err)
		ctx.JSON(http.StatusInternalServerError,
			getResponse(false, nil, err.Error(), "Taking the order has failed"))
		return
	}
	err = session.StartTransaction()
	if err != nil {
		log.Error(err)
		ctx.JSON(http.StatusInternalServerError,
			getResponse(false, nil, err.Error(), "Taking the order has failed"))
		return
	}
	err = mongo.WithSession(ctx, session, func(sc mongo.SessionContext) error {
		filter := bson.M{"id": req.OrderID}
		err := database.UpdateOrderStatus(h.Database, filter, database.OrderStatusType3)
		if err != nil {
			return err
		}

		_, err = h.Database.Database(database.tokenswapDatabase).Collection(database.OrderParticipantWalletCollection).InsertOne(context.TODO(), orderTakerWallet)
		if err != nil {
			return err
		}
		err = session.CommitTransaction(sc)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		log.Error(err)
		if err == database.ErrNonUpdated {
			ctx.JSON(http.StatusNotFound,
				getResponse(false, nil, err.Error(), err.Error()))
			return
		}
		ctx.JSON(http.StatusInternalServerError,
			getResponse(false, nil, err.Error(), "Failed to update the order status"))
		return
	}
	session.EndSession(ctx)
	checkOrderTimeout(h, req.OrderID, database.OrderStatusType3)
	ctx.JSON(http.StatusOK, getResponse(true, nil, "", "Takeing the order has succeeded"))
}

func (h *Handler) CancelOrder(ctx *gin.Context) {
	uuidStr, err := getUUIDStrFromCtx(ctx)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, getResponse(false, nil, err.Error(), err.Error()))
		return
	}
	req := struct {
		OrderID  string `json:"order_id"`
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
	filter = bson.M{"id": req.OrderID, "user_uuid": uuidStr}
	err = database.UpdateOrderStatus(h.Database, filter, database.OrderStatusType5)
	if err != nil {
		log.Error(err)
		if err == database.ErrNonUpdated {
			ctx.JSON(http.StatusNotFound,
				getResponse(false, nil, err.Error(), err.Error()))
			return
		}
		ctx.JSON(http.StatusInternalServerError,
			getResponse(false, nil, err.Error(), "Failed to update the order status"))
		return
	}
	ctx.JSON(http.StatusOK, getResponse(true, nil, "", "Updating an order status has succeeded"))
}
