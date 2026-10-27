package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

const resetMessage = "如果该邮箱已注册，密码重置验证码将发送到校园邮箱，请在 10 分钟内使用"

func (a *App) requestPasswordReset(w http.ResponseWriter, r *http.Request) {
	var v input
	if !decode(w, r, &v) {
		return
	}
	email := strings.ToLower(strings.TrimSpace(v.Email))
	if !validEmail(email) {
		fail(w, 400, "仅接受 @mails.ucas.ac.cn 校园邮箱")
		return
	}
	out := map[string]any{"message": resetMessage}
	var id string
	if err := a.db.QueryRow(`SELECT id FROM users WHERE email=?`, email).Scan(&id); err != nil {
		send(w, 200, out)
		return
	}
	var requestedAt int64
	err := a.db.QueryRow(`SELECT requested_at FROM password_reset_codes WHERE user_id=?`, id).Scan(&requestedAt)
	if err == nil && time.Now().Unix()-requestedAt < 60 {
		send(w, 200, out)
		return
	}
	if err != nil && err != sql.ErrNoRows {
		log.Printf("password reset lookup: %v", err)
		send(w, 200, out)
		return
	}
	code := fmt.Sprintf("%06d", randomNumber(1000000))
	now := time.Now().Unix()
	_, err = a.db.Exec(`INSERT INTO password_reset_codes(user_id,code_hash,expires_at,requested_at,attempts) VALUES(?,?,?,?,0)
		ON CONFLICT(user_id) DO UPDATE SET code_hash=excluded.code_hash,expires_at=excluded.expires_at,requested_at=excluded.requested_at,attempts=0`, id, digest(code), now+600, now)
	if err != nil {
		log.Printf("password reset save: %v", err)
		send(w, 200, out)
		return
	}
	if err := sendEmail(email, "Jianshan password reset", "您的见山密码重置验证码是 "+code+"，10 分钟内有效。如果不是您本人操作，请忽略此邮件。", a.dev); err != nil {
		log.Printf("password reset email: %v", err)
		_, _ = a.db.Exec(`DELETE FROM password_reset_codes WHERE user_id=? AND code_hash=?`, id, digest(code))
		send(w, 200, out)
		return
	}
	if a.dev && os.Getenv("SMTP_HOST") == "" {
		out["dev_code"] = code
	}
	send(w, 200, out)
}

func (a *App) confirmPasswordReset(w http.ResponseWriter, r *http.Request) {
	var v input
	if !decode(w, r, &v) {
		return
	}
	if !validEmail(v.Email) || len(v.Code) != 6 || len(v.Password) < 8 || len(v.Password) > 128 {
		fail(w, 400, "请填写校园邮箱、6 位验证码和 8–128 位新密码")
		return
	}
	for _, digit := range v.Code {
		if digit < '0' || digit > '9' {
			fail(w, 400, "验证码应为 6 位数字")
			return
		}
	}
	email := strings.ToLower(strings.TrimSpace(v.Email))
	tx, err := a.db.Begin()
	if err != nil {
		fail(w, 500, "重置失败，请稍后再试")
		return
	}
	defer tx.Rollback()
	var id, codeHash string
	var expiresAt int64
	var attempts int
	err = tx.QueryRow(`SELECT u.id,c.code_hash,c.expires_at,c.attempts FROM users u JOIN password_reset_codes c ON c.user_id=u.id WHERE u.email=?`, email).Scan(&id, &codeHash, &expiresAt, &attempts)
	if err != nil || expiresAt < time.Now().Unix() || attempts >= 5 {
		fail(w, 400, "验证码无效或已过期，请重新获取")
		return
	}
	if digest(v.Code) != codeHash {
		if _, err := tx.Exec(`UPDATE password_reset_codes SET attempts=attempts+1 WHERE user_id=?`, id); err != nil || tx.Commit() != nil {
			fail(w, 500, "重置失败，请稍后再试")
			return
		}
		fail(w, 400, "验证码无效或已过期，请重新获取")
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(v.Password), bcrypt.DefaultCost)
	if err != nil {
		fail(w, 500, "重置失败，请稍后再试")
		return
	}
	if _, err = tx.Exec(`UPDATE users SET password_hash=? WHERE id=?`, string(hash), id); err != nil {
		fail(w, 500, "重置失败，请稍后再试")
		return
	}
	if _, err = tx.Exec(`DELETE FROM sessions WHERE user_id=?`, id); err != nil {
		fail(w, 500, "重置失败，请稍后再试")
		return
	}
	if _, err = tx.Exec(`DELETE FROM password_reset_codes WHERE user_id=?`, id); err != nil || tx.Commit() != nil {
		fail(w, 500, "重置失败，请稍后再试")
		return
	}
	send(w, 200, map[string]string{"message": "密码已重置，请使用新密码登录"})
}
