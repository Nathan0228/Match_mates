package main

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
	_ "modernc.org/sqlite"
)

func TestRegistrationLocksGenderAndSeeking(t *testing.T) {
	t.Setenv("SMTP_HOST", "")
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
	post := func(handler http.HandlerFunc, body, token string) *httptest.ResponseRecorder {
		r := httptest.NewRequest("POST", "/api/test", strings.NewReader(body))
		if token != "" {
			r.AddCookie(&http.Cookie{Name: "mm_session", Value: token})
		}
		w := httptest.NewRecorder()
		handler(w, r)
		return w
	}
	invalid := post(a.register, `{"email":"new@mails.ucas.ac.cn","password":"password123","nickname":"同学"}`, "")
	if invalid.Code != 400 {
		t.Fatalf("register without direction: %d", invalid.Code)
	}
	valid := post(a.register, `{"email":"new@mails.ucas.ac.cn","password":"password123","nickname":"同学","gender":"male","seeking":"female"}`, "")
	if valid.Code != 201 {
		t.Fatalf("register with direction: %d: %s", valid.Code, valid.Body.String())
	}
	var id, gender, seeking, passwordHash string
	if err := db.QueryRow(`SELECT id,gender,seeking,password_hash FROM users WHERE email='new@mails.ucas.ac.cn'`).Scan(&id, &gender, &seeking, &passwordHash); err != nil {
		t.Fatal(err)
	}
	if passwordHash == "password123" || bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte("password123")) != nil {
		t.Fatal("registered password was not stored as a valid bcrypt hash")
	}
	if gender != "male" || seeking != "female" {
		t.Fatalf("stored direction: %s, %s", gender, seeking)
	}
	_, err = db.Exec(`INSERT INTO sessions(token_hash,user_id,expires_at) VALUES(?,?,?)`, digest("session"), id, time.Now().Add(time.Hour).Unix())
	if err != nil {
		t.Fatal(err)
	}
	profileBody := func(gender, seeking string) string {
		return `{"nickname":"同学","gender":"` + gender + `","seeking":"` + seeking + `","major":"计算机","campus":"雁栖湖","intro":"喜欢徒步和电影，想认识愿意一起在湖边散步的人。","hobbies":["徒步","电影"]}`
	}
	change := post(a.profile, profileBody("female", "male"), "session")
	if change.Code != 409 {
		t.Fatalf("changed direction: %d: %s", change.Code, change.Body.String())
	}
	same := post(a.profile, profileBody("male", "female"), "session")
	if same.Code != 200 {
		t.Fatalf("unchanged direction: %d: %s", same.Code, same.Body.String())
	}
	withPreferences := post(a.profile, `{"nickname":"同学","gender":"male","seeking":"female","major":"计算机","campus":"雁栖湖","intro":"喜欢徒步和电影，想认识愿意一起在湖边散步的人。","hobbies":["徒步","电影"],"preferences":{"values":["真诚沟通","尊重边界"],"pace":"balanced","contact":"deep","energy":3,"rhythm":"flexible","mbti":"INFP","daily":"周末散步","together":"一起看展"}}`, "session")
	if withPreferences.Code != 200 {
		t.Fatalf("save preferences: %d: %s", withPreferences.Code, withPreferences.Body.String())
	}
	stored, err := a.getUser(id)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Preferences.Pace != "balanced" || len(stored.Preferences.Values) != 2 || stored.Preferences.Together != "一起看展" {
		t.Fatalf("preferences not saved: %+v", stored.Preferences)
	}
}

func TestLegacyAccountCanSetDirectionOnce(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	if err := migrate(db); err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`INSERT INTO users(id,email,password_hash,nickname,created_at) VALUES('old','old@mails.ucas.ac.cn','hash','旧同学',?)`, time.Now().Unix())
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`INSERT INTO sessions(token_hash,user_id,expires_at) VALUES(?,?,?)`, digest("legacy"), "old", time.Now().Add(time.Hour).Unix())
	if err != nil {
		t.Fatal(err)
	}
	a := &App{db: db}
	call := func(gender string) int {
		body := `{"nickname":"旧同学","gender":"` + gender + `","seeking":"female","major":"计算机","campus":"雁栖湖","intro":"喜欢徒步和电影，想认识愿意一起在湖边散步的人。","hobbies":["徒步","电影"]}`
		r := httptest.NewRequest("PUT", "/api/profile", strings.NewReader(body))
		r.AddCookie(&http.Cookie{Name: "mm_session", Value: "legacy"})
		w := httptest.NewRecorder()
		a.profile(w, r)
		return w.Code
	}
	if status := call("male"); status != 200 {
		t.Fatalf("first direction: %d", status)
	}
	if status := call("female"); status != 409 {
		t.Fatalf("changed legacy direction: %d", status)
	}
}
