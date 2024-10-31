package order

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/rocky2015aaa/tokenswap-client/config"
	"github.com/rocky2015aaa/tokenswap-client/utils"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

const (
	flagID               = "id"
	flagChain            = "chain"
	flagNetwork          = "network"
	flagStatus           = "status"
	flagType             = "type"
	flagFeePayerType     = "fee_payer_type"
	flagPair             = "pair"
	flagMyOrder          = "myorder"
	flagReferral         = "referral"
	flagTypeValuePrivate = "private"
	flagTypeValueMainnet = "mainnet"

	fieldChains        = "chains"
	fieldPairs         = "pairs"
	fieldFeePayerTypes = "fee_payer_types"
	fieldTypes         = "types"
	fieldNetwork       = "networks"
	fieldVisibilities  = "visibilities"
	fieldStatuses      = "statuses"
	fieldFee           = "fee"
)

var (
	orderCommonInfo *OrderCommonInfo
	orderListCmd    = &cobra.Command{
		Use:   "list",
		Short: "List a trading order lists in the condition",
		Long:  `List a trading order lists in the condition`,
		Run: func(cmd *cobra.Command, args []string) {
			// TODO: pagination
			if len(orderID) > 0 {
				// Ensure no other flags or arguments are used
				if len(args) > 0 || cmd.Flags().NFlag() > 1 {
					fmt.Println("the --order-id flag must be used alone")
					return
				}
			}
			configData, err := config.ReadConfig()
			if err != nil {
				fmt.Println("Error while reading a config data")
				return
			}
			query := ""
			chain, _ := cmd.Flags().GetString(flagChain)
			if len(chain) > 0 {
				isValidChain := false
				for _, validChain := range orderCommonInfo.Chains {
					if validChain == chain {
						isValidChain = true
						break
					}
				}
				if !isValidChain {
					fmt.Println("Not a valid chain name")
					return
				}
				query += fmt.Sprintf("&%s=%s", flagChain, chain)
			}
			pair, _ := cmd.Flags().GetString(flagPair)
			if len(pair) > 0 {
				isValidPair := false
				for _, validPair := range orderCommonInfo.Pairs {
					if validPair == pair {
						isValidPair = true
						break
					}
				}
				if !isValidPair {
					fmt.Println("Not a valid order pair")
					return
				}
				query += fmt.Sprintf("&%s=%s", flagPair, pair)
			}
			network, _ := cmd.Flags().GetString(flagNetwork)
			if len(network) > 0 {
				if _, ok := orderCommonInfo.Networks[network]; !ok {
					fmt.Println("Not a valid network name")
					return
				}
				query += fmt.Sprintf("&%s=%s", flagNetwork, network)
			}
			orderType, _ := cmd.Flags().GetString(flagType)
			if len(orderType) > 0 {
				if _, ok := orderCommonInfo.Types[orderType]; !ok {
					fmt.Println("Not a valid order type")
					return
				}
				query += fmt.Sprintf("&%s=%s", flagType, orderType)
			}
			feePayerType, _ := cmd.Flags().GetString(flagFeePayerType)
			if len(feePayerType) > 0 {
				if _, ok := orderCommonInfo.FeePayerTypes[feePayerType]; !ok {
					fmt.Println("Not a valid fee payer type")
					return
				}
				query += fmt.Sprintf("&%s=%s", flagFeePayerType, feePayerType)
			}
			status, _ := cmd.Flags().GetString(flagStatus)
			if len(status) > 0 {
				if _, ok := orderCommonInfo.Statuses[status]; !ok {
					fmt.Println("Not a valid status")
					return
				}
				query += fmt.Sprintf("&%s=%s", flagStatus, status)
			}
			orderId, _ := cmd.Flags().GetString(flagID)
			if len(orderId) > 0 {
				query += fmt.Sprintf("&order_id=%s", orderId)
			}
			myOrder, _ := cmd.Flags().GetBool(flagMyOrder)
			if myOrder {
				query += fmt.Sprintf("&email=%s", configData.Email)
			}
			private, _ := cmd.Flags().GetBool(flagTypeValuePrivate)
			if !myOrder && private { // TODO: show all private or keep this way?
				fmt.Println("Only public orders can be allowed")
				return
			}
			if private {
				query += "&visibility=private"
			} else {
				query += "&visibility=public"
			}
			query = strings.Replace(query, "&", "?", 1)
			req, err := http.NewRequest("GET", config.tokenswapServerUrl+"/order/list"+query, nil)
			if err != nil {
				fmt.Println("Error while getting the order list")
				return
			}
			// Add the Bearer token to the Authorization header
			req.Header.Set("Authorization", "Bearer "+configData.AccessToken)
			response, err := utils.GetHttpResponse(req)
			if err != nil {
				fmt.Println("Error while getting the order list")
				return
			}
			if response.Success && response.Error == "" {
				dataList, ok := response.Data.([]interface{})
				if !ok {
					fmt.Println("Error while getting the order list")
					return
				}
				orders := []*OrderDetail{}
				for _, data := range dataList {
					order, err := parseOrderInfo(data.(map[string]interface{}))
					if err != nil {
						fmt.Println("Error while printing the order list")
						return
					}
					orders = append(orders, order)
				}
				fmt.Println("----[Order Information]-----")
				printOrderCommonInfo(orderCommonInfo)
				fmt.Println("--------[Order List]--------")
				if len(orders) == 0 {
					fmt.Println("There is no order.")
				} else {
					printOrders(orders)
				}
			} else if response.Error == utils.NoDocumentMessage {
				fmt.Println("There is no order list")
				return
			} else {
				fmt.Println("Error while getting the order list")
				return
			}
		},
	}

	OrderCmd = &cobra.Command{
		Use:   "order",
		Short: "The command for Token trading order",
		Long:  `The command for Token trading order`,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			err := config.ManageUserTokens()
			if err != nil {
				fmt.Println(err.Error())
				os.Exit(1)
			}
			orderCommonInfo, err = getOrderCommonInfo()
			if err != nil {
				fmt.Println("Error while getting the order common information")
				os.Exit(1)
			}
		},
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}
)

