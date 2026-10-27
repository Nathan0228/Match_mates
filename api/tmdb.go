package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const defaultTMDBBaseURL = "https://api.themoviedb.org"
const defaultTMDBImageBaseURL = "https://image.tmdb.org"

type tmdbResult struct {
	ID         int64  `json:"id"`
	Title      string `json:"title"`
	Year       string `json:"year"`
	PosterPath string `json:"poster_path"`
}

func (a *App) searchMedia(w http.ResponseWriter, r *http.Request) {
	if _, ok := a.mustAuth(w, r); !ok {
		return
	}
	kind := r.URL.Query().Get("kind")
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if kind != "movies" && kind != "shows" || len([]rune(query)) < 2 || len([]rune(query)) > 80 {
		fail(w, 400, "请选择电影或剧集，并填写 2–80 字的名称")
		return
	}
	if a.tmdbToken == "" && a.tmdbAPIKey == "" {
		fail(w, 503, "尚未配置 TMDB API 凭证")
		return
	}
	base := a.tmdbBaseURL
	if base == "" {
		base = defaultTMDBBaseURL
	}
	endpoint, err := url.Parse(base)
	if err != nil {
		fail(w, 500, "TMDB 接口配置不正确")
		return
	}
	endpoint.Path = "/3/search/movie"
	if kind == "shows" {
		endpoint.Path = "/3/search/tv"
	}
	params := endpoint.Query()
	params.Set("query", query)
	params.Set("language", "zh-CN")
	params.Set("include_adult", "false")
	params.Set("page", "1")
	if a.tmdbToken == "" {
		params.Set("api_key", a.tmdbAPIKey)
	}
	endpoint.RawQuery = params.Encode()
	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, endpoint.String(), nil)
	if err != nil {
		fail(w, 500, "TMDB 查询失败")
		return
	}
	req.Header.Set("Accept", "application/json")
	if a.tmdbToken != "" {
		req.Header.Set("Authorization", "Bearer "+a.tmdbToken)
	}
	response, err := a.tmdbClient().Do(req)
	if err != nil {
		message := "无法连接 TMDB，请检查后端网络或代理设置"
		failureKind := "connection"
		var netErr net.Error
		var dnsErr *net.DNSError
		if errors.Is(err, context.DeadlineExceeded) || errors.As(err, &netErr) && netErr.Timeout() {
			message = "连接 TMDB 超时，请检查后端网络或代理设置"
			failureKind = "timeout"
		} else if errors.As(err, &dnsErr) {
			failureKind = "dns"
		}
		// net/url.Error may contain an api_key query parameter; never log err here.
		log.Printf("TMDB search connection failed: %s", failureKind)
		fail(w, 502, message)
		return
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		message := "TMDB 服务暂不可用，请稍后再试"
		if response.StatusCode == http.StatusUnauthorized || response.StatusCode == http.StatusForbidden {
			message = "TMDB 凭证无效或权限不足"
		} else if response.StatusCode == http.StatusTooManyRequests {
			message = "TMDB 请求过于频繁，请稍后再试"
		}
		log.Printf("TMDB search returned HTTP %d", response.StatusCode)
		fail(w, 502, message)
		return
	}
	var data struct {
		Results []struct {
			ID           int64  `json:"id"`
			Title        string `json:"title"`
			Name         string `json:"name"`
			ReleaseDate  string `json:"release_date"`
			FirstAirDate string `json:"first_air_date"`
			PosterPath   string `json:"poster_path"`
			Adult        bool   `json:"adult"`
		} `json:"results"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&data); err != nil {
		fail(w, 502, "TMDB 返回的数据无法读取")
		return
	}
	items := make([]tmdbResult, 0, 6)
	for _, result := range data.Results {
		if result.ID <= 0 || result.Adult || len(items) >= 6 {
			continue
		}
		title, date := result.Title, result.ReleaseDate
		if kind == "shows" {
			title, date = result.Name, result.FirstAirDate
		}
		if title == "" {
			continue
		}
		poster := result.PosterPath
		if !tmdbPosterPath.MatchString(poster) {
			poster = ""
		}
		year := ""
		if len(date) >= 4 {
			year = date[:4]
		}
		items = append(items, tmdbResult{ID: result.ID, Title: title, Year: year, PosterPath: poster})
	}
	w.Header().Set("Cache-Control", "private, no-store")
	send(w, 200, map[string]any{"results": items})
}

func (a *App) tmdbClient() *http.Client {
	if a.tmdbHTTPClient != nil {
		return a.tmdbHTTPClient
	}
	return &http.Client{Timeout: 12 * time.Second}
}

func (a *App) mediaPoster(w http.ResponseWriter, r *http.Request) {
	if _, ok := a.mustAuth(w, r); !ok {
		return
	}
	size, file := r.PathValue("size"), r.PathValue("file")
	if size != "w92" && size != "w342" || !tmdbPosterPath.MatchString("/"+file) {
		fail(w, 400, "海报路径不正确")
		return
	}
	base := a.tmdbImageBaseURL
	if base == "" {
		base = defaultTMDBImageBaseURL
	}
	endpoint, err := url.Parse(base)
	if err != nil {
		fail(w, 500, "海报接口配置不正确")
		return
	}
	endpoint.Path = "/t/p/" + size + "/" + file
	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, endpoint.String(), nil)
	if err != nil {
		fail(w, 500, "海报请求失败")
		return
	}
	response, err := a.tmdbClient().Do(req)
	if err != nil {
		log.Print("TMDB poster connection failed")
		fail(w, 502, "海报暂不可用，请检查后端网络或代理设置")
		return
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		log.Printf("TMDB poster returned HTTP %d", response.StatusCode)
		fail(w, 502, "海报暂不可用")
		return
	}
	const maxPosterSize = 5 << 20
	poster, err := io.ReadAll(io.LimitReader(response.Body, maxPosterSize+1))
	if err != nil || len(poster) > maxPosterSize {
		fail(w, 502, "海报读取失败")
		return
	}
	contentType := http.DetectContentType(poster)
	if contentType != "image/jpeg" && contentType != "image/png" && contentType != "image/webp" {
		fail(w, 502, "海报格式不正确")
		return
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "private, max-age=3600")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(poster)
}
