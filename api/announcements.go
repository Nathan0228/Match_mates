package main

import (
	"net/http"
	"strconv"
	"strings"
	"time"
)

type announcement struct {
	ID             int64  `json:"id"`
	Title          string `json:"title"`
	Body           string `json:"body"`
	RecipientCount int64  `json:"recipient_count"`
	CreatedAt      int64  `json:"created_at"`
	AdminEmail     string `json:"admin_email"`
}

func (a *App) adminAnnouncements(w http.ResponseWriter, r *http.Request) {
	if _, ok := a.admin(w, r); !ok {
		return
	}
	rows, err := a.db.Query(`SELECT a.id,a.title,a.body,a.recipient_count,a.created_at,u.email
		FROM announcements a JOIN users u ON u.id=a.admin_id
		ORDER BY a.created_at DESC,a.id DESC LIMIT 30`)
	if err != nil {
		fail(w, 500, "获取发布记录失败")
		return
	}
	defer rows.Close()
	items := []announcement{}
	for rows.Next() {
		var item announcement
		if err := rows.Scan(&item.ID, &item.Title, &item.Body, &item.RecipientCount, &item.CreatedAt, &item.AdminEmail); err != nil {
			fail(w, 500, "获取发布记录失败")
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		fail(w, 500, "获取发布记录失败")
		return
	}
	w.Header().Set("Cache-Control", "private, no-store")
	send(w, 200, map[string]any{"announcements": items})
}

func (a *App) adminPublishAnnouncement(w http.ResponseWriter, r *http.Request) {
	admin, ok := a.admin(w, r)
	if !ok {
		return
	}
	var input struct {
		Title string `json:"title"`
		Body  string `json:"body"`
	}
	if !decode(w, r, &input) {
		return
	}
	title := strings.Join(strings.Fields(input.Title), " ")
	body := strings.TrimSpace(input.Body)
	if len([]rune(title)) < 2 || len([]rune(title)) > 60 || len([]rune(body)) < 1 || len([]rune(body)) > 1000 {
		fail(w, 400, "标题需为 2–60 字，内容需为 1–1000 字")
		return
	}
	now := time.Now().Unix()
	tx, err := a.db.Begin()
	if err != nil {
		fail(w, 500, "发布消息失败")
		return
	}
	defer tx.Rollback()
	result, err := tx.Exec(`INSERT INTO announcements(admin_id,title,body,created_at) VALUES(?,?,?,?)`, admin.ID, title, body, now)
	if err != nil {
		fail(w, 500, "发布消息失败")
		return
	}
	id, err := result.LastInsertId()
	if err != nil {
		fail(w, 500, "发布消息失败")
		return
	}
	result, err = tx.Exec(`INSERT INTO notifications(user_id,kind,entity_id,title,body,target,created_at)
		SELECT id,'announcement',?,?,?,'',? FROM users WHERE id<>?`, strconv.FormatInt(id, 10), title, body, now, admin.ID)
	if err != nil {
		fail(w, 500, "发布消息失败")
		return
	}
	count, err := result.RowsAffected()
	if err != nil {
		fail(w, 500, "发布消息失败")
		return
	}
	if _, err := tx.Exec(`UPDATE announcements SET recipient_count=? WHERE id=?`, count, id); err != nil {
		fail(w, 500, "发布消息失败")
		return
	}
	if err := tx.Commit(); err != nil {
		fail(w, 500, "发布消息失败")
		return
	}
	send(w, 201, announcement{ID: id, Title: title, Body: body, RecipientCount: count, CreatedAt: now, AdminEmail: admin.Email})
}