func parseOrderInfo(data map[string]interface{}) (*OrderDetail, error) {
	order := Order{}
	orderdetail := OrderDetail{}
	orderID, ok := data["id"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid '_id' in the order data response")
	}
	orderdetail.ID = orderID
	orderType, ok := data[flagType].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid 'type' in the order data response")
	}
	order.Type = orderType
	orderPair, ok := data[flagPair].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid 'pair' in the order data response")
	}
	order.Pair = orderPair
	orderAmount, ok := data["amount"].(float64)
	if !ok {
		return nil, fmt.Errorf("missing or invalid 'amount' in the order data response")
	}
	order.Amount = orderAmount
	orderPrice, ok := data["price"].(float64)
	if !ok {
		return nil, fmt.Errorf("missing or invalid 'price' in the order data response")
	}
	order.Price = orderPrice
	feePayerType, ok := data[flagFeePayerType].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid 'fee_payer_type' in the order data response")
	}
	order.FeePayerType = feePayerType
	chain, ok := data[flagChain].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid 'chain' in the order data response")
	}
	order.Chain = chain
	network, ok := data[flagNetwork].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid 'network' in the order data response")
	}
	order.Network = network
	referral, ok := data[flagReferral].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid 'referal' in the order data response")
	}
	order.Referral = referral
	status, ok := data[flagStatus].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid 'status' in the order data response")
	}
	order.Status = status
	createDateTime, ok := data["create_date_time"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid 'create_date_time' in the order data response")
	}
	orderdetail.CreateDateTime = createDateTime
	orderdetail.Order = &order
	return &orderdetail, nil
}

func printOrderCommonInfo(orderCommonInfo *OrderCommonInfo) {
	fmt.Printf("Order Fee: %.2f%%\n", orderCommonInfo.Fee)
}

func printOrders(orders []*OrderDetail) {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"ID", "Type", "Chain",
		"Network", "Pair", "Fee Payer Type",
		"Price", "Amount", "Referral",
		"Status", "Create Date Time"})

	// Customizing table appearance
	table.SetBorder(false)
	table.SetRowLine(true)
	table.SetAlignment(tablewriter.ALIGN_LEFT)

	for _, order := range orders {
		table.Append([]string{order.ID, order.Type, order.Chain,
			order.Network, order.Pair, order.FeePayerType,
			fmt.Sprintf("%.2f", order.Price), fmt.Sprintf("%.2f", order.Amount), order.Referral,
			order.Status, order.CreateDateTime})
	}

	table.Render()
}

