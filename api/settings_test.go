package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
)

func TestMatchingSettingControlsNewInteractions(t *testing.T) {
	a := paymentTestApp(t)
	for _, item := range []struct{ id, gender, seeking string }{
		{"alice", "male", "female"}, {"bob", "female", "male"},
	} {
		_, err := a.db.Exec(`UPDATE users SET gender=?,seeking=?,paid=1,card_status='approved',intro=?,hobbies='["阅读","徒步"]' WHERE id=?`, item.gender, item.seeking, strings.Repeat("认真交流，一起散步。", 3), item.id)
		if err != nil {
			t.Fatal(err)
		}
	}
	if got := paymentCall(a.candidates, "GET", "/api/candidates", "alice", "", ""); got.Code != 200 || !strings.Contains(got.Body.String(), `"id":"bob"`) {
		t.Fatalf("before pause: %d %s", got.Code, got.Body.String())
	}
	if got := paymentCall(a.updateMatchingSetting, "PUT", "/api/settings/matching", "", "", `{"enabled":false}`); got.Code != 401 {
		t.Fatalf("anonymous setting: %d", got.Code)
	}
	if got := paymentCall(a.updateMatchingSetting, "PUT", "/api/settings/matching", "bob", "", `{}`); got.Code != 400 {
		t.Fatalf("missing flag: %d", got.Code)
	}
	if got := paymentCall(a.updateMatchingSetting, "PUT", "/api/settings/matching", "bob", "", `{"enabled":false}`); got.Code != 200 {
		t.Fatalf("pause: %d %s", got.Code, got.Body.String())
	}
	if got := paymentCall(a.me, "GET", "/api/me", "bob", "", ""); got.Code != 200 || !strings.Contains(got.Body.String(), `"matching_enabled":false`) {
		t.Fatalf("saved setting: %d %s", got.Code, got.Body.String())
	}
	if got := paymentCall(a.candidates, "GET", "/api/candidates", "alice", "", ""); got.Code != 200 || strings.Contains(got.Body.String(), `"id":"bob"`) {
		t.Fatalf("paused user listed: %d %s", got.Code, got.Body.String())
	}
	if got := paymentCall(a.candidates, "GET", "/api/candidates", "bob", "", ""); got.Code != 200 || !strings.Contains(got.Body.String(), `"candidates":[]`) {
		t.Fatalf("paused discovery: %d %s", got.Code, got.Body.String())
	}
	if got := paymentCall(a.like, "POST", "/api/likes/bob", "alice", "bob", ""); got.Code != 403 {
		t.Fatalf("like paused user: %d", got.Code)
	}
	if got := paymentCall(a.like, "POST", "/api/likes/alice", "bob", "alice", ""); got.Code != 403 {
		t.Fatalf("paused user like: %d", got.Code)
	}
	avatarPath := filepath.Join(t.TempDir(), "bob.jpg")
	if err := os.WriteFile(avatarPath, []byte("avatar"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := a.db.Exec(`UPDATE users SET avatar_path=? WHERE id='bob'`, avatarPath); err != nil {
		t.Fatal(err)
	}
	if got := paymentCall(a.avatar, "GET", "/api/avatars/bob", "alice", "bob", ""); got.Code != 404 {
		t.Fatalf("paused avatar exposed before mutual match: %d", got.Code)
	}
	if _, err := a.db.Exec(`INSERT INTO likes(from_id,to_id,created_at) VALUES('alice','bob',1),('bob','alice',1)`); err != nil {
		t.Fatal(err)
	}
	if got := paymentCall(a.matches, "GET", "/api/matches", "alice", "", ""); got.Code != 200 || !strings.Contains(got.Body.String(), `"id":"bob"`) {
		t.Fatalf("existing mutual match hidden: %d %s", got.Code, got.Body.String())
	}
	if got := paymentCall(a.avatar, "GET", "/api/avatars/bob", "alice", "bob", ""); got.Code != 200 {
		t.Fatalf("existing mutual match avatar hidden: %d %s", got.Code, got.Body.String())
	}
	if got := paymentCall(a.adminUsers, "GET", "/api/admin/users?filter=inactive&q=bob", "admin", "", ""); got.Code != 200 || !strings.Contains(got.Body.String(), `"total":1`) {
		t.Fatalf("admin inactive filter: %d %s", got.Code, got.Body.String())
	}
	if got := paymentCall(a.updateMatchingSetting, "PUT", "/api/settings/matching", "bob", "", `{"enabled":true}`); got.Code != 200 {
		t.Fatalf("resume: %d %s", got.Code, got.Body.String())
	}
	if got := paymentCall(a.candidates, "GET", "/api/candidates", "alice", "", ""); got.Code != 200 || !strings.Contains(got.Body.String(), `"id":"bob"`) {
		t.Fatalf("after resume: %d %s", got.Code, got.Body.String())
	}
}

func TestChangePasswordKeepsCurrentSessionOnly(t *testing.T) {
	a := paymentTestApp(t)
	oldHash, err := bcrypt.GenerateFromPassword([]byte("old-password-123"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := a.db.Exec(`UPDATE users SET password_hash=? WHERE id='alice'`, string(oldHash)); err != nil {
		t.Fatal(err)
	}
	if _, err := a.db.Exec(`INSERT INTO sessions(token_hash,user_id,expires_at) VALUES(?,?,?)`, digest("other-device"), "alice", time.Now().Add(time.Hour).Unix()); err != nil {
		t.Fatal(err)
	}
	if got := paymentCall(a.changePassword, "POST", "/api/settings/password", "alice", "", `{"current_password":"wrong-password","new_password":"new-password-123"}`); got.Code != 400 {
		t.Fatalf("wrong current password: %d", got.Code)
	}
	if got := paymentCall(a.changePassword, "POST", "/api/settings/password", "alice", "", `{"current_password":"old-password-123","new_password":"short"}`); got.Code != 400 {
		t.Fatalf("short new password: %d", got.Code)
	}
	if got := paymentCall(a.changePassword, "POST", "/api/settings/password", "alice", "", `{"current_password":"old-password-123","new_password":"new-password-123"}`); got.Code != 200 {
		t.Fatalf("password change: %d %s", got.Code, got.Body.String())
	}
	if got := paymentCall(a.me, "GET", "/api/me", "alice", "", ""); got.Code != 200 {
		t.Fatalf("current session: %d", got.Code)
	}
	var sessionCount int
	if err := a.db.QueryRow(`SELECT COUNT(*) FROM sessions WHERE token_hash=?`, digest("other-device")).Scan(&sessionCount); err != nil || sessionCount != 0 {
		t.Fatalf("other device session: %d %v", sessionCount, err)
	}
	if got := paymentCall(a.login, "POST", "/api/login", "", "", `{"email":"alice@mails.ucas.ac.cn","password":"old-password-123"}`); got.Code != 401 {
		t.Fatalf("old password login: %d", got.Code)
	}
	if got := paymentCall(a.login, "POST", "/api/login", "", "", `{"email":"alice@mails.ucas.ac.cn","password":"new-password-123"}`); got.Code != 200 {
		t.Fatalf("new password login: %d %s", got.Code, got.Body.String())
	}
}
