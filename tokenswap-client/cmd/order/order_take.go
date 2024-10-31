package order

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/rocky2015aaa/tokenswap-client/config"
	"github.com/rocky2015aaa/tokenswap-client/utils"
	"github.com/spf13/cobra"
)

var (
	orderTakeCmd = &cobra.Command{
		Use:   "take <order ID>",
		Short: "Take a trading order",
		Long:  `Take a trading order`,
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
		Args: cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				cmd.Help()
				return
			}
			orderID := args[0]
			// TODO: id formation verification
			configData, err := config.ReadConfig()
			if err != nil {
				fmt.Println("Error while reading a config data")
				return
			}
			userPassword, err := utils.InputPassword("Enter password: ")
			if err != nil {
				fmt.Println("Error while getting the user password")
				return
			}
			userVerficationReq := struct {
				Password string `json:"password"`
			}{
				Password: userPassword,
			}
			jsonData, err := json.Marshal(userVerficationReq)
			if err != nil {
				fmt.Println("Error while verifying the user")
				return
			}
			req, err := http.NewRequest("POST", config.tokenswapServerUrl+"/user/verfication", bytes.NewBuffer(jsonData))
			if err != nil {
				fmt.Println("Error while verifying the user")
				return
			}
			// Add the Bearer token to the Authorization header
			req.Header.Set("Authorization", "Bearer "+configData.AccessToken)
			req.Header.Set("Content-Type", "application/json")
			response, err := utils.GetHttpResponse(req)
			if err != nil {
				fmt.Println("Error while verifying the user")
				return
			}
			if !(response.Success && response.Error == "") {
				fmt.Println("Error while verifying the user")
				return
			}
			query := fmt.Sprintf("?order_id=%s&status=%s", orderID, orderStatusType2)
			req, err = http.NewRequest("GET", config.tokenswapServerUrl+"/order/list"+query, nil)
			if err != nil {
				fmt.Println("Error while getting the order list")
				return
			}
			// Add the Bearer token to the Authorization header
			req.Header.Set("Authorization", "Bearer "+configData.AccessToken)
			response, err = utils.GetHttpResponse(req)
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
				if len(orders) == 0 {
					fmt.Println("There is no order.")
				} else {
					printOrders(orders)
					fmt.Println("* Your addess:")
					reader := bufio.NewReader(os.Stdin)
					orderTakerWalletAddress, err := reader.ReadString('\n')
					if err != nil {
						fmt.Println("Error while getting the your address")
						return
					}
					orderTakerWalletAddress = strings.TrimSpace(orderTakerWalletAddress)
					availableTokens := strings.Split(orders[0].Pair, "/")
					orderTokenName := ""
					if orders[0].Type == orderTypeBuy {
						orderTokenName = availableTokens[1]
					} else if orders[0].Type == orderTypeSell {
						orderTokenName = availableTokens[0]
					}
					if !utils.ValidateTokenAddress(orderTokenName, orderTakerWalletAddress) {
						fmt.Println("Your token address is not valid")
						return
					}
					fmt.Println("Confirm order take ([yes/no]):")
					reader = bufio.NewReader(os.Stdin)
					confirmOrderTake, err := reader.ReadString('\n')
					if err != nil {
						fmt.Println("Error while getting taking the order confirmation")
						return
					}
					confirmOrderTake = strings.TrimSpace(confirmOrderTake)
					if confirmOrderTake == "yes" {
						// TODO: handle token transaction
						orderTakeReq := struct {
							OrderID           string `json:"order_id"`
							OrderTakerAddress string `json:"ordertaker_address"`
							Password          string `json:"password"`
						}{
							OrderID:           orderID,
							OrderTakerAddress: orderTakerWalletAddress,
							Password:          userPassword,
						}
						jsonData, err := json.Marshal(orderTakeReq)
						if err != nil {
							fmt.Println("Error while taking the order")
							return
						}
						req, err := http.NewRequest("PATCH", config.tokenswapServerUrl+"/order/take", bytes.NewBuffer(jsonData))
						if err != nil {
							fmt.Println("Error while taking the order")
							return
						}
						// Add the Bearer token to the Authorization header
						req.Header.Set("Authorization", "Bearer "+configData.AccessToken)
						req.Header.Set("Content-Type", "application/json")
						response, err := utils.GetHttpResponse(req)
						if err != nil {
							fmt.Println("Error while taking the order")
							return
						}
						if response.Success && response.Error == "" {
							walletAddress, exists := orderCommonInfo.WalletAddresses[orderTokenName]
							if !exists {
								fmt.Println("Your token is not valid to order")
								return
							}
							fmt.Println("Taking an order has succeeded")
							fmt.Println("Order updated in takeInProgress state!")
							fmt.Printf("You now have %d min to deposit funds.\n", orderCommonInfo.DepositTimeout)
							fmt.Printf("Please send  ## %s from your wallet %s:%s to the tokenswap wallet %s:%s\n",
								orderTokenName, orderTokenName, orderTakeReq.OrderTakerAddress, orderTokenName, walletAddress)
							// TODO: SHOW QRCODE of tokenswap wallet"
						} else {
							fmt.Println("Error while taking the order")
							return
						}
					}
				}
			}
		},
	}
)

func init() {
	OrderCmd.AddCommand(orderTakeCmd)

	// orderTakeCmd.Flags().String(flagChain, "", "get the order with a valid blockchain")
	// orderTakeCmd.Flags().String(flagNetwork, "mainnet", "get the order with a valid network")
}
