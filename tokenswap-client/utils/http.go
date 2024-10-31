package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const (
	NoDocumentMessage = "mongo: no documents in result"
)

type Response struct {
	Data        interface{} `json:"data"`
	Description string      `json:"description"`
	Error       string      `json:"error"`
	Success     bool        `json:"success"`
}

func GetHttpResponse(req *http.Request) (*Response, error) {
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var response Response
	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling response: %w", err)
	}

	return &response, nil
}
