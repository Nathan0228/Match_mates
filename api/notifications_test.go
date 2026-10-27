package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

type notificationResponse struct {
	UnreadCount   int `json:"unread_count"`
	Notifications []struct {
		ID     int64  `json:"id"`
		Kind   string `json:"kind"`
		Body   string `json:"body"`
		Target string `json:"target"`
		ReadAt int64  `json:"read_at"`
	} `json:"notifications"`
}

func getNotifications(t *testing.T, a *App, user string) notificationResponse {
	t.Helper()
	w := paymentCall(a.notifications, "GET", "/api/notifications", user, "", "")
	var result notificationResponse
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &result) != nil {
		t.Fatalf("notifications for %s: %d %s", user, w.Code, w.Body.String())
	}
	return result
}

func TestMutualMatchNotificationsAndReadOwnership(t *testing.T) {
	a := paymentTestApp(t)
	for _, item := range []struct{ id, gender, seeking string }{{"alice", "male", "female"}, {"bob", "female", "male"}} {
		_, err := a.db.Exec(`UPDATE users SET gender=?,seeking=?,paid=1,card_status='approved',intro=?,hobbies='["阅读","徒步"]' WHERE id=?`, item.gender, item.seeking, strings.Repeat("喜欢认真交流。", 4), item.id)
		if err != nil {
			t.Fatal(err)
		}
	}
	if got := paymentCall(a.like, "POST", "/api/likes/bob", "alice", "bob", ""); got.Code != 200 {
		t.Fatalf("first like: %d %s", got.Code, got.Body.String())
	}
	if got := getNotifications(t, a, "alice"); got.UnreadCount != 0 {
		t.Fatalf("one-way like produced notification: %+v", got)
	}
	if got := paymentCall(a.like, "POST", "/api/likes/alice", "bob", "alice", ""); got.Code != 200 {
		t.Fatalf("mutual like: %d %s", got.Code, got.Body.String())
	}
	alice := getNotifications(t, a, "alice")
	bob := getNotifications(t, a, "bob")
	if alice.UnreadCount != 1 || bob.UnreadCount != 1 || len(alice.Notifications) != 1 || alice.Notifications[0].Target != "matches" || !strings.Contains(alice.Notifications[0].Body, "1 位同学") {
		t.Fatalf("mutual notifications: alice=%+v bob=%+v", alice, bob)
	}
	id := alice.Notifications[0].ID
	path := "/api/notifications/" + strconv.FormatInt(id, 10) + "/read"
	if got := paymentCall(a.readNotification, "POST", path, "bob", strconv.FormatInt(id, 10), ""); got.Code != 404 {
		t.Fatalf("other user's notification read: %d", got.Code)
	}
	if got := paymentCall(a.readNotification, "POST", path, "alice", strconv.FormatInt(id, 10), ""); got.Code != 200 {
		t.Fatalf("read notification: %d %s", got.Code, got.Body.String())
	}
	if got := getNotifications(t, a, "alice"); got.UnreadCount != 0 || got.Notifications[0].ReadAt == 0 {
		t.Fatalf("read status: %+v", got)
	}
	if got := paymentCall(a.like, "POST", "/api/likes/alice", "bob", "alice", ""); got.Code != 200 {
		t.Fatalf("repeated like: %d", got.Code)
	}
	if got := getNotifications(t, a, "alice"); len(got.Notifications) != 1 || got.UnreadCount != 0 {
		t.Fatalf("duplicate match notification: %+v", got)
	}
	if got := paymentCall(a.readAllNotifications, "POST", "/api/notifications/read-all", "bob", "", ""); got.Code != 200 {
		t.Fatalf("read all: %d", got.Code)
	}
	if got := getNotifications(t, a, "bob"); got.UnreadCount != 0 {
		t.Fatalf("read all status: %+v", got)
	}
}

func TestAdminPendingAndUserReviewNotifications(t *testing.T) {
	a := paymentTestApp(t)
	path := filepath.Join(t.TempDir(), "card.png")
	if err := os.WriteFile(path, []byte("test"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := a.db.Exec(`UPDATE users SET card_status='pending',card_path=?,card_submitted_at=? WHERE id='bob'`, path, time.Now().Unix()); err != nil {
		t.Fatal(err)
	}
	if _, err := a.db.Exec(`INSERT INTO payment_orders(id,user_id,channel,original_cents,discount_cents,payable_cents,code_tail,status,transaction_ref,created_at,submitted_at) VALUES('order-1','alice','wechat',990,0,990,'','submitted','WX123456',?,?)`, time.Now().Unix(), time.Now().Unix()); err != nil {
		t.Fatal(err)
	}
	admin := getNotifications(t, a, "admin")
	if admin.UnreadCount != 2 || len(admin.Notifications) != 2 {
		t.Fatalf("pending admin messages: %+v", admin)
	}
	if got := getNotifications(t, a, "bob"); got.UnreadCount != 0 {
		t.Fatalf("admin messages leaked to student: %+v", got)
	}
	if got := paymentCall(a.adminDecision, "POST", "/api/admin/reviews/bob", "admin", "bob", `{"decision":"approved"}`); got.Code != 200 {
		t.Fatalf("card review: %d %s", got.Code, got.Body.String())
	}
	if got := getNotifications(t, a, "bob"); got.UnreadCount != 1 || got.Notifications[0].Kind != "student_card_approved" {
		t.Fatalf("student approval message: %+v", got)
	}
	if got := paymentCall(a.adminPaymentDecision, "POST", "/api/admin/payment/orders/order-1", "admin", "order-1", `{"decision":"approved","verified":true}`); got.Code != 200 {
		t.Fatalf("payment review: %d %s", got.Code, got.Body.String())
	}
	if got := getNotifications(t, a, "alice"); got.UnreadCount != 1 || got.Notifications[0].Kind != "payment_approved" {
		t.Fatalf("payment approval message: %+v", got)
	}
	if got := getNotifications(t, a, "admin"); got.UnreadCount != 0 || len(got.Notifications) != 0 {
		t.Fatalf("resolved admin messages remain: %+v", got)
	}
}
