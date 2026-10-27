package main

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

const adminUserPageSize = 20

const matchableUserCondition = "suspended_at=0 AND email_verified=1 AND paid=1 AND card_status='approved' AND matching_enabled=1 AND gender<>'' AND seeking<>'' AND length(intro)>=20 AND json_array_length(hobbies)>=2"

func migrateUserManagement(db *sql.DB) error {
	for _, column := range []struct{ name, definition string }{
		{"suspended_at", "INTEGER NOT NULL DEFAULT 0"},
		{"suspension_reason", "TEXT NOT NULL DEFAULT ''"},
		{"suspended_by", "TEXT NOT NULL DEFAULT ''"},
		{"matching_enabled", "INTEGER NOT NULL DEFAULT 1"},
	} {
		if err := ensureColumn(db, "users", column.name, column.definition); err != nil {
			return err
		}
	}
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS user_admin_events (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id TEXT NOT NULL REFERENCES users(id),
		admin_id TEXT NOT NULL REFERENCES users(id),
		action TEXT NOT NULL CHECK(action IN ('suspended','restored')),
		reason TEXT NOT NULL DEFAULT '',
		created_at INTEGER NOT NULL
	);
	CREATE INDEX IF NOT EXISTS user_admin_events_user ON user_admin_events(user_id,created_at DESC,id DESC);
	CREATE INDEX IF NOT EXISTS users_admin_recent ON users(created_at DESC,id DESC);`)
	return err
}

func (a *App) registerUserManagementRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/admin/users", a.adminUsers)
	mux.HandleFunc("GET /api/admin/users/{id}", a.adminUserDetail)
	mux.HandleFunc("POST /api/admin/users/{id}/status", a.adminUserStatus)
}

type adminUserRow struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	Nickname      string `json:"nickname"`
	Major         string `json:"major"`
	EmailVerified bool   `json:"email_verified"`
	Paid          bool   `json:"paid"`
	CardStatus    string `json:"card_status"`
	SuspendedAt   int64  `json:"suspended_at"`
	CreatedAt     int64  `json:"created_at"`
	IsAdmin       bool   `json:"is_admin"`
}

func (a *App) adminUsers(w http.ResponseWriter, r *http.Request) {
	if _, ok := a.admin(w, r); !ok {
		return
	}
	search := strings.TrimSpace(r.URL.Query().Get("q"))
	filter := r.URL.Query().Get("filter")
	if utf8.RuneCountInString(search) > 80 {
		fail(w, 400, "搜索内容不能超过 80 字")
		return
	}
	if filter == "" {
		filter = "all"
	}
	page := 1
	if raw := r.URL.Query().Get("page"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 || parsed > 100000 {
			fail(w, 400, "页码不正确")
			return
		}
		page = parsed
	}
	where := " WHERE 1=1"
	args := []any{}
	if search != "" {
		where += " AND (instr(lower(email),lower(?))>0 OR instr(lower(nickname),lower(?))>0 OR instr(lower(id),lower(?))>0)"
		args = append(args, search, search, search)
	}
	switch filter {
	case "all":
	case "suspended":
		where += " AND suspended_at>0"
	case "unpaid":
		where += " AND suspended_at=0 AND paid=0"
	case "card_none":
		where += " AND suspended_at=0 AND card_status='none'"
	case "card_pending":
		where += " AND suspended_at=0 AND card_status='pending'"
	case "active":
		where += " AND " + matchableUserCondition
	case "inactive":
		where += " AND NOT (" + matchableUserCondition + ")"
	default:
		fail(w, 400, "筛选条件不正确")
		return
	}
	var total int
	if err := a.db.QueryRow("SELECT COUNT(*) FROM users"+where, args...).Scan(&total); err != nil {
		fail(w, 500, "获取用户失败")
		return
	}
	queryArgs := append(append([]any{}, args...), adminUserPageSize, (page-1)*adminUserPageSize)
	rows, err := a.db.Query(`SELECT id,email,nickname,major,email_verified,paid,card_status,suspended_at,created_at FROM users`+where+` ORDER BY created_at DESC,id DESC LIMIT ? OFFSET ?`, queryArgs...)
	if err != nil {
		fail(w, 500, "获取用户失败")
		return
	}
	defer rows.Close()
	users := []adminUserRow{}
	for rows.Next() {
		var item adminUserRow
		var verified, paid int
		if err := rows.Scan(&item.ID, &item.Email, &item.Nickname, &item.Major, &verified, &paid, &item.CardStatus, &item.SuspendedAt, &item.CreatedAt); err != nil {
			fail(w, 500, "获取用户失败")
			return
		}
		item.EmailVerified, item.Paid = verified != 0, paid != 0
		item.IsAdmin = a.adminEmail != "" && item.Email == a.adminEmail
		users = append(users, item)
	}
	if err := rows.Err(); err != nil {
		fail(w, 500, "获取用户失败")
		return
	}
	send(w, 200, map[string]any{"users": users, "total": total, "page": page, "page_size": adminUserPageSize})
}

func (a *App) adminUserDetail(w http.ResponseWriter, r *http.Request) {
	if _, ok := a.admin(w, r); !ok {
		return
	}
	id := r.PathValue("id")
	u, err := a.getUser(id)
	if errors.Is(err, sql.ErrNoRows) {
		fail(w, 404, "用户不存在")
		return
	}
	if err != nil {
		fail(w, 500, "获取用户失败")
		return
	}
	var createdAt int64
	var reason string
	if err := a.db.QueryRow(`SELECT created_at,suspension_reason FROM users WHERE id=?`, id).Scan(&createdAt, &reason); err != nil {
		fail(w, 500, "获取用户失败")
		return
	}
	type event struct {
		Action     string `json:"action"`
		Reason     string `json:"reason"`
		AdminEmail string `json:"admin_email"`
		CreatedAt  int64  `json:"created_at"`
	}
	events := []event{}
	rows, err := a.db.Query(`SELECT e.action,e.reason,admin.email,e.created_at FROM user_admin_events e JOIN users admin ON admin.id=e.admin_id WHERE e.user_id=? ORDER BY e.created_at DESC,e.id DESC LIMIT 20`, id)
	if err != nil {
		fail(w, 500, "获取管理记录失败")
		return
	}
	for rows.Next() {
		var item event
		if err := rows.Scan(&item.Action, &item.Reason, &item.AdminEmail, &item.CreatedAt); err != nil {
			rows.Close()
			fail(w, 500, "获取管理记录失败")
			return
		}
		events = append(events, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		fail(w, 500, "获取管理记录失败")
		return
	}
	rows.Close()
	send(w, 200, map[string]any{"user": u, "created_at": createdAt, "suspended_at": u.SuspendedAt, "suspension_reason": reason, "events": events})
}

func (a *App) adminUserStatus(w http.ResponseWriter, r *http.Request) {
	admin, ok := a.admin(w, r)
	if !ok {
		return
	}
	var input struct {
		Suspended *bool  `json:"suspended"`
		Reason    string `json:"reason"`
	}
	if !decode(w, r, &input) {
		return
	}
	if input.Suspended == nil {
		fail(w, 400, "请选择账号状态")
		return
	}
	reason := strings.TrimSpace(input.Reason)
	if utf8.RuneCountInString(reason) > 200 || (*input.Suspended && utf8.RuneCountInString(reason) < 4) {
		fail(w, 400, "停用原因须为 4 至 200 字")
		return
	}
	id := r.PathValue("id")
	if id == admin.ID {
		fail(w, 403, "不能修改自己的账号状态")
		return
	}
	tx, err := a.db.Begin()
	if err != nil {
		fail(w, 500, "更新账号失败")
		return
	}
	defer tx.Rollback()
	var email string
	var current int64
	err = tx.QueryRow(`SELECT email,suspended_at FROM users WHERE id=?`, id).Scan(&email, &current)
	if errors.Is(err, sql.ErrNoRows) {
		fail(w, 404, "用户不存在")
		return
	}
	if err != nil {
		fail(w, 500, "更新账号失败")
		return
	}
	if a.adminEmail != "" && email == a.adminEmail {
		fail(w, 403, "不能修改管理员账号状态")
		return
	}
	if (current != 0) == *input.Suspended {
		fail(w, 409, "账号状态未变化，请刷新后重试")
		return
	}
	now := time.Now().Unix()
	action := "restored"
	if *input.Suspended {
		action = "suspended"
		if _, err := tx.Exec(`UPDATE users SET suspended_at=?,suspension_reason=?,suspended_by=? WHERE id=?`, now, reason, admin.ID, id); err != nil {
			fail(w, 500, "更新账号失败")
			return
		}
		if _, err := tx.Exec(`DELETE FROM sessions WHERE user_id=?`, id); err != nil {
			fail(w, 500, "更新账号失败")
			return
		}
	} else if _, err := tx.Exec(`UPDATE users SET suspended_at=0,suspension_reason='',suspended_by='' WHERE id=?`, id); err != nil {
		fail(w, 500, "更新账号失败")
		return
	}
	if _, err := tx.Exec(`INSERT INTO user_admin_events(user_id,admin_id,action,reason,created_at) VALUES(?,?,?,?,?)`, id, admin.ID, action, reason, now); err != nil {
		fail(w, 500, "保存管理记录失败")
		return
	}
	if err := tx.Commit(); err != nil {
		fail(w, 500, "更新账号失败")
		return
	}
	send(w, 200, map[string]any{"status": action})
}
