package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"

	"github.com/rocky2015aaa/tokenswap-client/utils"
)

const (
	tokenExpirationMessage = "Token is expired"
)

func ManageUserTokens() error {
	_, err := os.Stat(AbsoluteConfigFilePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("Please run the client application with --init flag for the initialization")
		} else {
			return fmt.Errorf("Error while checking a config file")
		}
	}
	configData, err := ReadConfig()
	if err != nil {
		return fmt.Errorf("Error while reading a config file")
	}
	req, err := http.NewRequest("GET", tokenswapServerUrl+"/auth/ping", nil)
	if err != nil {
		return fmt.Errorf("Error while checking access token validity")
	}
	// Add the Bearer token to the Authorization header
	req.Header.Set("Authorization", "Bearer "+configData.AccessToken)
	response, err := utils.GetHttpResponse(req)
	if err != nil {
		return fmt.Errorf("Error while checking access token validity")
	}
	// If access token has expired, proceed refresh token process
	if response.Error == tokenExpirationMessage {
		fmt.Println("The access token is expired. Please enter your password to continue.")
		userPassword, err := utils.InputPassword("Enter password: ")
		if err != nil {
			return fmt.Errorf("Error while getting the user password")
		}
		req := ConfigRequest{
			Email:    configData.Email,
			Password: userPassword,
		}
		response, err := refreshToken(&req, configData)
		if err != nil {
			return fmt.Errorf("Error while updating the access token")
		}
		if response.Error == "" {
			if data, ok := response.Data.(map[string]interface{}); ok {
				accessToken, ok := data["access_token"].(string)
				if !ok {
					return fmt.Errorf("Error while updating the access token")
				}
				configData.AccessToken = accessToken
				err := UpdateConfig(configData)
				if err != nil {
					return fmt.Errorf("Error while updating the config file")
				}
				fmt.Println("The access token has been updated")
			} else {
				return fmt.Errorf("Error while updating the access token")
			}
			// If the refresh token also has expired, proceed renew tokens process
		} else if response.Error == tokenExpirationMessage {
			fmt.Println("The refresh token is also expired. The access token and the refresh token will be updated.")
			response, err = renewTokens(&req)
			if err != nil {
				return fmt.Errorf("Error while updating the access token and the refresh token")
			}
			if response.Error == "" {
				if data, ok := response.Data.(map[string]interface{}); ok {
					accessToken, ok := data["access_token"].(string)
					if !ok {
						return fmt.Errorf("Error while updating the access token and the refresh token")
					}
					refreshToken, ok := data["refresh_token"].(string)
					if !ok {
						return fmt.Errorf("Error while updating the access token and the refresh token")
					}
					configData.AccessToken = accessToken
					configData.RefreshToken = refreshToken
					err := UpdateConfig(configData)
					if err != nil {
						return fmt.Errorf("Error while updating the config file")
					}
					fmt.Println("The access token and the refresh token have been updated")
				} else {
					return fmt.Errorf("Error while updating the access token and the refresh token")
				}
			} else {
				return fmt.Errorf("Error while updating the access token and the refresh token")
			}
		} else {
			return fmt.Errorf("Error while updating the access token and the refresh token")
		}
	}
	return nil
}

func refreshToken(configReq *ConfigRequest, configData *Config) (*utils.Response, error) {
	data := RefreshRequest{
		ConfigRequest: configReq,
		RefreshToken:  configData.RefreshToken,
	}
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("error while marshaling a refresh token data: %s", err)
	}
	req, err := http.NewRequest("POST", tokenswapServerUrl+"/token/refresh", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("error while creating a refresh token request: %s", err)
	}
	req.Header.Set("Content-Type", "application/json")
	response, err := utils.GetHttpResponse(req)
	if err != nil {
		return nil, fmt.Errorf("error while getting a http response: %s", err)
	}
	return response, nil
}

func renewTokens(configReq *ConfigRequest) (*utils.Response, error) {
	jsonData, err := json.Marshal(configReq)
	if err != nil {
		return nil, fmt.Errorf("error marshaling a renew tokens request data: %s", err)
	}
	req, err := http.NewRequest("POST", tokenswapServerUrl+"/token/renew", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("error creating a renew tokens request: %s", err)
	}
	req.Header.Set("Content-Type", "application/json")
	response, err := utils.GetHttpResponse(req)
	if err != nil {
		return nil, fmt.Errorf("error getting a http response: %s", err)
	}
	return response, nil
}
