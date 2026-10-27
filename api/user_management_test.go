package main

import (
	"encoding/json"
	"strconv"
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestAdminUserManagementSuspensionAndRestore(t *testing.T) {
	a := paymentTestApp(t)
	hash, err := bcrypt.GenerateFromPassword([]byte("safe-password-123"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"alice", "bob"} {
		gender, seeking := "female", "male"
		if id == "alice" {
			gender, seeking = "male", "female"
		}
		_, err := a.db.Exec(`UPDATE users SET password_hash=?,gender=?,seeking=?,email_verified=1,paid=1,card_status='approved',intro=?,hobbies='["阅读","徒步"]' WHERE id=?`, string(hash), gender, seeking, strings.Repeat("喜欢认真交流。", 4), id)
		if err != nil {
			t.Fatal(err)
		}
	}
	if got := paymentCall(a.adminUsers, "GET", "/api/admin/users", "alice", "", "").Code; got != 403 {
		t.Fatalf("non-admin list = %d", got)
	}
	if got := paymentCall(a.adminUserDetail, "GET", "/api/admin/users/bob", "alice", "bob", "").Code; got != 403 {
		t.Fatalf("non-admin detail = %d", got)
	}
	if got := paymentCall(a.adminUserStatus, "POST", "/api/admin/users/bob/status", "alice", "bob", `{"suspended":true,"reason":"违规行为待核实"}`).Code; got != 403 {
		t.Fatalf("non-admin status change = %d", got)
	}
	list := paymentCall(a.adminUsers, "GET", "/api/admin/users?filter=active&q=bob&page=1", "admin", "", "")
	if list.Code != 200 || !strings.Contains(list.Body.String(), `"total":1`) || strings.Contains(list.Body.String(), "password_hash") {
		t.Fatalf("filtered list: %d %s", list.Code, list.Body.String())
	}
	for _, check := range []struct {
		filter, query string
		total         int
	}{
		{"inactive", "bob", 0},
		{"inactive", "admin", 1},
		{"card_none", "admin", 1},
		{"card_none", "bob", 0},
	} {
		got := paymentCall(a.adminUsers, "GET", "/api/admin/users?filter="+check.filter+"&q="+check.query, "admin", "", "")
		if got.Code != 200 || !strings.Contains(got.Body.String(), `"total":`+strconv.Itoa(check.total)) {
			t.Fatalf("%s filter for %s: %d %s", check.filter, check.query, got.Code, got.Body.String())
		}
	}
	if got := paymentCall(a.adminUsers, "GET", "/api/admin/users?filter=unverified", "admin", "", ""); got.Code != 400 {
		t.Fatalf("removed unverified filter: %d %s", got.Code, got.Body.String())
	}
	before := paymentCall(a.candidates, "GET", "/api/candidates", "alice", "", "")
	if before.Code != 200 || !strings.Contains(before.Body.String(), `"id":"bob"`) {
		t.Fatalf("candidate before suspension: %d %s", before.Code, before.Body.String())
	}
	if _, err := a.db.Exec(`INSERT INTO likes(from_id,to_id,created_at) VALUES('alice','bob',1),('bob','alice',1)`); err != nil {
		t.Fatal(err)
	}
	if got := paymentCall(a.adminUserStatus, "POST", "/api/admin/users/admin/status", "admin", "admin", `{"suspended":true,"reason":"测试停用管理员"}`).Code; got != 403 {
		t.Fatalf("self suspension = %d", got)
	}
	if got := paymentCall(a.adminUserStatus, "POST", "/api/admin/users/bob/status", "admin", "bob", `{"suspended":true,"reason":"短"}`).Code; got != 400 {
		t.Fatalf("short reason = %d", got)
	}
	suspended := paymentCall(a.adminUserStatus, "POST", "/api/admin/users/bob/status", "admin", "bob", `{"suspended":true,"reason":"违规行为待核实"}`)
	if suspended.Code != 200 {
		t.Fatalf("suspend: %d %s", suspended.Code, suspended.Body.String())
	}
	if got := paymentCall(a.me, "GET", "/api/me", "bob", "", "").Code; got != 401 {
		t.Fatalf("old session after suspension = %d", got)
	}
	login := paymentCall(a.login, "POST", "/api/login", "", "", `{"email":"bob@mails.ucas.ac.cn","password":"safe-password-123"}`)
	if login.Code != 403 {
		t.Fatalf("suspended login: %d %s", login.Code, login.Body.String())
	}
	after := paymentCall(a.candidates, "GET", "/api/candidates", "alice", "", "")
	if after.Code != 200 || strings.Contains(after.Body.String(), `"id":"bob"`) {
		t.Fatalf("suspended candidate exposed: %d %s", after.Code, after.Body.String())
	}
	if matches := paymentCall(a.matches, "GET", "/api/matches", "alice", "", ""); matches.Code != 200 || strings.Contains(matches.Body.String(), `"id":"bob"`) {
		t.Fatalf("suspended match exposed: %d %s", matches.Code, matches.Body.String())
	}
	detail := paymentCall(a.adminUserDetail, "GET", "/api/admin/users/bob", "admin", "bob", "")
	var got struct {
		User struct {
			Status string `json:"status"`
		} `json:"user"`
		SuspendedAt int64 `json:"suspended_at"`
		Events      []struct {
			Action string `json:"action"`
		} `json:"events"`
	}
	if detail.Code != 200 || json.Unmarshal(detail.Body.Bytes(), &got) != nil || got.User.Status != "suspended" || got.SuspendedAt == 0 || len(got.Events) != 1 || got.Events[0].Action != "suspended" {
		t.Fatalf("suspended detail: %d %s", detail.Code, detail.Body.String())
	}
	if got := paymentCall(a.adminUsers, "GET", "/api/admin/users?filter=suspended&q=bob", "admin", "", ""); got.Code != 200 || !strings.Contains(got.Body.String(), `"total":1`) {
		t.Fatalf("suspended filter: %d %s", got.Code, got.Body.String())
	}
	if got := paymentCall(a.adminUsers, "GET", "/api/admin/users?filter=inactive&q=bob", "admin", "", ""); got.Code != 200 || !strings.Contains(got.Body.String(), `"total":1`) {
		t.Fatalf("inactive filter after suspension: %d %s", got.Code, got.Body.String())
	}
	restored := paymentCall(a.adminUserStatus, "POST", "/api/admin/users/bob/status", "admin", "bob", `{"suspended":false,"reason":"已核查并恢复"}`)
	if restored.Code != 200 {
		t.Fatalf("restore: %d %s", restored.Code, restored.Body.String())
	}
	if got := paymentCall(a.me, "GET", "/api/me", "bob", "", "").Code; got != 401 {
		t.Fatalf("revoked session revived after restore = %d", got)
	}
	if got := paymentCall(a.login, "POST", "/api/login", "", "", `{"email":"bob@mails.ucas.ac.cn","password":"safe-password-123"}`).Code; got != 200 {
		t.Fatalf("login after restore = %d", got)
	}
	var events int
	if err := a.db.QueryRow(`SELECT COUNT(*) FROM user_admin_events WHERE user_id='bob'`).Scan(&events); err != nil || events != 2 {
		t.Fatalf("audit events = %d, %v", events, err)
	}
}
