package main

import (
	"database/sql"
	"errors"
	"log"
	"net/http"
	"os"
	"time"
)

func (a *App) admin(w http.ResponseWriter, r *http.Request) (User, bool) {
	u, ok := a.mustAuth(w, r)
	if !ok {
		return u, false
	}
	if !u.IsAdmin {
		fail(w, 403, "仅管理员可访问")
		return u, false
	}
	return u, true
}

func (a *App) adminSummary(w http.ResponseWriter, r *http.Request) {
	if _, ok := a.admin(w, r); !ok {
		return
	}
	var total, paid, pending, approved, approvedUsers, rejected, suspended int
	err := a.db.QueryRow(`SELECT COUNT(*),
		COALESCE(SUM(CASE WHEN paid=1 THEN 1 ELSE 0 END),0),
		COALESCE(SUM(CASE WHEN card_status='pending' THEN 1 ELSE 0 END),0),
		COALESCE(SUM(CASE WHEN card_status='approved' THEN 1 ELSE 0 END),0),
		COALESCE(SUM(CASE WHEN paid=1 AND card_status='approved' THEN 1 ELSE 0 END),0),
		COALESCE(SUM(CASE WHEN card_status='rejected' THEN 1 ELSE 0 END),0),
		COALESCE(SUM(CASE WHEN suspended_at>0 THEN 1 ELSE 0 END),0)
		FROM users`).Scan(&total, &paid, &pending, &approved, &approvedUsers, &rejected, &suspended)
	if err != nil {
		fail(w, 500, "获取管理概览失败")
		return
	}
	var pendingPayments int
	if err := a.db.QueryRow(`SELECT COUNT(*) FROM payment_orders WHERE status='submitted'`).Scan(&pendingPayments); err != nil {
		fail(w, 500, "获取管理概览失败")
		return
	}
	send(w, 200, map[string]int{"total_users": total, "paid_users": paid, "pending_reviews": pending, "pending_payments": pendingPayments, "approved_cards": approved, "approved_users": approvedUsers, "rejected_cards": rejected, "suspended_users": suspended})
}

func (a *App) adminReviews(w http.ResponseWriter, r *http.Request) {
	if _, ok := a.admin(w, r); !ok {
		return
	}
	rows, err := a.db.Query(`SELECT id,email,nickname,major,campus,card_submitted_at FROM users WHERE card_status='pending' ORDER BY card_submitted_at,id`)
	if err != nil {
		fail(w, 500, "获取待审核列表失败")
		return
	}
	defer rows.Close()
	type review struct {
		ID          string `json:"id"`
		Email       string `json:"email"`
		Nickname    string `json:"nickname"`
		Major       string `json:"major"`
		Campus      string `json:"campus"`
		SubmittedAt int64  `json:"submitted_at"`
	}
	result := []review{}
	for rows.Next() {
		var item review
		if err := rows.Scan(&item.ID, &item.Email, &item.Nickname, &item.Major, &item.Campus, &item.SubmittedAt); err != nil {
			fail(w, 500, "获取待审核列表失败")
			return
		}
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		fail(w, 500, "获取待审核列表失败")
		return
	}
	send(w, 200, map[string]any{"reviews": result})
}

func (a *App) adminHistory(w http.ResponseWriter, r *http.Request) {
	if _, ok := a.admin(w, r); !ok {
		return
	}
	rows, err := a.db.Query(`SELECT e.id,u.nickname,u.email,e.decision,e.created_at,reviewer.email
		FROM review_events e JOIN users u ON u.id=e.user_id JOIN users reviewer ON reviewer.id=e.reviewer_id
		ORDER BY e.created_at DESC,e.id DESC LIMIT 30`)
	if err != nil {
		fail(w, 500, "获取审核记录失败")
		return
	}
	defer rows.Close()
	type event struct {
		ID            int64  `json:"id"`
		Nickname      string `json:"nickname"`
		Email         string `json:"email"`
		Decision      string `json:"decision"`
		CreatedAt     int64  `json:"created_at"`
		ReviewerEmail string `json:"reviewer_email"`
	}
	result := []event{}
	for rows.Next() {
		var item event
		if err := rows.Scan(&item.ID, &item.Nickname, &item.Email, &item.Decision, &item.CreatedAt, &item.ReviewerEmail); err != nil {
			fail(w, 500, "获取审核记录失败")
			return
		}
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		fail(w, 500, "获取审核记录失败")
		return
	}
	send(w, 200, map[string]any{"history": result})
}

func (a *App) adminDecision(w http.ResponseWriter, r *http.Request) {
	reviewer, ok := a.admin(w, r)
	if !ok {
		return
	}
	var v input
	if !decode(w, r, &v) {
		return
	}
	if v.Decision != "approved" && v.Decision != "rejected" {
		fail(w, 400, "决定只能是 approved 或 rejected")
		return
	}
	tx, err := a.db.Begin()
	if err != nil {
		fail(w, 500, "审核失败")
		return
	}
	defer tx.Rollback()
	id := r.PathValue("id")
	var path string
	err = tx.QueryRow(`SELECT card_path FROM users WHERE id=? AND card_status='pending'`, id).Scan(&path)
	if errors.Is(err, sql.ErrNoRows) {
		fail(w, 404, "未找到待审核证件")
		return
	}
	if err != nil {
		fail(w, 500, "审核失败")
		return
	}
	if path == "" {
		fail(w, 404, "未找到待审核证件")
		return
	}
	result, err := tx.Exec(`UPDATE users SET card_status=?,card_path='' WHERE id=? AND card_status='pending'`, v.Decision, id)
	if err != nil {
		fail(w, 500, "审核失败")
		return
	}
	affected, err := result.RowsAffected()
	if err != nil || affected != 1 {
		fail(w, 409, "该申请已处理，请刷新列表")
		return
	}
	if _, err := tx.Exec(`INSERT INTO review_events(user_id,reviewer_id,decision,created_at) VALUES(?,?,?,?)`, id, reviewer.ID, v.Decision, time.Now().Unix()); err != nil {
		fail(w, 500, "审核记录保存失败")
		return
	}
	noticeTitle, noticeBody := "学生证审核已通过", "学生证认证成功。完成其余入驻步骤后即可参与匹配。"
	if v.Decision == "rejected" {
		noticeTitle, noticeBody = "学生证审核未通过", "请检查学生证照片后重新提交。"
	}
	if err := addNotification(tx, id, "student_card_"+v.Decision, digest(path), noticeTitle, noticeBody, "discover", time.Now().Unix()); err != nil {
		fail(w, 500, "审核失败")
		return
	}
	if _, err := tx.Exec(`DELETE FROM notifications WHERE user_id=? AND kind='admin_card_pending' AND entity_id=?`, reviewer.ID, id); err != nil {
		fail(w, 500, "审核失败")
		return
	}
	if err := tx.Commit(); err != nil {
		fail(w, 500, "审核失败")
		return
	}
	fileDeleted := true
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		log.Printf("remove reviewed card: %v", err)
		fileDeleted = false
	}
	send(w, 200, map[string]any{"status": v.Decision, "file_deleted": fileDeleted})
}

func (a *App) adminCard(w http.ResponseWriter, r *http.Request) {
	if _, ok := a.admin(w, r); !ok {
		return
	}
	var path string
	if err := a.db.QueryRow(`SELECT card_path FROM users WHERE id=? AND card_status='pending'`, r.PathValue("id")).Scan(&path); err != nil || path == "" {
		fail(w, 404, "文件不存在")
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	http.ServeFile(w, r, path)
}
