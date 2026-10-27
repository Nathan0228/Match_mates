package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func TestMediaShelfProfileAndCandidate(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	if err := migrate(db); err != nil {
		t.Fatal(err)
	}
	a := &App{db: db, dev: true}
	intro := "喜欢在闲暇时间听音乐、看电影和读书，也想认识能分享作品的人。"
	for _, user := range []struct{ id, gender, seeking string }{{"a", "male", "female"}, {"b", "female", "male"}} {
		_, err := db.Exec(`INSERT INTO users(id,email,password_hash,nickname,gender,seeking,intro,hobbies,email_verified,paid,card_status,created_at) VALUES(?,?,?,?,?,?,?,'["音乐","阅读"]',1,1,'approved',?)`, user.id, user.id+"@mails.ucas.ac.cn", "hash", user.id, user.gender, user.seeking, intro, time.Now().Unix())
		if err != nil {
			t.Fatal(err)
		}
		_, err = db.Exec(`INSERT INTO sessions(token_hash,user_id,expires_at) VALUES(?,?,?)`, digest(user.id+"-token"), user.id, time.Now().Add(time.Hour).Unix())
		if err != nil {
			t.Fatal(err)
		}
	}

	media := MediaShelf{
		Music:  []MediaItem{{Title: " 稻香 ", Creator: " 周杰伦 "}},
		Movies: []MediaItem{{Title: "海街日记", Creator: "是枝裕和", TMDBID: 123, PosterPath: "/movie123.jpg"}},
		Books:  []MediaItem{{Title: "悉达多", Creator: "赫尔曼·黑塞"}},
		Shows:  []MediaItem{{Title: "请回答1988", TMDBID: 456, PosterPath: "/show456.jpg"}},
		Games:  []MediaItem{{Title: "双人成行"}},
	}
	profile := input{Nickname: "a", Gender: "male", Seeking: "female", Intro: intro, Hobbies: []string{"音乐", "阅读"}, Media: &media}
	save := func(value input) *httptest.ResponseRecorder {
		body, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		req := httptest.NewRequest(http.MethodPut, "/api/profile", strings.NewReader(string(body)))
		req.AddCookie(&http.Cookie{Name: "mm_session", Value: "a-token"})
		response := httptest.NewRecorder()
		a.profile(response, req)
		return response
	}
	if response := save(profile); response.Code != 200 {
		t.Fatalf("save media: %d %s", response.Code, response.Body.String())
	}
	owner, err := a.getUser("a")
	if err != nil || len(owner.Media.Music) != 1 || owner.Media.Music[0].Title != "稻香" || owner.Media.Music[0].Creator != "周杰伦" {
		t.Fatalf("media not saved and trimmed: %+v, %v", owner.Media, err)
	}
	if len(owner.Media.Books) != 1 || len(owner.Media.Shows) != 1 || len(owner.Media.Games) != 1 || len(owner.Media.Movies) != 1 || owner.Media.Movies[0].PosterPath != "/movie123.jpg" {
		t.Fatalf("missing media categories: %+v", owner.Media)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/candidates", nil)
	req.AddCookie(&http.Cookie{Name: "mm_session", Value: "b-token"})
	response := httptest.NewRecorder()
	a.candidates(response, req)
	if response.Code != 200 {
		t.Fatalf("get candidates: %d %s", response.Code, response.Body.String())
	}
	var result struct {
		Candidates []Candidate `json:"candidates"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if len(result.Candidates) != 1 || result.Candidates[0].Media.Music[0].Title != "稻香" || result.Candidates[0].Media.Shows[0].PosterPath != "/show456.jpg" {
		t.Fatalf("candidate media missing: %+v", result.Candidates)
	}

	profile.Media.Music = []MediaItem{{Title: ""}}
	if response := save(profile); response.Code != 400 {
		t.Fatalf("empty title accepted: %d", response.Code)
	}
	profile.Media.Music = []MediaItem{{Title: "稻香"}}
	profile.Media.Movies[0].PosterPath = "https://bad.example/poster.jpg"
	if response := save(profile); response.Code != 400 {
		t.Fatalf("external poster URL accepted: %d", response.Code)
	}
	profile.Media.Movies[0].PosterPath = "/movie123.jpg"
	profile.Media.Music = []MediaItem{{Title: "同一首歌"}, {Title: "同一首歌"}}
	if response := save(profile); response.Code != 400 {
		t.Fatalf("duplicate title accepted: %d", response.Code)
	}
	profile.Media.Music = []MediaItem{{Title: "一"}, {Title: "二"}, {Title: "三"}, {Title: "四"}, {Title: "五"}, {Title: "六"}}
	if response := save(profile); response.Code != 400 {
		t.Fatalf("too many items accepted: %d", response.Code)
	}
	if owner, err = a.getUser("a"); err != nil || owner.Media.Music[0].Title != "稻香" {
		t.Fatalf("invalid update changed saved media: %+v, %v", owner.Media, err)
	}
}
