package main

import (
	"regexp"
	"strings"
)

var tmdbPosterPath = regexp.MustCompile(`^/[A-Za-z0-9_-]+\.(?:jpg|jpeg|png|webp)$`)

type MediaItem struct {
	Title      string `json:"title"`
	Creator    string `json:"creator"`
	TMDBID     int64  `json:"tmdb_id,omitempty"`
	PosterPath string `json:"poster_path,omitempty"`
}

type MediaShelf struct {
	Music  []MediaItem `json:"music"`
	Movies []MediaItem `json:"movies"`
	Books  []MediaItem `json:"books"`
	Shows  []MediaItem `json:"shows"`
	Games  []MediaItem `json:"games"`
}

func (m *MediaShelf) collections() []*[]MediaItem {
	return []*[]MediaItem{&m.Music, &m.Movies, &m.Books, &m.Shows, &m.Games}
}

func (m *MediaShelf) normalize() {
	for _, items := range m.collections() {
		if *items == nil {
			*items = []MediaItem{}
		}
	}
}

func (m *MediaShelf) valid() bool {
	for category, items := range m.collections() {
		if len(*items) > 5 {
			return false
		}
		seen := make(map[string]bool, len(*items))
		for i := range *items {
			item := &(*items)[i]
			item.Title = strings.TrimSpace(item.Title)
			item.Creator = strings.TrimSpace(item.Creator)
			if category != 1 && category != 3 {
				item.TMDBID, item.PosterPath = 0, ""
			} else if item.TMDBID < 0 || item.TMDBID > 2147483647 || item.PosterPath != "" && (item.TMDBID == 0 || !tmdbPosterPath.MatchString(item.PosterPath)) {
				return false
			}
			if len([]rune(item.Title)) < 1 || len([]rune(item.Title)) > 80 || len([]rune(item.Creator)) > 60 {
				return false
			}
			key := strings.ToLower(item.Title + "\x00" + item.Creator)
			if seen[key] {
				return false
			}
			seen[key] = true
		}
	}
	m.normalize()
	return true
}
