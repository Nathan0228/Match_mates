package main

import (
	"encoding/json"
	"strconv"
	"strings"
	"testing"
)

func TestAdminAnnouncementPublishingAndReading(t *testing.T) {
	a := paymentTestApp(t)
	path := "/api/admin/announcements"
	message := `{"title":" 服务公告 ","body":"本周五晚间进行维护。\n请提前保存资料。"}`
	if got := paymentCall(a.adminPublishAnnouncement, "POST", path, "", "", message); got.Code != 401 {
		t.Fatalf("anonymous publish status = %d", got.Code)
	}
	if got := paymentCall(a.adminPublishAnnouncement, "POST", path, "alice", "", message); got.Code != 403 {
		t.Fatalf("student publish status = %d", got.Code)
	}
	if got := paymentCall(a.adminAnnouncements, "GET", path, "alice", "", ""); got.Code != 403 {
		t.Fatalf("student history status = %d", got.Code)
	}
	if got := paymentCall(a.adminPublishAnnouncement, "POST", path, "admin", "", `{"title":" ","body":"内容"}`); got.Code != 400 {
		t.Fatalf("blank title status = %d", got.Code)
	}
	if got := paymentCall(a.adminPublishAnnouncement, "POST", path, "admin", "", `{"title":"公告","body":"`+strings.Repeat("字", 1001)+`"}`); got.Code != 400 {
		t.Fatalf("oversized body status = %d", got.Code)
	}
	published := paymentCall(a.adminPublishAnnouncement, "POST", path, "admin", "", message)
	if published.Code != 201 {
		t.Fatalf("publish: %d %s", published.Code, published.Body.String())
	}
	var item announcement
	if err := json.Unmarshal(published.Body.Bytes(), &item); err != nil {
		t.Fatal(err)
	}
	if item.Title != "服务公告" || item.RecipientCount != 2 || item.AdminEmail != "admin@mails.ucas.ac.cn" {
		t.Fatalf("published item: %+v", item)
	}
	for _, user := range []string{"alice", "bob"} {
		got := getNotifications(t, a, user)
		if got.UnreadCount != 1 || len(got.Notifications) != 1 || got.Notifications[0].Kind != "announcement" || got.Notifications[0].Target != "" || !strings.Contains(got.Notifications[0].Body, "请提前保存资料") {
			t.Fatalf("%s announcement: %+v", user, got)
		}
	}
	if got := getNotifications(t, a, "admin"); got.UnreadCount != 0 || len(got.Notifications) != 0 {
		t.Fatalf("admin received own announcement: %+v", got)
	}
	aliceNotificationID := strconv.FormatInt(getNotifications(t, a, "alice").Notifications[0].ID, 10)
	if got := paymentCall(a.readNotification, "POST", "/api/notifications/"+aliceNotificationID+"/read", "alice", aliceNotificationID, ""); got.Code != 200 {
		t.Fatalf("read announcement: %d %s", got.Code, got.Body.String())
	}
	if got := getNotifications(t, a, "alice"); got.UnreadCount != 0 || got.Notifications[0].ReadAt == 0 {
		t.Fatalf("announcement read state: %+v", got)
	}
	history := paymentCall(a.adminAnnouncements, "GET", path, "admin", "", "")
	var list struct {
		Announcements []announcement `json:"announcements"`
	}
	if history.Code != 200 || json.Unmarshal(history.Body.Bytes(), &list) != nil || len(list.Announcements) != 1 || list.Announcements[0].ID != item.ID || list.Announcements[0].RecipientCount != 2 {
		t.Fatalf("history: %d %s", history.Code, history.Body.String())
	}
}
