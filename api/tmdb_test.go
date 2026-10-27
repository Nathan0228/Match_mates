package main

import (
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestTMDBSearchAndMediaPosterValidation(t *testing.T) {
	a := paymentTestApp(t)
	if got := paymentCall(a.searchMedia, "GET", "/api/media/search?kind=movies&q=海街日记", "alice", "", ""); got.Code != 503 {
		t.Fatalf("missing TMDB credentials: %d", got.Code)
	}
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.Header.Get("Authorization") != "Bearer test-token" || r.URL.Query().Get("query") != "海街日记" || r.URL.Query().Get("language") != "zh-CN" || r.URL.Query().Get("include_adult") != "false" {
			t.Errorf("unexpected TMDB request: %s %v", r.URL, r.Header)
		}
		switch r.URL.Path {
		case "/3/search/movie":
			_, _ = w.Write([]byte(`{"results":[{"id":1,"title":"海街日记","release_date":"2015-06-13","poster_path":"/poster123.jpg"},{"id":2,"title":"成人片","adult":true,"poster_path":"/adult.jpg"},{"id":3,"title":"其他作品","poster_path":"https://bad.example/p.jpg"}]}`))
		case "/3/search/tv":
			_, _ = w.Write([]byte(`{"results":[{"id":4,"name":"请回答1988","first_air_date":"2015-11-06","poster_path":"/show456.png"}]}`))
		default:
			t.Errorf("unexpected TMDB path: %s", r.URL.Path)
		}
	}))
	defer server.Close()
	a.tmdbBaseURL = server.URL
	a.tmdbToken = "test-token"
	for _, path := range []string{"/api/media/search?kind=movies&q=海街日记", "/api/media/search?kind=shows&q=海街日记"} {
		if got := paymentCall(a.searchMedia, "GET", path, "", "", ""); got.Code != 401 {
			t.Fatalf("anonymous search: %d", got.Code)
		}
	}
	if got := paymentCall(a.searchMedia, "GET", "/api/media/search?kind=games&q=海街日记", "alice", "", ""); got.Code != 400 {
		t.Fatalf("invalid kind: %d", got.Code)
	}
	if got := paymentCall(a.searchMedia, "GET", "/api/media/search?kind=movies&q=海街日记", "alice", "", ""); got.Code != 200 {
		t.Fatalf("movie search: %d %s", got.Code, got.Body.String())
	} else {
		var result struct {
			Results []tmdbResult `json:"results"`
		}
		if err := json.Unmarshal(got.Body.Bytes(), &result); err != nil || len(result.Results) != 2 || result.Results[0].PosterPath != "/poster123.jpg" || result.Results[0].Year != "2015" || result.Results[1].PosterPath != "" {
			t.Fatalf("movie results: %+v %v", result, err)
		}
	}
	if got := paymentCall(a.searchMedia, "GET", "/api/media/search?kind=shows&q=海街日记", "alice", "", ""); got.Code != 200 || !strings.Contains(got.Body.String(), `"poster_path":"/show456.png"`) {
		t.Fatalf("TV search: %d %s", got.Code, got.Body.String())
	}
	if requests != 2 {
		t.Fatalf("TMDB request count = %d", requests)
	}
	media := MediaShelf{Movies: []MediaItem{{Title: "海街日记", TMDBID: 1, PosterPath: "/poster123.jpg"}}}
	if !media.valid() || media.Movies[0].PosterPath != "/poster123.jpg" {
		t.Fatalf("valid poster rejected: %+v", media)
	}
	media.Movies[0].PosterPath = "https://bad.example/p.jpg"
	if media.valid() {
		t.Fatal("external poster URL accepted")
	}
}

func TestTMDBPosterProxiesImageForSignedInUser(t *testing.T) {
	a := paymentTestApp(t)
	var poster bytes.Buffer
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	img.Set(0, 0, color.RGBA{R: 15, G: 30, B: 45, A: 255})
	if err := png.Encode(&poster, img); err != nil {
		t.Fatal(err)
	}
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.URL.Path != "/t/p/w92/poster123.png" {
			t.Errorf("unexpected poster path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write(poster.Bytes())
	}))
	defer server.Close()
	a.tmdbImageBaseURL = server.URL
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/media/poster/{size}/{file}", a.mediaPoster)
	call := func(path, user string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		if user != "" {
			req.AddCookie(&http.Cookie{Name: "mm_session", Value: user + "-token"})
		}
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		return w
	}
	if got := call("/api/media/poster/w92/poster123.png", ""); got.Code != 401 {
		t.Fatalf("anonymous poster = %d", got.Code)
	}
	if got := call("/api/media/poster/w92/poster123.png", "alice"); got.Code != 200 || got.Header().Get("Content-Type") != "image/png" || !bytes.Equal(got.Body.Bytes(), poster.Bytes()) {
		t.Fatalf("poster proxy = %d, content type %q", got.Code, got.Header().Get("Content-Type"))
	}
	if got := call("/api/media/poster/w92/not-a-poster.svg", "alice"); got.Code != 400 {
		t.Fatalf("invalid poster = %d", got.Code)
	}
	if requests != 1 {
		t.Fatalf("TMDB image requests = %d", requests)
	}
}
