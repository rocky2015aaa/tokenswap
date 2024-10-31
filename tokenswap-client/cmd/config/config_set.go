package config

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/rocky2015aaa/tokenswap-client/config"
	"github.com/rocky2015aaa/tokenswap-client/utils"
)

const (
	minimumExpirationTime = 60
)

var (
	configSetAccesstokenExpirationSetCmd = &cobra.Command{
		Use:   "expiration",
		Short: "Set the access token expiration time",
		Long:  `Set the access token expiration time`,
		Run: func(cmd *cobra.Command, args []string) {
			userPassword, err := utils.InputPassword("Enter password: ")
			if err != nil {
				fmt.Println("Error while getting the user password")
				return
			}
			fmt.Printf("Enter new expiration time in seconds for the access token (must be more than %d seconds): ", minimumExpirationTime)
			reader := bufio.NewReader(os.Stdin)
			newExpirationTime, err := reader.ReadString('\n')
			if err != nil {
				fmt.Println("Error while getting the new access token expiration time")
				return
			}
			newExpirationTime = strings.TrimSpace(newExpirationTime)
			newExpirationTimeInSeconds, err := strconv.Atoi(newExpirationTime)
			if err != nil {
				fmt.Printf("The new access token expiration time is not a number: %s", newExpirationTime)
				return
			}
			if newExpirationTimeInSeconds <= minimumExpirationTime {
				fmt.Printf("Invalid new expiration time input. Please enter a number greater than %d seconds.\n", minimumExpirationTime)
				return
			}
			data := config.UpdateTokenExpirationRequest{
				ConfigRequest: &config.ConfigRequest{
					Password: userPassword,
				},
				AccessTokenExpirationTimeInSeconds:  newExpirationTimeInSeconds,
				RefreshTokenExpirationTimeInSeconds: newExpirationTimeInSeconds * 2, // for refresh token, twice of access token expiration time. can be updated later
			}
			jsonData, err := json.Marshal(data)
			if err != nil {
				fmt.Println("Error while updating the new access token expiration time")
				return
			}
			req, err := http.NewRequest("POST", config.tokenswapServerUrl+"/token/renew-exp", bytes.NewBuffer(jsonData))
			if err != nil {
				fmt.Println("Error while updating the new access token expiration time")
				return
			}
			configData, err := config.ReadConfig()
			if err != nil {
				fmt.Println("Error while reading a config file")
				return
			}
			// Add the Bearer token to the Authorization header
			req.Header.Set("Authorization", "Bearer "+configData.AccessToken)
			req.Header.Set("Content-Type", "application/json")
			response, err := utils.GetHttpResponse(req)
			if err != nil {
				fmt.Println("Error while updating the new access token expiration")
				return
			}
			if response.Success && response.Error == "" {
				if data, ok := response.Data.(map[string]interface{}); ok {
					accessToken, ok := data["access_token"].(string)
					if !ok {
						fmt.Println("Error while updating a config file with the new access token expiration")
						return
					}
					refreshToken, ok := data["refresh_token"].(string)
					if !ok {
						fmt.Println("Error while updating a config file with the new access token expiration")
						return
					}
					configData.AccessToken = accessToken
					configData.RefreshToken = refreshToken
					err := config.UpdateConfig(configData)
					if err != nil {
						fmt.Println("Error while updating a config file with the new access token expiration")
						return
					}
				} else {
					fmt.Println("Error while updating the new access token expiration")
					return
				}
			} else {
				fmt.Println("Error while updating the new access token expiration")
				return
			}
			fmt.Println("The access token and the refresh token have been updated with new expiration time")
		},
	}

	configPasswordSetCmd = &cobra.Command{
		Use:   "password",
		Short: "Set the application password",
		Long:  `Set the application password`,
		Run: func(cmd *cobra.Command, args []string) {
			currentUserPassword, err := utils.InputPassword("Enter password: ")
			if err != nil {
				fmt.Println("Error while getting the user password")
				return
			}
			confirmed, err := utils.ConfirmPasswordWarning(utils.ConfirmationSentence)
			if err != nil {
				fmt.Println("Error while confirming a password warning")
				return
			}
			if !confirmed {
				fmt.Println("Confirm password warning input is wrong. Please try again.")
				return
			}
			newUserPassword, err := utils.GetPassword()
			if err != nil {
				fmt.Println("Error while getting the new user password")
				return
			}
			data := config.PasswordUpdateRequest{
				ConfigRequest: &config.ConfigRequest{
					Password: currentUserPassword,
				},
				NewPassword: newUserPassword,
			}
			jsonData, err := json.Marshal(data)
			if err != nil {
				fmt.Println("Error while updating the new user password")
				return
			}
			req, err := http.NewRequest("PATCH", config.tokenswapServerUrl+"/user/update-password", bytes.NewBuffer(jsonData))
			if err != nil {
				fmt.Println("Error while updating the new user password")
				return
			}
			// Add the Bearer token to the Authorization header
			configData, err := config.ReadConfig()
			if err != nil {
				fmt.Println("Error while reading a config file")
				return
			}
			req.Header.Set("Authorization", "Bearer "+configData.AccessToken)
			req.Header.Set("Content-Type", "application/json")
			response, err := utils.GetHttpResponse(req)
			if err != nil {
				fmt.Println("Error while updating the new user password")
				return
			}
			if response.Success && response.Error == "" {
				fmt.Println("The new user password has been updated")
			} else {
				fmt.Println("Error while updating the new user password")
			}
		},
	}
)

func init() {
	ConfigSetCmd.AddCommand(configSetAccesstokenExpirationSetCmd)
	ConfigSetCmd.AddCommand(configPasswordSetCmd)
}
