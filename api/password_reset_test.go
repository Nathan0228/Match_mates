package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
	_ "modernc.org/sqlite"
)

func TestPasswordResetInvalidatesSessionsAndCode(t *testing.T) {
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
	oldHash, err := bcrypt.GenerateFromPassword([]byte("old-password"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`INSERT INTO users(id,email,password_hash,nickname,email_verified,created_at) VALUES('student','student@mails.ucas.ac.cn',?,'同学',1,?)`, string(oldHash), time.Now().Unix())
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`INSERT INTO sessions(token_hash,user_id,expires_at) VALUES(?,?,?)`, digest("old-session"), "student", time.Now().Add(time.Hour).Unix())
	if err != nil {
		t.Fatal(err)
	}
	a := &App{db: db, dev: true}
	call := func(handler http.HandlerFunc, body string) *httptest.ResponseRecorder {
		r := httptest.NewRequest("POST", "/api/password-reset", strings.NewReader(body))
		w := httptest.NewRecorder()
		handler(w, r)
		return w
	}
	unknown := call(a.requestPasswordReset, `{"email":"unknown@mails.ucas.ac.cn"}`)
	request := call(a.requestPasswordReset, `{"email":"student@mails.ucas.ac.cn"}`)
	if unknown.Code != 200 || request.Code != 200 {
		t.Fatalf("request status: %d, %d", unknown.Code, request.Code)
	}
	var unknownData, requestData map[string]string
	if err := json.Unmarshal(unknown.Body.Bytes(), &unknownData); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(request.Body.Bytes(), &requestData); err != nil {
		t.Fatal(err)
	}
	if unknownData["message"] != requestData["message"] || requestData["dev_code"] == "" {
		t.Fatalf("unexpected request responses: %v, %v", unknownData, requestData)
	}
	second := call(a.requestPasswordReset, `{"email":"student@mails.ucas.ac.cn"}`)
	var secondData map[string]string
	if err := json.Unmarshal(second.Body.Bytes(), &secondData); err != nil {
		t.Fatal(err)
	}
	if secondData["dev_code"] != "" || secondData["message"] != requestData["message"] {
		t.Fatalf("cooldown response: %v", secondData)
	}
	code := requestData["dev_code"]
	bad := call(a.confirmPasswordReset, `{"email":"student@mails.ucas.ac.cn","code":"111111","password":"new-password"}`)
	if code == "111111" {
		bad = call(a.confirmPasswordReset, `{"email":"student@mails.ucas.ac.cn","code":"222222","password":"new-password"}`)
	}
	if bad.Code != 400 {
		t.Fatalf("wrong code status %d", bad.Code)
	}
	goodBody := `{"email":"student@mails.ucas.ac.cn","code":"` + code + `","password":"new-password"}`
	good := call(a.confirmPasswordReset, goodBody)
	if good.Code != 200 {
		t.Fatalf("reset status %d: %s", good.Code, good.Body.String())
	}
	if reused := call(a.confirmPasswordReset, goodBody); reused.Code != 400 {
		t.Fatalf("reused code status %d", reused.Code)
	}
	if oldLogin := call(a.login, `{"email":"student@mails.ucas.ac.cn","password":"old-password"}`); oldLogin.Code != 401 {
		t.Fatalf("old password login status %d", oldLogin.Code)
	}
	if newLogin := call(a.login, `{"email":"student@mails.ucas.ac.cn","password":"new-password"}`); newLogin.Code != 200 {
		t.Fatalf("new password login status %d", newLogin.Code)
	}
	var sessionCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM sessions WHERE token_hash=?`, digest("old-session")).Scan(&sessionCount); err != nil {
		t.Fatal(err)
	}
	if sessionCount != 0 {
		t.Fatal("old session still valid")
	}
}

func TestPasswordResetExpiresAndLimitsAttempts(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	if err := migrate(db); err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`INSERT INTO users(id,email,password_hash,nickname,created_at) VALUES('student','student@mails.ucas.ac.cn','hash','同学',?)`, time.Now().Unix())
	if err != nil {
		t.Fatal(err)
	}
	a := &App{db: db, dev: true}
	call := func(code string) int {
		r := httptest.NewRequest("POST", "/api/password-reset/confirm", strings.NewReader(`{"email":"student@mails.ucas.ac.cn","code":"`+code+`","password":"new-password"}`))
		w := httptest.NewRecorder()
		a.confirmPasswordReset(w, r)
		return w.Code
	}
	_, err = db.Exec(`INSERT INTO password_reset_codes(user_id,code_hash,expires_at,requested_at) VALUES('student',?,?,?)`, digest("123456"), time.Now().Add(-time.Second).Unix(), time.Now().Unix())
	if err != nil {
		t.Fatal(err)
	}
	if status := call("123456"); status != 400 {
		t.Fatalf("expired code status %d", status)
	}
	_, err = db.Exec(`UPDATE password_reset_codes SET expires_at=?,attempts=0`, time.Now().Add(time.Minute).Unix())
	if err != nil {
		t.Fatal(err)
	}
	for range 5 {
		if status := call("000000"); status != 400 {
			t.Fatalf("wrong code status %d", status)
		}
	}
	if status := call("123456"); status != 400 {
		t.Fatalf("code accepted after five attempts: %d", status)
	}
}
