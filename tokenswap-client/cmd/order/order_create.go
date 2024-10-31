package order

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/rocky2015aaa/tokenswap-client/config"
	"github.com/rocky2015aaa/tokenswap-client/utils"
	"github.com/spf13/cobra"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

const (
	orderTypeBuy     = "buy"
	orderTypeSell    = "sell"
	orderStatusType1 = "waitingForDeposit"
	orderStatusType2 = "active"
	orderStatusType3 = "takeInProgress"

	floatDecialmalregex = `^-?\d+(\.\d{1,2})?$`

	orderMinimumAmount = 10
)

var (
	orderCreateCmd = &cobra.Command{
		Use:   "create",
		Short: "Create a trading order",
		Long:  `Create a trading order`,
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
			if listPairs {
				// Ensure no other flags or arguments are used
				if len(args) > 0 || cmd.Flags().NFlag() > 1 {
					fmt.Println("the --list-pairs flag must be used alone")
					return
				}
				fmt.Println("Order Pair:")
				for _, pair := range orderCommonInfo.Pairs {
					fmt.Printf("- %s\n", pair)
				}
				return
			}
			configData, err := config.ReadConfig()
			if err != nil {
				fmt.Println("Error while reading a config file")
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
			orderReq := OrderRequest{
				Order: &Order{},
			}
			err = checkOrderOptions(&orderReq, cmd)
			if err != nil {
				// Handle error
				fmt.Printf("The flag has an invalid value: %s\n", err)
				return
			}
			fmt.Println("----[Creating an order]-----")
			fmt.Println("* Order Type")
			orderTypes := make([]string, 0, len(orderCommonInfo.Types))
			for orderType := range orderCommonInfo.Types {
				orderTypes = append(orderTypes, orderType)
			}
			sort.Strings(orderTypes)
			for idx, orderType := range orderTypes {
				fmt.Printf("%d) %s\n", idx+1, orderType)
			}
			fmt.Println("* Order Type Select:")
			reader := bufio.NewReader(os.Stdin)
			orderTypeSelection, err := reader.ReadString('\n')
			if err != nil {
				fmt.Println("Error while getting the order type")
				return
			}
			orderTypeSelection = strings.TrimSpace(orderTypeSelection)
			orderTypeSelectionNumber, err := strconv.Atoi(orderTypeSelection)
			if err != nil {
				// Handle error
				fmt.Printf("The order type selection is not a number: %s\n", orderTypeSelection)
				return
			}
			if orderTypeSelectionNumber <= 0 || orderTypeSelectionNumber-1 > len(orderCommonInfo.Types)-1 {
				fmt.Printf("The order type selection is out of the selection range: %d\n", orderTypeSelectionNumber)
				return
			}
			orderReq.Type = orderTypes[orderTypeSelectionNumber-1]

			fmt.Println("* Order Pair")
			for idx, pair := range orderCommonInfo.Pairs {
				fmt.Printf("%d) %s\n", idx+1, pair)
			}
			fmt.Println("* Order Pair Select:")
			reader = bufio.NewReader(os.Stdin)
			orderPairSelection, err := reader.ReadString('\n')
			if err != nil {
				fmt.Println("Error while getting the order pair")
				return
			}
			orderPairSelection = strings.TrimSpace(orderPairSelection)
			orderPairSelectionNumber, err := strconv.Atoi(orderPairSelection)
			if err != nil {
				// Handle error
				fmt.Printf("The order pair selection is not a number: %s\n", orderPairSelection)
				return
			}
			if orderPairSelectionNumber <= 0 || orderPairSelectionNumber-1 > len(orderCommonInfo.Pairs)-1 {
				fmt.Printf("The order pair selection is out of the selection range: %d\n", orderPairSelectionNumber)
				return
			}
			orderReq.Pair = orderCommonInfo.Pairs[orderPairSelectionNumber-1]

			fmt.Printf("* Order Amount (equal or more than %d):\n", orderMinimumAmount)
			reader = bufio.NewReader(os.Stdin)
			orderAmount, err := reader.ReadString('\n')
			if err != nil {
				fmt.Println("Error while getting the order amount")
				return
			}
			orderAmount = strings.TrimSpace(orderAmount)
			if !regexp.MustCompile(floatDecialmalregex).MatchString(orderAmount) {
				fmt.Println("Invalid order amount format. Please enter a number with exactly 2 decimal places.")
				return
			}
			// Convert to float64 after validation
			orderAmountNumber, err := strconv.ParseFloat(orderAmount, 64)
			if err != nil {
				fmt.Printf("The order amount is not a float number: %s\n", orderAmount)
				return
			}
			if orderAmountNumber < orderMinimumAmount {
				fmt.Printf("The order amount is must be bigger than %d: %.2f\n", orderMinimumAmount, orderAmountNumber)
				return
			}
			orderReq.Amount = orderAmountNumber

			fmt.Println("* Order Price (more than 0):")
			reader = bufio.NewReader(os.Stdin)
			orderPrice, err := reader.ReadString('\n')
			if err != nil {
				fmt.Println("Error while getting the order price")
				return
			}
			orderPrice = strings.TrimSpace(orderPrice)
			if !regexp.MustCompile(floatDecialmalregex).MatchString(orderPrice) {
				fmt.Println("Invalid order price format. Please enter a number with exactly 2 decimal places.")
				return
			}
			// Convert to float64 after validation
			orderPriceNumber, err := strconv.ParseFloat(orderPrice, 64)
			if err != nil {
				fmt.Printf("The order price is not a float number: %s\n", orderPrice)
				return
			}
			if orderPriceNumber <= 0 {
				fmt.Printf("The order price is must be bigger than 0: %f\n", orderPriceNumber)
				return
			}
			orderReq.Price = orderPriceNumber

			fmt.Println("* Fee Payer Type")
			feePayerTypes := make([]string, 0, len(orderCommonInfo.FeePayerTypes))
			for feePayerType := range orderCommonInfo.FeePayerTypes {
				feePayerTypes = append(feePayerTypes, feePayerType)
			}
			sort.SliceStable(feePayerTypes, func(i, j int) bool {
				return feePayerTypes[i] > feePayerTypes[j] // Reverse order for strings
			})
			for idx, feePayerType := range feePayerTypes {
				feePayerType = cases.Title(language.English).String(feePayerType)
				if idx == 0 {
					fmt.Printf("%d) %s a fee payment with buyer and seller both\n", idx+1, feePayerType)
				} else {
					fmt.Printf("%d) %s pays the fee\n", idx+1, feePayerType)
				}
			}
			fmt.Println("Fee Payer Type Select:")
			reader = bufio.NewReader(os.Stdin)
			feePayerTypeSelection, err := reader.ReadString('\n')
			if err != nil {
				fmt.Println("Error while getting the order payer type")
				return
			}
			feePayerTypeSelection = strings.TrimSpace(feePayerTypeSelection)
			feePayerTypeSelectionNumber, err := strconv.Atoi(feePayerTypeSelection)
			if err != nil {
				// Handle error
				fmt.Printf("The fee payer type selection is not a number: %s\n", feePayerTypeSelection)
				return
			}
			if feePayerTypeSelectionNumber <= 0 || feePayerTypeSelectionNumber-1 > len(orderCommonInfo.FeePayerTypes)-1 {
				fmt.Printf("The order pair selection is out of the selection range: %d\n", feePayerTypeSelectionNumber)
				return
			}
			orderReq.FeePayerType = feePayerTypes[feePayerTypeSelectionNumber-1]
			fmt.Println("* Your addess:")
			reader = bufio.NewReader(os.Stdin)
			ordererWalletAddress, err := reader.ReadString('\n')
			if err != nil {
				fmt.Println("Error while getting the your address")
				return
			}
			ordererWalletAddress = strings.TrimSpace(ordererWalletAddress)
			availableTokens := strings.Split(orderReq.Pair, "/")
			orderTokenName := ""
			if orderReq.Type == orderTypeBuy {
				orderTokenName = availableTokens[1]
			} else if orderReq.Type == orderTypeSell {
				orderTokenName = availableTokens[0]
			}
			if !utils.ValidateTokenAddress(orderTokenName, ordererWalletAddress) {
				fmt.Println("Your token address is not valid")
				return
			}
			orderReq.OrdererWalletAddress = ordererWalletAddress
			orderReq.Status = orderStatusType1
			fmt.Println("----[Order Summary]-----")
			printOrderCommonInfo(orderCommonInfo)
			fmt.Println("------------------------")
			fmt.Println("Order Type:", orderReq.Type)
			fmt.Println("Order Pair:", orderReq.Pair)
			fmt.Println("Amount:", orderReq.Amount)
			fmt.Println("Price:", orderReq.Price)
			fmt.Println("Fees Payer:", orderReq.FeePayerType)
			fmt.Println("Chain:", orderReq.Chain)
			fmt.Println("Network:", orderReq.Network)
			fmt.Println("Fees Payer:", orderReq.Visibility)
			fmt.Println("Referral:", orderReq.Referral)
			fmt.Println("Your wallet address:", orderReq.OrdererWalletAddress)
			walletAddress, exists := orderCommonInfo.WalletAddresses[orderTokenName]
			if !exists {
				fmt.Println("Your token is not valid to order")
				return
			}
			if orderReq.Type == orderTypeBuy {
				fmt.Printf("you will need to send a total of %.2f %s (including fees) to wallet %s on chain %s\n", orderReq.Amount, orderTokenName, walletAddress, orderReq.Chain)
			} else if orderReq.Type == orderTypeSell {
				fmt.Printf("you will need to send a total of %.2f %s to the wallet %s\n", orderReq.Amount, orderTokenName, walletAddress)
			}
			fmt.Println("Confirm order  ([yes/no]):")
			reader = bufio.NewReader(os.Stdin)
			confirmOrder, err := reader.ReadString('\n')
			if err != nil {
				fmt.Println("Error while getting order creation confirmation")
				return
			}
			confirmOrder = strings.TrimSpace(confirmOrder)
			if confirmOrder == "yes" {
				orderReq.Password = userPassword
				jsonData, err := json.Marshal(orderReq)
				if err != nil {
					fmt.Println("Error while creating the order")
					return
				}
				req, err := http.NewRequest("POST", config.tokenswapServerUrl+"/order", bytes.NewBuffer(jsonData))
				if err != nil {
					fmt.Println("Error while creating the order")
					return
				}
				configData, err := config.ReadConfig()
				if err != nil {
					fmt.Println("Error while reading a the order")
					return
				}
				// Add the Bearer token to the Authorization header
				req.Header.Set("Authorization", "Bearer "+configData.AccessToken)
				req.Header.Set("Content-Type", "application/json")
				response, err := utils.GetHttpResponse(req)
				if err != nil {
					fmt.Println("Error while creating the order")
					return
				}
				if response.Success && response.Error == "" {
					if _, ok := response.Data.(map[string]interface{}); ok {
						fmt.Println("Creating an order has succeeded")
						fmt.Println("Order created in waitingForDeposit state!")
						fmt.Printf("You now have %d min to deposit funds.\n", orderCommonInfo.DepositTimeout)
						fmt.Printf("Please send  ## XEL from your wallet %s to the tokenswap wallet %s\n",
							orderReq.OrdererWalletAddress, walletAddress)
						// TODO: SHOW QRCODE of tokenswap wallet"
					} else {
						fmt.Println("Error while creating the order")
						return
					}
				} else {
					fmt.Println("Error while creating the order")
					return
				}
			} else {
				fmt.Println("Not confirmed to create an order")
				return
			}
		},
	}
)

