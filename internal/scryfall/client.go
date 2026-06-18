package scryfall

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const baseURL = "https://api.scryfall.com"

// Client is a minimal Scryfall API client.
type Client struct {
	http    *http.Client
	limiter <-chan time.Time
}

func New() *Client {
	return &Client{
		http:    &http.Client{Timeout: 10 * time.Second},
		limiter: time.Tick(100 * time.Millisecond), // stay well under 10 req/s
	}
}

type cardResponse struct {
	ImageURIs *imageURIs `json:"image_uris"`
	CardFaces []struct {
		ImageURIs *imageURIs `json:"image_uris"`
	} `json:"card_faces"`
}

type imageURIs struct {
	Normal string `json:"normal"`
}

// GetImageURL returns the normal image URL for a card by set code and collector number.
// Double-faced cards fall back to the front face image.
func (c *Client) GetImageURL(ctx context.Context, set, number string) (string, error) {
	<-c.limiter

	url := fmt.Sprintf("%s/cards/%s/%s", baseURL, strings.ToLower(set), number)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("scryfall %s/%s: status %d", set, number, resp.StatusCode)
	}

	var card cardResponse
	if err := json.NewDecoder(resp.Body).Decode(&card); err != nil {
		return "", err
	}

	if card.ImageURIs != nil {
		return card.ImageURIs.Normal, nil
	}
	// double-faced card — use front face
	if len(card.CardFaces) > 0 && card.CardFaces[0].ImageURIs != nil {
		return card.CardFaces[0].ImageURIs.Normal, nil
	}

	return "", fmt.Errorf("no image found for %s/%s", set, number)
}
