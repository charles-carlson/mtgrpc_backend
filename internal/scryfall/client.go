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
	USD       string `json:"usd"`
	USDFoil   string `json:"usd_foil"`
	EUR       string `json:"eur"`
	EURFoil   string `json:"eur_foil"`
	TIX       string `json:"tix"`
	USDEtched string `json:"usd_etched"`
	EUREtched string `json:"eur_etched"`
}

// CardInfo holds the data fetched from Scryfall for a single printing.
type CardInfo struct {
	ImageURL string
	Prices   Prices
	Colors   []string
	Rarity   string
	TypeLine string
}
type SetList struct {
	Data     []SetInfo `json:"data"`
	HasMore  bool      `json:"has_more"`  // only meaningful on /cards/search
	NextPage string    `json:"next_page"` // ditto
}
type SetInfo struct {
	Name        string `json:"name"`
	Code        string `json:"code"`
	CardCount   int    `json:"card_count"`
	PrintedSize *int   `json:"printed_size"`
	IconSVGUri  string `json:"icon_svg_uri"`
	Digital     bool   `json:"digital"`
	SetType     string `json:"set_type"`
	NonfoilOnly bool   `json:"nonfoil_only"`
	FoilOnly    bool   `json:"foil_only"`
}
type cardResponse struct {
	ImageURIs *imageURIs `json:"image_uris"`
	CardFaces []struct {
		ImageURIs *imageURIs `json:"image_uris"`
	} `json:"card_faces"`
	Prices   Prices   `json:"prices"`
	Colors   []string `json:"colors"`
	Rarity   string   `json:"rarity"`
	TypeLine string   `json:"type_line"`
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
		Colors:   card.Colors,
		Rarity:   card.Rarity,
		TypeLine: card.TypeLine,
	}, nil
}
func (c *Client) GetSetsInfo(ctx context.Context) (*map[string]SetInfo, error) {
	<-c.limiter
	url := fmt.Sprintf("%s/sets", baseURL)
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
		return nil, fmt.Errorf("scryfall sets: status %d", resp.StatusCode)
	}
	var setList SetList
	if err := json.NewDecoder(resp.Body).Decode(&setList); err != nil {
		return nil, err
	}
	m := make(map[string]SetInfo, len(setList.Data))
	for _, s := range setList.Data {
		m[s.Code] = s
	}
	return &m, nil
}