func getOrderCommonInfo() (*OrderCommonInfo, error) {
	configData, err := config.ReadConfig()
	if err != nil {
		return nil, fmt.Errorf("error while reading a config data: %s", err)
	}
	req, err := http.NewRequest("GET", config.tokenswapServerUrl+"/order/common", nil)
	if err != nil {
		return nil, fmt.Errorf("error while getting the order common info: %s", err)
	}
	// Add the Bearer token to the Authorization header
	req.Header.Set("Authorization", "Bearer "+configData.AccessToken)
	response, err := utils.GetHttpResponse(req)
	if err != nil {
		return nil, fmt.Errorf("error while getting the order common info: %s", err)
	}
	commonData, ok := response.Data.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("error while getting the order common info")
	}
	orderFee, ok := commonData[fieldFee].(float64)
	if !ok {
		return nil, fmt.Errorf("error while getting fee data")
	}

	orderPairList, ok := commonData[fieldPairs].([]interface{})
	if !ok {
		return nil, fmt.Errorf("error while getting pairs data")
	}
	orderPairs := []string{}
	walletAddressList := map[string]string{}
	for _, orderPair := range orderPairList {
		orderPair := orderPair.(string)
		orderPairs = append(orderPairs, orderPair)
		availableTokens := strings.Split(orderPair, "/")
		for _, availableToken := range availableTokens {
			if availableToken == "XEL" {
				xelisWalletAddress, ok := commonData["xelis_wallet_address"].(string)
				if !ok {
					return nil, fmt.Errorf("error while getting xelis_wallet_address data")
				}
				if _, ok := walletAddressList[availableToken]; !ok {
					walletAddressList[availableToken] = xelisWalletAddress
				}
			}
			if availableToken == "USDT" {
				usdtWalletAddress, ok := commonData["usdt_wallet_address"].(string)
				if !ok {
					return nil, fmt.Errorf("error while getting xelis_wallet_address data")
				}
				if _, ok := walletAddressList[availableToken]; !ok {
					walletAddressList[availableToken] = usdtWalletAddress
				}
			}
			if availableToken == "USDC" {
				usdcWalletAddress, ok := commonData["usdc_wallet_address"].(string)
				if !ok {
					return nil, fmt.Errorf("error while getting xelis_wallet_address data")
				}
				if _, ok := walletAddressList[availableToken]; !ok {
					walletAddressList[availableToken] = usdcWalletAddress
				}
			}
		}
	}
	orderChainList, ok := commonData[fieldChains].([]interface{})
	if !ok {
		return nil, fmt.Errorf("error while getting chains data")
	}
	orderChains := []string{}
	for _, orderChain := range orderChainList {
		orderChains = append(orderChains, orderChain.(string))
	}

	depositTimeout, ok := commonData["deposit_timeout"].(float64)
	if !ok {
		return nil, fmt.Errorf("error while getting deposit_timeout data")
	}
	feePayerTypeList, ok := commonData[fieldFeePayerTypes].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("error while getting fee_payer_types data")
	}
	typeList, ok := commonData[fieldTypes].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("error while getting types data")
	}
	networkList, ok := commonData[fieldNetwork].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("error while getting networks data")
	}
	visibilityList, ok := commonData[fieldVisibilities].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("error while getting visibilities data")
	}
	statusList, ok := commonData[fieldStatuses].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("error while getting statuses data")
	}
	orderCommonInfo := OrderCommonInfo{
		Fee:             orderFee,
		Pairs:           orderPairs,
		Chains:          orderChains,
		WalletAddresses: walletAddressList,
		DepositTimeout:  int(depositTimeout),
		FeePayerTypes:   feePayerTypeList,
		Types:           typeList,
		Networks:        networkList,
		Visibilities:    visibilityList,
		Statuses:        statusList,
	}

	return &orderCommonInfo, nil
}

var orderID string

func init() {
	OrderCmd.AddCommand(orderListCmd)

	orderListCmd.Flags().String(flagChain, "", "List the order with a valid blockchain")
	orderListCmd.Flags().String(flagNetwork, flagTypeValueMainnet, "List the order with a valid network")
	orderListCmd.Flags().String(flagStatus, "", "List the order with a specific status")
	orderListCmd.Flags().String(flagType, "", "List the order with the order type")
	orderListCmd.Flags().String(flagFeePayerType, "", "List the order with the fee_payer_type")
	orderListCmd.Flags().String(flagPair, "", "List the order with the pair")
	orderListCmd.Flags().StringVar(&orderID, flagID, "", "List the order by order_id")
	orderListCmd.Flags().Bool(flagMyOrder, false, "List all my order in condition")
	orderListCmd.Flags().Bool(flagTypeValuePrivate, false, "Show the list orders(public/private)") // can be another name for boolean?
}
