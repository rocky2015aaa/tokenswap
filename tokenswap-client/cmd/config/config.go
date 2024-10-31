package config

import (
	"fmt"
	"net/http"
	"time"

	"github.com/rocky2015aaa/tokenswap-client/config"
	"github.com/rocky2015aaa/tokenswap-client/utils"
	"github.com/spf13/cobra"
)

var (
	configGetCmd = &cobra.Command{
		Use:   "get",
		Short: "List the application configuration",
		Long:  `List the application configuration`,
		Run: func(cmd *cobra.Command, args []string) {
			configData, err := config.ReadConfig()
			if err != nil {
				fmt.Println("Error while reading a config file")
				return
			}
			req, err := http.NewRequest("GET", config.tokenswapServerUrl+"/user", nil)
			if err != nil {
				fmt.Println("Error while getting the user information")
				return
			}
			// Add the Bearer token to the Authorization header
			req.Header.Set("Authorization", "Bearer "+configData.AccessToken)
			response, err := utils.GetHttpResponse(req)
			if err != nil {
				fmt.Println("Error while getting the user information")
				return
			}
			if response.Success && response.Error == "" {
				data, ok := response.Data.(map[string]interface{})
				if !ok {
					fmt.Println("Error while printing the user information")
					return
				}
				err = parseAndPrintUserInfo(data)
				if err != nil {
					fmt.Println("Error while printing the user information")
					return
				}
			} else {
				fmt.Println("Error while getting the user information")
				return
			}
		},
	}

	ConfigSetCmd = &cobra.Command{
		Use:   "set",
		Short: "Set the application configuration",
		Long:  `Set the application configuration`,
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}

	ConfigCmd = &cobra.Command{
		Use:   "config",
		Short: "The command for the application configuration",
		Long:  `The command for the application configuration. List configuration setting. Update configuration setting`,
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}
)

func init() {
	ConfigCmd.AddCommand(configGetCmd)
	ConfigCmd.AddCommand(ConfigSetCmd)
}

func parseAndPrintUserInfo(data map[string]interface{}) error {
	tokenExpirationDateTimeStr, ok := data["token_expiration_date_time"].(string)
	if !ok {
		return fmt.Errorf("missing or invalid 'token_expiration_date_time' in config data")
	}
	registrationDateTimeStr, ok := data["registration_date_time"].(string)
	if !ok {
		return fmt.Errorf("missing or invalid 'registration_date_time' in config data")
	}
	accessTokenExpirationDateTime, err := parseTime(tokenExpirationDateTimeStr, config.TimeFormat)
	if err != nil {
		return fmt.Errorf("error parsing access token expiration date time: %w", err)
	}
	registrationDateTime, err := parseTime(registrationDateTimeStr, config.TimeFormat)
	if err != nil {
		return fmt.Errorf("error parsing registration date time: %w", err)
	}

	localLocation, err := time.LoadLocation("Local")
	if err != nil {
		return fmt.Errorf("error loading timezone location: %s", err)
	}

	// Convert UTC time to local time
	registrationDateTimeInLocalTimeStr := registrationDateTime.In(localLocation).Format(config.TimeFormat)
	sessionLifeTime := time.Until(accessTokenExpirationDateTime)
	uuid, ok := data["uuid"].(string)
	if !ok {
		return fmt.Errorf("missing or invalid 'uuid' in config data")
	}
	fmt.Println("----[Your Setting]-----")
	fmt.Println("UUID:", uuid)
	fmt.Printf("Session Lifetime: %s\n", sessionLifeTime.String())
	fmt.Println("----[Info]-----")
	fmt.Println("Registration Datetime:", registrationDateTimeInLocalTimeStr)
	fmt.Println("This session is still valid for:", sessionLifeTime)

	return nil
}

func parseTime(timeStr string, format string) (time.Time, error) {
	return time.Parse(format, timeStr)
}
