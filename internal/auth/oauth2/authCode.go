package oauth2

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
)

type AuthCodeRequest struct {
	AuthUrl *url.URL
	Client  *Client
	Scope   string
}

type AuthCodeResponse struct {
	AuthCode   string
	Location   *url.URL
	StatusCode int
}

func extractAuthCode(location *url.URL) (string, error) {
	var authCode string
	var authCodeErr error
	if authCodes, ok := location.Query()["code"]; ok {
		if len(authCodes) == 1 { // only one code query parameter accepted
			authCode = authCodes[0]
		} else {
			authCodeErr = fmt.Errorf("Not exactly 1 code parameter found in location parameter: %s", location.String())
		}
	} else {
		authCodeErr = fmt.Errorf("Location %s contains no code query parameter", location)
	}
	return authCode, authCodeErr
}

func RequestAuthCode(authRequest *AuthCodeRequest, user *url.Userinfo, state string, code_challenge string) (*AuthCodeResponse, error) {
	client := authRequest.Client
	scope := authRequest.Scope
	URL := authRequest.AuthUrl
	query := url.Values{
		"response_type":  []string{"code"},
		"client_id":      []string{client.Client_id},
		"code_challenge": []string{code_challenge},
		"redirect_uri":   []string{client.Redirect_uri},
		"scope":          []string{scope},
		"state":          []string{state},
	}
	URL.RawQuery = query.Encode()
	req, err := http.NewRequest("GET", URL.String(), nil)
	if err != nil {
		log.Printf("ERROR: Login Request: %v", err)
		return &AuthCodeResponse{StatusCode: http.StatusInternalServerError}, err
	}
	if password, ok := user.Password(); ok {
		req.SetBasicAuth(user.Username(), password)
	} else {
		return &AuthCodeResponse{StatusCode: http.StatusUnauthorized}, fmt.Errorf("No password set for user %s", user.Username())
	}
	httpClient := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	res, resErr := httpClient.Do(req)
	if resErr != nil {
		log.Printf("ERROR: Login upstream: %v Response: %v", resErr, res)
		if res != nil {
			return &AuthCodeResponse{StatusCode: res.StatusCode}, resErr
		} else {
			return &AuthCodeResponse{StatusCode: http.StatusUnauthorized}, resErr
		}
	}
	log.Printf("StatusCode: %v", res.StatusCode)
	loc, locErr := res.Location()
	if locErr != nil {
		log.Printf("Error with response location: %v Response: %v", locErr, res)
		return &AuthCodeResponse{StatusCode: http.StatusInternalServerError}, locErr
	}
	log.Printf("Location: %s", loc)
	authCode, authCodeErr := extractAuthCode(loc)
	if authCodeErr != nil {
		log.Printf("ERROR: %v", err)
		return &AuthCodeResponse{StatusCode: http.StatusInternalServerError}, err
	}
	return &AuthCodeResponse{
		AuthCode:   authCode,
		Location:   loc,
		StatusCode: http.StatusOK,
	}, nil
}
