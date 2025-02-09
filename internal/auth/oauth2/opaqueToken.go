package oauth2

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
)

type Token struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    uint64 `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
}

type TokenRequest struct {
	AuthCodeResponse *AuthCodeResponse
	Client           *Client
	TokenUrl         *url.URL
}

func RequestToken(tokenRequest *TokenRequest) (*Token, error) {
	authCode := tokenRequest.AuthCodeResponse.AuthCode
	client := tokenRequest.Client
	Url := tokenRequest.TokenUrl
	log.Printf("AuthCode: %s", authCode)
	query := url.Values{
		"redirect_uri": []string{client.Redirect_uri},
		"grant_type":   []string{"authorization_code"},
		"code":         []string{authCode},
		"client_id":    []string{client.Client_id},
	}
	Url.RawQuery = query.Encode()
	log.Printf("Token-URL: %s", Url.String())
	req, reqErr := http.NewRequest("POST", Url.String(), nil)
	if reqErr != nil {
		log.Printf("ERROR: Could not construct token request: %v", reqErr)
		return nil, reqErr
	}
	httpClient := &http.Client{}
	res, err := httpClient.Do(req)
	if err != nil {
		log.Printf("ERROR obtaining token: %v", err)
		return nil, err
	}
	data, bodyErr := ioutil.ReadAll(res.Body)
	if bodyErr != nil {
		log.Printf("ERROR: Could not resd token response: %v", bodyErr)
		return nil, bodyErr
	}
	log.Printf("Token response: %s", string(data))
	jToken := &Token{}
	errJson := json.Unmarshal(data, jToken)
	if errJson != nil {
		log.Printf("ERROR: JSON unmarshalling error: %v", errJson)
		return nil, errJson
	}
	return jToken, nil
}
