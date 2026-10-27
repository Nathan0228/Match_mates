package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func TestAdminReviewWorkflow(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	if err := migrate(db); err != nil {
		t.Fatal(err)
	}
	card := filepath.Join(t.TempDir(), "student.png")
	if err := os.WriteFile(card, []byte("student card"), 0600); err != nil {
		t.Fatal(err)
	}
	a := &App{db: db, adminEmail: "admin@mails.ucas.ac.cn"}
	for _, user := range []struct {
		id, email, status, path string
		paid                    int
	}{
		{"admin", "admin@mails.ucas.ac.cn", "none", "", 1},
		{"student", "student@mails.ucas.ac.cn", "pending", card, 1},
		{"other", "other@mails.ucas.ac.cn", "rejected", "", 1},
		{"unpaid-approved", "unpaid-approved@mails.ucas.ac.cn", "approved", "", 0},
	} {
		_, err := db.Exec(`INSERT INTO users(id,email,password_hash,nickname,major,campus,paid,card_status,card_path,card_submitted_at,created_at) VALUES(?,?,?,?,?,?,?,?,?,?,?)`, user.id, user.email, "hash", user.id, "计算机", "雁栖湖", user.paid, user.status, user.path, time.Now().Unix(), time.Now().Unix())
		if err != nil {
			t.Fatal(err)
		}
		_, err = db.Exec(`INSERT INTO sessions(token_hash,user_id,expires_at) VALUES(?,?,?)`, digest(user.id+"-token"), user.id, time.Now().Add(time.Hour).Unix())
		if err != nil {
			t.Fatal(err)
		}
	}
	call := func(handler http.HandlerFunc, method, path, token, id, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		if token != "" {
			req.AddCookie(&http.Cookie{Name: "mm_session", Value: token + "-token"})
		}
		if id != "" {
			req.SetPathValue("id", id)
		}
		response := httptest.NewRecorder()
		handler(response, req)
		return response
	}
	if got := call(a.adminReviews, "GET", "/api/admin/reviews", "", "", "").Code; got != 401 {
		t.Fatalf("anonymous reviews status = %d", got)
	}
	if got := call(a.adminSummary, "GET", "/api/admin/summary", "student", "", "").Code; got != 403 {
		t.Fatalf("non-admin summary status = %d", got)
	}
	if got := call(a.adminCard, "GET", "/api/admin/cards/student", "student", "student", "").Code; got != 403 {
		t.Fatalf("non-admin card status = %d", got)
	}
	if got := call(a.adminDecision, "POST", "/api/admin/reviews/student", "student", "student", `{"decision":"approved"}`).Code; got != 403 {
		t.Fatalf("non-admin decision status = %d", got)
	}
	if _, err := os.Stat(card); err != nil {
		t.Fatalf("unauthorized decision removed card: %v", err)
	}

	summary := call(a.adminSummary, "GET", "/api/admin/summary", "admin", "", "")
	if summary.Code != 200 || !strings.Contains(summary.Body.String(), `"pending_reviews":1`) || !strings.Contains(summary.Body.String(), `"approved_users":0`) {
		t.Fatalf("summary: %d %s", summary.Code, summary.Body.String())
	}
	queue := call(a.adminReviews, "GET", "/api/admin/reviews", "admin", "", "")
	var pending struct {
		Reviews []struct {
			ID          string `json:"id"`
			SubmittedAt int64  `json:"submitted_at"`
		} `json:"reviews"`
	}
	if queue.Code != 200 || json.Unmarshal(queue.Body.Bytes(), &pending) != nil || len(pending.Reviews) != 1 || pending.Reviews[0].ID != "student" || pending.Reviews[0].SubmittedAt == 0 {
		t.Fatalf("pending queue: %d %s", queue.Code, queue.Body.String())
	}
	if got := call(a.adminCard, "GET", "/api/admin/cards/student", "admin", "student", "").Code; got != 200 {
		t.Fatalf("admin card status = %d", got)
	}
	if got := call(a.adminDecision, "POST", "/api/admin/reviews/student", "admin", "student", `{"decision":"invalid"}`).Code; got != 400 {
		t.Fatalf("invalid decision status = %d", got)
	}
	approved := call(a.adminDecision, "POST", "/api/admin/reviews/student", "admin", "student", `{"decision":"approved"}`)
	if approved.Code != 200 || !strings.Contains(approved.Body.String(), `"file_deleted":true`) {
		t.Fatalf("approve: %d %s", approved.Code, approved.Body.String())
	}
	summary = call(a.adminSummary, "GET", "/api/admin/summary", "admin", "", "")
	if summary.Code != 200 || !strings.Contains(summary.Body.String(), `"approved_users":1`) {
		t.Fatalf("summary after approval: %d %s", summary.Code, summary.Body.String())
	}
	if _, err := os.Stat(card); !os.IsNotExist(err) {
		t.Fatalf("reviewed card still exists: %v", err)
	}
	if got := call(a.adminCard, "GET", "/api/admin/cards/student", "admin", "student", "").Code; got != 404 {
		t.Fatalf("reviewed card status = %d", got)
	}
	if got := call(a.adminDecision, "POST", "/api/admin/reviews/student", "admin", "student", `{"decision":"rejected"}`).Code; got != 404 {
		t.Fatalf("repeat decision status = %d", got)
	}
	secondCard := filepath.Join(t.TempDir(), "replacement.png")
	if err := os.WriteFile(secondCard, []byte("replacement card"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE users SET card_status='pending',card_path=?,card_submitted_at=? WHERE id='other'`, secondCard, time.Now().Unix()); err != nil {
		t.Fatal(err)
	}
	rejected := call(a.adminDecision, "POST", "/api/admin/reviews/other", "admin", "other", `{"decision":"rejected"}`)
	if rejected.Code != 200 || !strings.Contains(rejected.Body.String(), `"file_deleted":true`) {
		t.Fatalf("reject: %d %s", rejected.Code, rejected.Body.String())
	}
	if _, err := os.Stat(secondCard); !os.IsNotExist(err) {
		t.Fatalf("rejected card still exists: %v", err)
	}
	queue = call(a.adminReviews, "GET", "/api/admin/reviews", "admin", "", "")
	if json.Unmarshal(queue.Body.Bytes(), &pending) != nil || len(pending.Reviews) != 0 {
		t.Fatalf("queue after review: %s", queue.Body.String())
	}
	history := call(a.adminHistory, "GET", "/api/admin/history", "admin", "", "")
	if history.Code != 200 || !strings.Contains(history.Body.String(), `"decision":"approved"`) || !strings.Contains(history.Body.String(), `"decision":"rejected"`) || !strings.Contains(history.Body.String(), `"reviewer_email":"admin@mails.ucas.ac.cn"`) {
		t.Fatalf("history: %d %s", history.Code, history.Body.String())
	}
}

func TestAdminMigrationAddsSubmissionTime(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	_, err = db.Exec(`CREATE TABLE users (id TEXT PRIMARY KEY, email TEXT NOT NULL UNIQUE, password_hash TEXT NOT NULL, nickname TEXT NOT NULL,
		gender TEXT NOT NULL DEFAULT '', seeking TEXT NOT NULL DEFAULT '', major TEXT NOT NULL DEFAULT '', campus TEXT NOT NULL DEFAULT '雁栖湖',
		intro TEXT NOT NULL DEFAULT '', hobbies TEXT NOT NULL DEFAULT '[]', email_verified INTEGER NOT NULL DEFAULT 0, paid INTEGER NOT NULL DEFAULT 0,
		card_status TEXT NOT NULL DEFAULT 'none', card_path TEXT NOT NULL DEFAULT '', created_at INTEGER NOT NULL)`)
	if err != nil {
		t.Fatal(err)
	}
	if err := migrate(db); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO users(id,email,password_hash,nickname,created_at) VALUES('old','old@mails.ucas.ac.cn','hash','旧用户',0)`); err != nil {
		t.Fatalf("migrated user insert: %v", err)
	}
	var submitted int64
	if err := db.QueryRow(`SELECT card_submitted_at FROM users WHERE id='old'`).Scan(&submitted); err != nil || submitted != 0 {
		t.Fatalf("submission column: %d %v", submitted, err)
	}
}
