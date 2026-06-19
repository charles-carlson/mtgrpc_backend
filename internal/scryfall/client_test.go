package scryfall

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newTestClient(srv *httptest.Server) *Client {
	return &Client{
		http:    srv.Client(),
		limiter: alwaysReady(),
	}
}

// alwaysReady returns a channel that always has a value ready so tests don't wait on the rate limiter.
func alwaysReady() <-chan struct{} {
	ch := make(chan struct{}, 1)
	ch <- struct{}{}
	go func() {
		for {
			ch <- struct{}{}
		}
	}()
	return ch
}

func TestGetCardInfo_Normal(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"image_uris": map[string]string{
				"normal": "https://example.com/card.jpg",
			},
			"prices": map[string]string{
				"usd": "0.15",
				"eur": "0.10",
			},
		})
	}))
	defer srv.Close()

	c := newTestClient(srv)
	origBase := baseURL
	baseURL = srv.URL
	defer func() { baseURL = origBase }()

	info, err := c.GetCardInfo(context.Background(), "M10", "149")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.ImageURL != "https://example.com/card.jpg" {
		t.Errorf("got image %q, want %q", info.ImageURL, "https://example.com/card.jpg")
	}
	if info.Prices.USD != "0.15" {
		t.Errorf("got usd %q, want %q", info.Prices.USD, "0.15")
	}
	if info.Prices.EUR != "0.10" {
		t.Errorf("got eur %q, want %q", info.Prices.EUR, "0.10")
	}
}

func TestGetCardInfo_DoubleFaced(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"card_faces": []map[string]any{
				{"image_uris": map[string]string{"normal": "https://example.com/front.jpg"}},
				{"image_uris": map[string]string{"normal": "https://example.com/back.jpg"}},
			},
			"prices": map[string]string{
				"usd": "5.00",
			},
		})
	}))
	defer srv.Close()

	c := newTestClient(srv)
	origBase := baseURL
	baseURL = srv.URL
	defer func() { baseURL = origBase }()

	info, err := c.GetCardInfo(context.Background(), "MID", "98")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.ImageURL != "https://example.com/front.jpg" {
		t.Errorf("got %q, want front face URL", info.ImageURL)
	}
}

func TestGetCardInfo_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c := newTestClient(srv)
	origBase := baseURL
	baseURL = srv.URL
	defer func() { baseURL = origBase }()

	_, err := c.GetCardInfo(context.Background(), "BAD", "999")
	if err == nil {
		t.Fatal("expected error for 404, got nil")
	}
}
