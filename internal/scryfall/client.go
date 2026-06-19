package scryfall

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

var baseURL = "https://api.scryfall.com"

// Client is a minimal Scryfall API client.
type Client struct {
	http    *http.Client
	limiter <-chan struct{}
}

func New() *Client {
	ch := make(chan struct{})
	go func() {
		t := time.NewTicker(100 * time.Millisecond)
		for range t.C {
			ch <- struct{}{}
		}
	}()
	return &Client{
		http:    &http.Client{Timeout: 10 * time.Second},
		limiter: ch,
	}
}

// Prices holds Scryfall market prices. Values are decimal strings (e.g. "0.15") or empty if unavailable.
type Prices struct {
	USD     string `json:"usd"`
	USDFoil string `json:"usd_foil"`
	EUR     string `json:"eur"`
	EURFoil string `json:"eur_foil"`
	TIX     string `json:"tix"`
}

// CardInfo holds the data fetched from Scryfall for a single printing.
type CardInfo struct {
	ImageURL string
	Prices   Prices
}

type cardResponse struct {
	ImageURIs *imageURIs `json:"image_uris"`
	CardFaces []struct {
		ImageURIs *imageURIs `json:"image_uris"`
	} `json:"card_faces"`
	Prices Prices `json:"prices"`
}

type imageURIs struct {
	Normal string `json:"normal"`
}

// GetCardInfo returns the image URL and prices for a card by set code and collector number.
// Double-faced cards fall back to the front face image.
func (c *Client) GetCardInfo(ctx context.Context, set, number string) (*CardInfo, error) {
	<-c.limiter

	url := fmt.Sprintf("%s/cards/%s/%s", baseURL, strings.ToLower(set), number) //nolint:gocritic
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("scryfall %s/%s: status %d", set, number, resp.StatusCode)
	}

	var card cardResponse
	if err := json.NewDecoder(resp.Body).Decode(&card); err != nil {
		return nil, err
	}

	var imageURL string
	if card.ImageURIs != nil {
		imageURL = card.ImageURIs.Normal
	} else if len(card.CardFaces) > 0 && card.CardFaces[0].ImageURIs != nil {
		// double-faced card — use front face
		imageURL = card.CardFaces[0].ImageURIs.Normal
	} else {
		return nil, fmt.Errorf("no image found for %s/%s", set, number)
	}

	return &CardInfo{
		ImageURL: imageURL,
		Prices:   card.Prices,
	}, nil
}
