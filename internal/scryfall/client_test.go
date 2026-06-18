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

func TestGetImageURL_Normal(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"image_uris": map[string]string{
				"normal": "https://example.com/card.jpg",
			},
		})
	}))
	defer srv.Close()

	c := newTestClient(srv)
	// override baseURL for test
	origBase := baseURL
	baseURL = srv.URL
	defer func() { baseURL = origBase }()

	url, err := c.GetImageURL(context.Background(), "M10", "149")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if url != "https://example.com/card.jpg" {
		t.Errorf("got %q, want %q", url, "https://example.com/card.jpg")
	}
}

func TestGetImageURL_DoubleFaced(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"card_faces": []map[string]any{
				{"image_uris": map[string]string{"normal": "https://example.com/front.jpg"}},
				{"image_uris": map[string]string{"normal": "https://example.com/back.jpg"}},
			},
		})
	}))
	defer srv.Close()

	c := newTestClient(srv)
	origBase := baseURL
	baseURL = srv.URL
	defer func() { baseURL = origBase }()

	url, err := c.GetImageURL(context.Background(), "MID", "98")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if url != "https://example.com/front.jpg" {
		t.Errorf("got %q, want front face URL", url)
	}
}

func TestGetImageURL_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c := newTestClient(srv)
	origBase := baseURL
	baseURL = srv.URL
	defer func() { baseURL = origBase }()

	_, err := c.GetImageURL(context.Background(), "BAD", "999")
	if err == nil {
		t.Fatal("expected error for 404, got nil")
	}
}
