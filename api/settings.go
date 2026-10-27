package main

import (
	"net/http"

	"golang.org/x/crypto/bcrypt"
)

func (a *App) updateMatchingSetting(w http.ResponseWriter, r *http.Request) {
	u, ok := a.mustAuth(w, r)
	if !ok {
		return
	}
	var v struct {
		Enabled *bool `json:"enabled"`
	}
	if !decode(w, r, &v) {
		return
	}
	if v.Enabled == nil {
		fail(w, 400, "请选择匹配展示状态")
		return
	}
	if _, err := a.db.Exec(`UPDATE users SET matching_enabled=? WHERE id=?`, *v.Enabled, u.ID); err != nil {
		fail(w, 500, "保存设置失败")
		return
	}
	send(w, 200, map[string]bool{"matching_enabled": *v.Enabled})
}

func (a *App) changePassword(w http.ResponseWriter, r *http.Request) {
	u, ok := a.mustAuth(w, r)
	if !ok {
		return
	}
	var v struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}
	if !decode(w, r, &v) {
		return
	}
	if len(v.NewPassword) < 8 || len(v.NewPassword) > 128 {
		fail(w, 400, "新密码需为 8–128 位")
		return
	}
	var currentHash string
	if err := a.db.QueryRow(`SELECT password_hash FROM users WHERE id=?`, u.ID).Scan(&currentHash); err != nil {
		fail(w, 500, "修改密码失败")
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(currentHash), []byte(v.CurrentPassword)) != nil {
		fail(w, 400, "当前密码不正确")
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(currentHash), []byte(v.NewPassword)) == nil {
		fail(w, 400, "新密码不能与当前密码相同")
		return
	}
	newHash, err := bcrypt.GenerateFromPassword([]byte(v.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		fail(w, 500, "修改密码失败")
		return
	}
	currentSession, _ := r.Cookie("mm_session")
	tx, err := a.db.Begin()
	if err != nil {
		fail(w, 500, "修改密码失败")
		return
	}
	defer tx.Rollback()
	result, err := tx.Exec(`UPDATE users SET password_hash=? WHERE id=? AND password_hash=?`, string(newHash), u.ID, currentHash)
	if err != nil {
		fail(w, 500, "修改密码失败")
		return
	}
	changed, err := result.RowsAffected()
	if err != nil || changed != 1 {
		fail(w, 409, "密码已发生变化，请重新登录后再试")
		return
	}
	if _, err := tx.Exec(`DELETE FROM sessions WHERE user_id=? AND token_hash<>?`, u.ID, digest(currentSession.Value)); err != nil {
		fail(w, 500, "修改密码失败")
		return
	}
	if _, err := tx.Exec(`DELETE FROM password_reset_codes WHERE user_id=?`, u.ID); err != nil {
		fail(w, 500, "修改密码失败")
		return
	}
	if err := tx.Commit(); err != nil {
		fail(w, 500, "修改密码失败")
		return
	}
	send(w, 200, map[string]string{"message": "密码已更新，其他设备需要重新登录"})
}