func checkOrderOptions(reqOrder *OrderRequest, cmd *cobra.Command) error {
	chainVal, _ := cmd.Flags().GetString(flagChain)
	if chainVal != "" {
		for _, chain := range orderCommonInfo.Chains {
			if chainVal == chain {
				reqOrder.Order.Chain = chainVal
				break
			}
		}
		if reqOrder.Order.Chain == "" {
			return fmt.Errorf("not a valid chain name")
		}
	}
	network, _ := cmd.Flags().GetString(flagNetwork)
	if network != "" {
		if _, ok := orderCommonInfo.Networks[network]; ok {
			reqOrder.Order.Network = network
		} else {
			return fmt.Errorf("not a valid network name")
		}
	}
	private, _ := cmd.Flags().GetBool(flagTypeValuePrivate)
	if private {
		reqOrder.Order.Visibility = flagTypeValuePrivate
	} else {
		reqOrder.Order.Visibility = "public"
	}
	referral, _ := cmd.Flags().GetString(flagReferral) // TODO: define referral code management
	if referral != "" {
		reqOrder.Order.Referral = referral
	}
	return nil
}

var listPairs bool

func init() {
	OrderCmd.AddCommand(orderCreateCmd)

	orderCreateCmd.Flags().String(flagChain, "polygon", "Set blockchain")
	orderCreateCmd.Flags().String(flagNetwork, flagTypeValueMainnet, "Set network")
	orderCreateCmd.Flags().BoolVar(&listPairs, "list-pairs", false, "List supported pairs and chain")
	orderCreateCmd.Flags().Bool(flagTypeValuePrivate, false, "Set the creating a private order") // can be another name for boolean?
	orderCreateCmd.Flags().String(flagReferral, "", "Set a referral code")
}
