package auth

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
)

type Client struct {
	Client_id    string
	Redirect_uri string
}

func RequestAuthCode(URL *url.URL, user *url.Userinfo, client Client, code_challenge string) (*http.Response, error, int) {
	query := url.Values{
		"response_type":  []string{"code"},
		"client_id":      []string{client.Client_id},
		"code_challenge": []string{code_challenge},
		"redirect_uri":   []string{client.Redirect_uri},
	}
	URL.RawQuery = query.Encode()
	req, err := http.NewRequest("GET", URL.String(), nil)
	if err != nil {
		log.Printf("ERROR: Login Request: %v", err)
		return nil, err, http.StatusInternalServerError
	}
	if password, ok := user.Password(); ok {
		req.SetBasicAuth(user.Username(), password)
	} else {
		return nil, fmt.Errorf("No password set for user %s", user.Username()), http.StatusUnauthorized
	}
	httpClient := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	res, resErr := httpClient.Do(req)
	if resErr != nil {
		log.Printf("ERROR: Login upstream: %v", resErr)
		if res != nil {
			return nil, resErr, res.StatusCode
		} else {
			return nil, resErr, http.StatusInternalServerError
		}
	}
	if data, err := ioutil.ReadAll(res.Body); err != nil {
		log.Printf("ERROR: Login response body: %v", err)
		return nil, err, res.StatusCode
	} else {
		log.Printf("StatusCode: %v", res.StatusCode)
		for k, v := range res.Header {
			log.Printf("%s: %s", k, v)
		}
		loc, locErr := res.Location()
		if locErr != nil {
			log.Printf("Error with response location: %v", locErr)
			return res, locErr, res.StatusCode
		}
		log.Printf("Location: %s", loc)
		log.Printf("data: %v", string(data))
	}
	return res, nil, res.StatusCode
}
