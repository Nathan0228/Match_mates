package main

import (
	"database/sql"
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func TestSimilarityIdentity(t *testing.T) {
	prefs := MatchPreferences{Values: []string{"真诚沟通", "共同成长"}, Pace: "slow", Contact: "deep", Energy: 3, Rhythm: "flexible"}
	u := User{Campus: "雁栖湖", Hobbies: []string{"徒步", "电影", "咖啡"}, Preferences: prefs}
	c := Candidate{Campus: u.Campus, Hobbies: u.Hobbies, Preferences: prefs}
	if got := similarity(u, c); math.Abs(got-1) > 0.0001 {
		t.Fatalf("same profile score = %v, want 1", got)
	}
}

func TestSimilarityMissingFieldsAndMBTI(t *testing.T) {
	u := User{Campus: "雁栖湖", Hobbies: []string{"徒步", "电影"}, Preferences: MatchPreferences{MBTI: "INFP"}}
	c := Candidate{Campus: "雁栖湖", Hobbies: []string{"徒步", "电影"}, Preferences: MatchPreferences{MBTI: "ESTJ"}}
	got := similarity(u, c)
	if got <= 0.5 || got >= 0.8 {
		t.Fatalf("partial profile score = %v, want neutral-shrunk score", got)
	}
	c.Preferences.MBTI = "INFP"
	if score := similarity(u, c); math.Abs(score-got) > 0.0001 {
		t.Fatalf("MBTI changed score: %v -> %v", got, score)
	}
	c.Hobbies = []string{"阅读", "音乐"}
	if score := similarity(u, c); score >= got {
		t.Fatalf("unshared interests did not reduce score: %v -> %v", got, score)
	}
}

func TestSimilarityIsSymmetric(t *testing.T) {
	a := User{Campus: "雁栖湖", Hobbies: []string{"徒步", "电影", "阅读"}, Preferences: MatchPreferences{Values: []string{"尊重边界", "共同成长"}, Pace: "slow", Contact: "deep", Energy: 2, Rhythm: "early"}}
	b := User{Campus: "玉泉路", Hobbies: []string{"电影", "阅读", "音乐"}, Preferences: MatchPreferences{Values: []string{"共同成长", "热爱生活"}, Pace: "soon", Contact: "frequent", Energy: 4, Rhythm: "late"}}
	toCandidate := func(u User) Candidate {
		return Candidate{Campus: u.Campus, Hobbies: u.Hobbies, Preferences: u.Preferences}
	}
	if x, y := similarity(a, toCandidate(b)), similarity(b, toCandidate(a)); math.Abs(x-y) > 0.0001 || x < 0 || x > 1 {
		t.Fatalf("asymmetric or out of bounds: %v vs %v", x, y)
	}
}

func TestSimilarityDissimilar(t *testing.T) {
	u := User{Major: "计算机", Campus: "雁栖湖", Intro: "徒步电影读书摄影", Hobbies: []string{"徒步", "电影"}}
	c := Candidate{Major: "生物", Campus: "玉泉路", Intro: "羽毛球做饭音乐骑行", Hobbies: []string{"做饭", "音乐"}}
	if got := similarity(u, c); got >= 0.8 {
		t.Fatalf("unrelated profiles scored %v", got)
	}
}

func TestCampusEmail(t *testing.T) {
	for _, s := range []string{"a@mails.ucas.ac.cn", "student.name@mails.ucas.ac.cn"} {
		if !validEmail(s) {
			t.Fatalf("rejected %q", s)
		}
	}
	for _, s := range []string{"a@gmail.com", "a@ucas.ac.cn", "@mails.ucas.ac.cn", "a@mails.ucas.ac.cn.evil.com"} {
		if validEmail(s) {
			t.Fatalf("accepted %q", s)
		}
	}
}

func TestMatchingRequiresReciprocalDirectionAndLikes(t *testing.T) {
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
	intro := "喜欢徒步和电影，想认识愿意一起在湖边散步的人。"
	for _, u := range []struct{ id, email, gender, seeking string }{
		{"a", "a@mails.ucas.ac.cn", "male", "female"},
		{"b", "b@mails.ucas.ac.cn", "female", "male"},
		{"c", "c@mails.ucas.ac.cn", "female", "female"},
	} {
		_, err = db.Exec(`INSERT INTO users(id,email,password_hash,nickname,gender,seeking,major,campus,intro,hobbies,email_verified,paid,card_status,created_at) VALUES(?,?,?,?,?,?,?,?,?,?,1,1,'approved',?)`, u.id, u.email, "hash", u.id, u.gender, u.seeking, "计算机", "雁栖湖", intro, `["徒步","电影"]`, time.Now().Unix())
		if err != nil {
			t.Fatal(err)
		}
		token := u.id + "-token"
		_, err = db.Exec(`INSERT INTO sessions(token_hash,user_id,expires_at) VALUES(?,?,?)`, digest(token), u.id, time.Now().Add(time.Hour).Unix())
		if err != nil {
			t.Fatal(err)
		}
	}
	_, err = db.Exec(`INSERT INTO users(id,email,password_hash,nickname,gender,seeking,major,campus,intro,hobbies,email_verified,paid,card_status,created_at) VALUES('d','d@mails.ucas.ac.cn','hash','d','female','male','生物','玉泉路',?,'["阅读","音乐"]',1,1,'approved',?)`, intro, time.Now().Unix())
	if err != nil {
		t.Fatal(err)
	}
	call := func(user, method, path string, handler http.HandlerFunc) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, nil)
		req.AddCookie(&http.Cookie{Name: "mm_session", Value: user + "-token"})
		if strings.Contains(path, "/likes/") {
			req.SetPathValue("id", path[strings.LastIndex(path, "/")+1:])
		}
		w := httptest.NewRecorder()
		handler(w, req)
		return w
	}
	first := call("a", "GET", "/api/candidates", a.candidates)
	if first.Code != 200 {
		t.Fatalf("candidates status %d: %s", first.Code, first.Body.String())
	}
	var candidates struct {
		Candidates []Candidate `json:"candidates"`
	}
	if err := json.Unmarshal(first.Body.Bytes(), &candidates); err != nil {
		t.Fatal(err)
	}
	if len(candidates.Candidates) != 2 || candidates.Candidates[0].ID != "b" || candidates.Candidates[1].ID != "d" || candidates.Candidates[1].Score >= 0.8 {
		t.Fatalf("expected reciprocal candidates including lower-scoring d, got %+v", candidates.Candidates)
	}
	before := call("a", "GET", "/api/matches", a.matches)
	if strings.Contains(before.Body.String(), "b@mails") {
		t.Fatal("email leaked before mutual likes")
	}
	call("a", "POST", "/api/likes/b", a.like)
	oneSided := call("a", "GET", "/api/matches", a.matches)
	if strings.Contains(oneSided.Body.String(), "b@mails") {
		t.Fatal("email leaked after only one like")
	}
	call("b", "POST", "/api/likes/a", a.like)
	mutual := call("a", "GET", "/api/matches", a.matches)
	if !strings.Contains(mutual.Body.String(), "b@mails.ucas.ac.cn") {
		t.Fatalf("email not unlocked after mutual likes: %s", mutual.Body.String())
	}
}
