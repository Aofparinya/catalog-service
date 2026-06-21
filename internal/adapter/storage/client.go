package storage

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	commonclient "github.com/saaof/order-platform/catalog-service/internal/adapter/common"
	"net/http"
	"strings"
)

type Client struct {
	baseURL string
	numbers *commonclient.Client
}

func New(baseURL string, tokenProvider *commonclient.Client) *Client {
	return &Client{baseURL: strings.TrimRight(baseURL, "/"), numbers: tokenProvider}
}
func (c *Client) ValidateImage(ctx context.Context, id uuid.UUID) error {
	token, err := c.numbers.Token(ctx)
	if err != nil {
		return err
	}
	req, _ := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/api/v1/files/"+id.String(), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return errors.New("storage file not found")
	}
	var out struct {
		MimeType string `json:"mimeType"`
	}
	if json.NewDecoder(res.Body).Decode(&out) != nil || !strings.HasPrefix(out.MimeType, "image/") {
		return errors.New("file must be an image")
	}
	return nil
}
