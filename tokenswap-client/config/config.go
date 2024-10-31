package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/user"
	"path/filepath"

	"github.com/rocky2015aaa/tokenswap-client/utils"
)

var (
	// Construct the absolute file path
	AbsoluteConfigFilePath string
	tokenswapServerUrl      string
)

const (
	ConfigFilePath = ".tokenswap/config"
	TimeFormat     = "2006-01-02 15:04:05 MST"
	serverUrlEnv   = "tokenswap_SERVER_URL"

	UserExistanceMessage = "The user already exists"
)

func init() {
	// Write encrypted config to file
	usr, err := user.Current()
	if err != nil {
		fmt.Println("Error while getting current user location:", err)
		os.Exit(1)
	}

	// Construct the absolute file path
	AbsoluteConfigFilePath = filepath.Join(usr.HomeDir, ConfigFilePath)
	tokenswapServerUrl = os.Getenv(serverUrlEnv)
	if tokenswapServerUrl == "" {
		fmt.Printf("Error while getting swapter server url. no env %s value\n", serverUrlEnv)
		os.Exit(1)
	}
}

func CreateConfig() error {
	email, err := utils.InputUserEmail()
	if err != nil {
		return fmt.Errorf("error while entering user email. %s", err)
	}
	confirmed, err := utils.ConfirmPasswordWarning(utils.ConfirmationSentence)
	if err != nil {
		return fmt.Errorf("error while entering confirm password warning. %s", err)
	}
	if !confirmed {
		return fmt.Errorf("confirm password warning input is wrong")
	}
	userPassword, err := utils.GetPassword()
	if err != nil {
		return fmt.Errorf("error while entering password. %s", err)
	}

	data := ConfigRequest{
		Email:    email,
		Password: userPassword,
	}
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("error while marshaling register data. %s", err)
	}
	req, err := http.NewRequest("POST", tokenswapServerUrl+"/user/register", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("error while creating register request. %s", err)
	}
	req.Header.Set("Content-Type", "application/json")
	response, err := utils.GetHttpResponse(req)
	if err != nil {
		return fmt.Errorf("error while getting a http response. %s", err)
	}
	if data, ok := response.Data.(map[string]interface{}); ok {
		return createConfigFile(email, data)
	} else {
		if response.Error == UserExistanceMessage {
			return fmt.Errorf(UserExistanceMessage)
		}
		return fmt.Errorf("error while getting response data")
	}
}

func createConfigFile(email string, data map[string]interface{}) error {
	// Extract access and refresh tokens
	accessToken, ok := data["access_token"].(string)
	if !ok {
		return fmt.Errorf("missing or invalid 'access_token' in config data")
	}
	refreshToken, ok := data["refresh_token"].(string)
	if !ok {
		return fmt.Errorf("missing or invalid 'refresh_token' in config data")
	}
	config := &Config{
		Email:        email,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}
	configData, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("error while marshaling %+v config data. %s", config, err)
	}
	err = utils.CreateFile(configData, AbsoluteConfigFilePath, 0755)
	if err != nil {
		return fmt.Errorf("failed to create a config file. %s", err)
	}
	return nil
}

func ReadConfig() (*Config, error) {
	config := Config{}
	err := utils.ReadFile(AbsoluteConfigFilePath, &config)
	if err != nil {
		return nil, fmt.Errorf("error while reading config JSON. %s", err)
	}

	return &config, nil
}

func UpdateConfig(config *Config) error {
	err := utils.UpdateFile(config, AbsoluteConfigFilePath)
	if err != nil {
		return fmt.Errorf("error while updating a config JSON data. %s", err)
	}
	return nil
}
