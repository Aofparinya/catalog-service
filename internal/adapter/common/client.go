package common

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"
)

type Client struct {
	baseURL, authURL, clientID, clientSecret string
	http                                     *http.Client
	mutex                                    sync.Mutex
	token                                    string
	expires                                  time.Time
}

func New(baseURL, authURL, clientID, clientSecret string) *Client {
	return &Client{baseURL: strings.TrimRight(baseURL, "/"), authURL: strings.TrimRight(authURL, "/"), clientID: clientID, clientSecret: clientSecret, http: &http.Client{Timeout: 5 * time.Second}}
}
func (c *Client) NextNumber(ctx context.Context, kind string) (string, error) {
	token, err := c.getToken(ctx)
	if err != nil {
		return "", err
	}
	body, _ := json.Marshal(map[string]string{"documentType": kind})
	req, _ := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/v1/document-numbers/next", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	res, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return "", errors.New("common service number request failed")
	}
	var out struct {
		Number string `json:"number"`
	}
	if json.NewDecoder(res.Body).Decode(&out) != nil || out.Number == "" {
		return "", errors.New("invalid common service response")
	}
	return out.Number, nil
}
func (c *Client) Token(ctx context.Context) (string, error) { return c.getToken(ctx) }
func (c *Client) getToken(ctx context.Context) (string, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if c.token != "" && time.Now().Before(c.expires.Add(-20*time.Second)) {
		return c.token, nil
	}
	body, _ := json.Marshal(map[string]string{"clientId": c.clientID, "clientSecret": c.clientSecret})
	req, _ := http.NewRequestWithContext(ctx, "POST", c.authURL+"/api/v1/auth/service-token", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	res, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return "", errors.New("service token request failed")
	}
	var out struct {
		AccessToken string `json:"accessToken"`
		ExpiresIn   int    `json:"expiresIn"`
	}
	if json.NewDecoder(res.Body).Decode(&out) != nil {
		return "", errors.New("invalid service token response")
	}
	c.token = out.AccessToken
	c.expires = time.Now().Add(time.Duration(out.ExpiresIn) * time.Second)
	return c.token, nil
}
