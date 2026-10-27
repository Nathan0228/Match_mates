package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

type notificationExec interface {
	Exec(string, ...any) (sql.Result, error)
}

func migrateNotifications(db *sql.DB) error {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS notifications (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		kind TEXT NOT NULL,
		entity_id TEXT NOT NULL,
		title TEXT NOT NULL,
		body TEXT NOT NULL,
		target TEXT NOT NULL,
		created_at INTEGER NOT NULL,
		read_at INTEGER NOT NULL DEFAULT 0,
		UNIQUE(user_id,kind,entity_id)
	);
	CREATE INDEX IF NOT EXISTS notifications_user_recent ON notifications(user_id,created_at DESC,id DESC);
	CREATE INDEX IF NOT EXISTS notifications_user_unread ON notifications(user_id,read_at);
	CREATE TABLE IF NOT EXISTS announcements (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		admin_id TEXT NOT NULL REFERENCES users(id),
		title TEXT NOT NULL,
		body TEXT NOT NULL,
		recipient_count INTEGER NOT NULL DEFAULT 0,
		created_at INTEGER NOT NULL
	);
	CREATE INDEX IF NOT EXISTS announcements_recent ON announcements(created_at DESC,id DESC);`)
	return err
}

func (a *App) registerNotificationRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/notifications", a.notifications)
	mux.HandleFunc("POST /api/notifications/{id}/read", a.readNotification)
	mux.HandleFunc("POST /api/notifications/read-all", a.readAllNotifications)
	mux.HandleFunc("GET /api/admin/announcements", a.adminAnnouncements)
	mux.HandleFunc("POST /api/admin/announcements", a.adminPublishAnnouncement)
}

func addNotification(q notificationExec, userID, kind, entityID, title, body, target string, when int64) error {
	_, err := q.Exec(`INSERT OR IGNORE INTO notifications(user_id,kind,entity_id,title,body,target,created_at) VALUES(?,?,?,?,?,?,?)`, userID, kind, entityID, title, body, target, when)
	return err
}

func (a *App) addAdminNotification(q notificationExec, kind, entityID, title, body, target string, when int64) error {
	if a.adminEmail == "" {
		return nil
	}
	_, err := q.Exec(`INSERT INTO notifications(user_id,kind,entity_id,title,body,target,created_at)
		SELECT id,?,?,?,?,?,? FROM users WHERE email=? AND suspended_at=0
		ON CONFLICT(user_id,kind,entity_id) DO UPDATE SET title=excluded.title,body=excluded.body,target=excluded.target,created_at=excluded.created_at,read_at=0`, kind, entityID, title, body, target, when, a.adminEmail)
	return err
}

func (a *App) syncAdminNotifications(userID string) error {
	if _, err := a.db.Exec(`DELETE FROM notifications WHERE user_id=? AND kind='admin_card_pending'
		AND NOT EXISTS (SELECT 1 FROM users WHERE id=notifications.entity_id AND card_status='pending')`, userID); err != nil {
		return err
	}
	if _, err := a.db.Exec(`DELETE FROM notifications WHERE user_id=? AND kind='admin_payment_pending'
		AND NOT EXISTS (SELECT 1 FROM payment_orders WHERE id=notifications.entity_id AND status='submitted')`, userID); err != nil {
		return err
	}
	if _, err := a.db.Exec(`INSERT OR IGNORE INTO notifications(user_id,kind,entity_id,title,body,target,created_at)
		SELECT ?,'admin_card_pending',id,'学生证待审核',nickname||'提交了学生证，等待审核。','admin:queue',card_submitted_at
		FROM users WHERE card_status='pending'`, userID); err != nil {
		return err
	}
	_, err := a.db.Exec(`INSERT OR IGNORE INTO notifications(user_id,kind,entity_id,title,body,target,created_at)
		SELECT ?,'admin_payment_pending',o.id,'会费待核款',u.nickname||'提交了付款信息，等待核对。','admin:payments',o.submitted_at
		FROM payment_orders o JOIN users u ON u.id=o.user_id WHERE o.status='submitted'`, userID)
	return err
}

func (a *App) notifications(w http.ResponseWriter, r *http.Request) {
	u, ok := a.mustAuth(w, r)
	if !ok {
		return
	}
	if u.IsAdmin {
		if err := a.syncAdminNotifications(u.ID); err != nil {
			fail(w, 500, "获取提醒失败")
			return
		}
	}
	rows, err := a.db.Query(`SELECT id,kind,title,body,target,created_at,read_at FROM notifications WHERE user_id=? ORDER BY created_at DESC,id DESC LIMIT 50`, u.ID)
	if err != nil {
		fail(w, 500, "获取提醒失败")
		return
	}
	type item struct {
		ID        int64  `json:"id"`
		Kind      string `json:"kind"`
		Title     string `json:"title"`
		Body      string `json:"body"`
		Target    string `json:"target"`
		CreatedAt int64  `json:"created_at"`
		ReadAt    int64  `json:"read_at"`
	}
	items := []item{}
	for rows.Next() {
		var n item
		if err := rows.Scan(&n.ID, &n.Kind, &n.Title, &n.Body, &n.Target, &n.CreatedAt, &n.ReadAt); err != nil {
			rows.Close()
			fail(w, 500, "获取提醒失败")
			return
		}
		items = append(items, n)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		fail(w, 500, "获取提醒失败")
		return
	}
	rows.Close()
	var unread int
	if err := a.db.QueryRow(`SELECT COUNT(*) FROM notifications WHERE user_id=? AND read_at=0`, u.ID).Scan(&unread); err != nil {
		fail(w, 500, "获取提醒失败")
		return
	}
	w.Header().Set("Cache-Control", "private, no-store")
	send(w, 200, map[string]any{"notifications": items, "unread_count": unread})
}

func (a *App) readNotification(w http.ResponseWriter, r *http.Request) {
	u, ok := a.mustAuth(w, r)
	if !ok {
		return
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		fail(w, 400, "提醒编号不正确")
		return
	}
	result, err := a.db.Exec(`UPDATE notifications SET read_at=CASE WHEN read_at=0 THEN ? ELSE read_at END WHERE id=? AND user_id=?`, time.Now().Unix(), id, u.ID)
	if err != nil {
		fail(w, 500, "更新提醒失败")
		return
	}
	if count, _ := result.RowsAffected(); count == 0 {
		fail(w, 404, "提醒不存在")
		return
	}
	send(w, 200, map[string]bool{"read": true})
}

func (a *App) readAllNotifications(w http.ResponseWriter, r *http.Request) {
	u, ok := a.mustAuth(w, r)
	if !ok {
		return
	}
	if _, err := a.db.Exec(`UPDATE notifications SET read_at=? WHERE user_id=? AND read_at=0`, time.Now().Unix(), u.ID); err != nil {
		fail(w, 500, "更新提醒失败")
		return
	}
	send(w, 200, map[string]bool{"read": true})
}

func addMutualMatchNotifications(tx *sql.Tx, u, target User) error {
	var uCount, targetCount int
	query := `SELECT COUNT(*) FROM likes mine JOIN likes theirs ON theirs.from_id=mine.to_id AND theirs.to_id=mine.from_id WHERE mine.from_id=?`
	if err := tx.QueryRow(query, u.ID).Scan(&uCount); err != nil {
		return err
	}
	if err := tx.QueryRow(query, target.ID).Scan(&targetCount); err != nil {
		return err
	}
	now := time.Now().Unix()
	if err := addNotification(tx, u.ID, "mutual_match", target.ID, "有新的双向心动", fmt.Sprintf("你已与 %d 位同学互相表达想认识。%s 的校园邮箱已解锁，前往双向心动查看。", uCount, target.Nickname), "matches", now); err != nil {
		return err
	}
	return addNotification(tx, target.ID, "mutual_match", u.ID, "有新的双向心动", fmt.Sprintf("你已与 %d 位同学互相表达想认识。%s 的校园邮箱已解锁，前往双向心动查看。", targetCount, u.Nickname), "matches", now)
}
