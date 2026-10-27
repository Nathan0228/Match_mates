package main

import (
	"database/sql"
	"errors"
	"net/http"
	"strings"
	"time"
)

func (a *App) adminPromoCodes(w http.ResponseWriter, r *http.Request) {
	if _, ok := a.admin(w, r); !ok {
		return
	}
	rows, err := a.db.Query(`SELECT id,code_tail,kind,pay_percent,max_uses,used_count,valid_from,valid_until,active,created_at,code_ciphertext<>'' FROM promo_codes ORDER BY created_at DESC,id DESC LIMIT 200`)
	if err != nil {
		fail(w, 500, "获取代码失败")
		return
	}
	defer rows.Close()
	type item struct {
		ID            string `json:"id"`
		CodeTail      string `json:"code_tail"`
		Kind          string `json:"kind"`
		PayPercent    int    `json:"pay_percent"`
		MaxUses       int    `json:"max_uses"`
		UsedCount     int    `json:"used_count"`
		ValidFrom     int64  `json:"valid_from"`
		ValidUntil    int64  `json:"valid_until"`
		Active        bool   `json:"active"`
		CreatedAt     int64  `json:"created_at"`
		CopyAvailable bool   `json:"copy_available"`
	}
	list := []item{}
	for rows.Next() {
		var v item
		var active, copyAvailable int
		if err := rows.Scan(&v.ID, &v.CodeTail, &v.Kind, &v.PayPercent, &v.MaxUses, &v.UsedCount, &v.ValidFrom, &v.ValidUntil, &active, &v.CreatedAt, &copyAvailable); err != nil {
			fail(w, 500, "获取代码失败")
			return
		}
		v.Active = active != 0
		v.CopyAvailable = copyAvailable != 0
		list = append(list, v)
	}
	if err := rows.Err(); err != nil {
		fail(w, 500, "获取代码失败")
		return
	}
	send(w, 200, map[string]any{"codes": list})
}

func (a *App) adminCreatePromoCode(w http.ResponseWriter, r *http.Request) {
	admin, ok := a.admin(w, r)
	if !ok {
		return
	}
	var v struct {
		Kind       string `json:"kind"`
		PayPercent int    `json:"pay_percent"`
		MaxUses    int    `json:"max_uses"`
		ValidFrom  int64  `json:"valid_from"`
		ValidUntil int64  `json:"valid_until"`
	}
	if !decode(w, r, &v) {
		return
	}
	if err := validatePromoCreate(v.Kind, v.PayPercent, v.MaxUses, v.ValidFrom, v.ValidUntil); err != nil {
		fail(w, 400, err.Error())
		return
	}
	if v.ValidUntil <= time.Now().Unix() {
		fail(w, 400, "结束时间必须晚于当前时间")
		return
	}
	if v.Kind == "exchange" {
		v.PayPercent = 0
	} else {
		v.MaxUses = 0
	}
	code := "JM-" + strings.ToUpper(randomHex(12))
	id := randomHex(16)
	ciphertext, err := a.encryptPromoCode(id, code)
	if err != nil {
		fail(w, 500, "生成代码失败")
		return
	}
	_, err = a.db.Exec(`INSERT INTO promo_codes(id,code_hash,code_tail,code_ciphertext,kind,pay_percent,max_uses,valid_from,valid_until,created_at,created_by) VALUES(?,?,?,?,?,?,?,?,?,?,?)`, id, digest(code), code[len(code)-4:], ciphertext, v.Kind, v.PayPercent, v.MaxUses, v.ValidFrom, v.ValidUntil, time.Now().Unix(), admin.ID)
	if err != nil {
		fail(w, 500, "生成代码失败")
		return
	}
	send(w, 201, map[string]any{"id": id, "code": code, "kind": v.Kind})
}

func (a *App) adminPromoCodeValue(w http.ResponseWriter, r *http.Request) {
	if _, ok := a.admin(w, r); !ok {
		return
	}
	w.Header().Set("Cache-Control", "private, no-store")
	var hash, ciphertext string
	err := a.db.QueryRow(`SELECT code_hash,code_ciphertext FROM promo_codes WHERE id=?`, r.PathValue("id")).Scan(&hash, &ciphertext)
	if errors.Is(err, sql.ErrNoRows) {
		fail(w, 404, "代码不存在")
		return
	}
	if err != nil {
		fail(w, 500, "读取代码失败")
		return
	}
	if ciphertext == "" {
		fail(w, 409, "旧代码只保存了摘要，无法恢复完整内容；请生成新代码")
		return
	}
	code, err := a.decryptPromoCode(r.PathValue("id"), ciphertext)
	if err != nil || digest(code) != hash {
		fail(w, 500, "读取代码失败，请检查代码加密密钥")
		return
	}
	send(w, 200, map[string]string{"code": code})
}

func (a *App) adminPromoCodeStatus(w http.ResponseWriter, r *http.Request) {
	if _, ok := a.admin(w, r); !ok {
		return
	}
	var v struct {
		Active *bool `json:"active"`
	}
	if !decode(w, r, &v) {
		return
	}
	if v.Active == nil {
		fail(w, 400, "请指定启用状态")
		return
	}
	active := 0
	if *v.Active {
		active = 1
	}
	result, err := a.db.Exec(`UPDATE promo_codes SET active=? WHERE id=?`, active, r.PathValue("id"))
	if err != nil {
		fail(w, 500, "更新失败")
		return
	}
	if n, _ := result.RowsAffected(); n != 1 {
		fail(w, 404, "代码不存在")
		return
	}
	send(w, 200, map[string]any{"active": *v.Active})
}

func (a *App) adminPromoUses(w http.ResponseWriter, r *http.Request) {
	if _, ok := a.admin(w, r); !ok {
		return
	}
	rows, err := a.db.Query(`SELECT v.id,u.nickname,u.email,p.kind,p.code_tail,v.original_cents,v.deducted_cents,v.payable_cents,COALESCE(v.order_id,''),v.created_at FROM promo_uses v JOIN users u ON u.id=v.user_id JOIN promo_codes p ON p.id=v.code_id ORDER BY v.created_at DESC,v.id DESC LIMIT 100`)
	if err != nil {
		fail(w, 500, "获取使用记录失败")
		return
	}
	defer rows.Close()
	type item struct {
		ID            int64  `json:"id"`
		Nickname      string `json:"nickname"`
		Email         string `json:"email"`
		Kind          string `json:"kind"`
		CodeTail      string `json:"code_tail"`
		OriginalCents int    `json:"original_cents"`
		DiscountCents int    `json:"discount_cents"`
		PayableCents  int    `json:"payable_cents"`
		OrderID       string `json:"order_id"`
		CreatedAt     int64  `json:"created_at"`
	}
	list := []item{}
	for rows.Next() {
		var v item
		if err := rows.Scan(&v.ID, &v.Nickname, &v.Email, &v.Kind, &v.CodeTail, &v.OriginalCents, &v.DiscountCents, &v.PayableCents, &v.OrderID, &v.CreatedAt); err != nil {
			fail(w, 500, "获取使用记录失败")
			return
		}
		list = append(list, v)
	}
	if err := rows.Err(); err != nil {
		fail(w, 500, "获取使用记录失败")
		return
	}
	send(w, 200, map[string]any{"uses": list})
}

type adminPaymentOrder struct {
	ID             string `json:"id"`
	Nickname       string `json:"nickname"`
	Email          string `json:"email"`
	Channel        string `json:"channel"`
	PayeeName      string `json:"payee_name"`
	OriginalCents  int    `json:"original_cents"`
	DiscountCents  int    `json:"discount_cents"`
	PayableCents   int    `json:"payable_cents"`
	CodeTail       string `json:"code_tail"`
	Status         string `json:"status"`
	TransactionRef string `json:"transaction_ref"`
	ReviewNote     string `json:"review_note"`
	SubmittedAt    int64  `json:"submitted_at"`
	ReviewedAt     int64  `json:"reviewed_at"`
	DuplicateRef   bool   `json:"duplicate_ref"`
}

func (a *App) listPaymentOrders(status string, limit int) ([]adminPaymentOrder, error) {
	rows, err := a.db.Query(`SELECT o.id,u.nickname,u.email,o.channel,o.payee_name,o.original_cents,o.discount_cents,o.payable_cents,o.code_tail,o.status,o.transaction_ref,o.review_note,o.submitted_at,o.reviewed_at,
	EXISTS(SELECT 1 FROM payment_orders x WHERE x.id<>o.id AND x.channel=o.channel AND x.transaction_ref=o.transaction_ref AND x.transaction_ref<>'' AND x.status IN ('submitted','approved'))
	FROM payment_orders o JOIN users u ON u.id=o.user_id WHERE o.status=? ORDER BY CASE WHEN ?='submitted' THEN o.submitted_at ELSE o.reviewed_at END DESC,o.id DESC LIMIT ?`, status, status, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []adminPaymentOrder{}
	for rows.Next() {
		var item adminPaymentOrder
		var duplicate int
		if err := rows.Scan(&item.ID, &item.Nickname, &item.Email, &item.Channel, &item.PayeeName, &item.OriginalCents, &item.DiscountCents, &item.PayableCents, &item.CodeTail, &item.Status, &item.TransactionRef, &item.ReviewNote, &item.SubmittedAt, &item.ReviewedAt, &duplicate); err != nil {
			return nil, err
		}
		item.DuplicateRef = duplicate != 0
		items = append(items, item)
	}
	return items, rows.Err()
}

func (a *App) adminPaymentOrders(w http.ResponseWriter, r *http.Request) {
	if _, ok := a.admin(w, r); !ok {
		return
	}
	pending, err := a.listPaymentOrders("submitted", 100)
	if err != nil {
		fail(w, 500, "获取待核款订单失败")
		return
	}
	approved, err := a.listPaymentOrders("approved", 20)
	if err != nil {
		fail(w, 500, "获取核款记录失败")
		return
	}
	rejected, err := a.listPaymentOrders("rejected", 20)
	if err != nil {
		fail(w, 500, "获取核款记录失败")
		return
	}
	send(w, 200, map[string]any{"pending": pending, "approved": approved, "rejected": rejected})
}

func (a *App) adminPaymentDecision(w http.ResponseWriter, r *http.Request) {
	admin, ok := a.admin(w, r)
	if !ok {
		return
	}
	var v struct {
		Decision string `json:"decision"`
		Verified bool   `json:"verified"`
		Note     string `json:"note"`
	}
	if !decode(w, r, &v) {
		return
	}
	v.Note = strings.TrimSpace(v.Note)
	if v.Decision != "approved" && v.Decision != "rejected" {
		fail(w, 400, "请选择通过或拒绝")
		return
	}
	if v.Decision == "approved" && !v.Verified {
		fail(w, 400, "请先确认已在收款账户核对到账金额和交易单号")
		return
	}
	if v.Decision == "rejected" && (len([]rune(v.Note)) < 4 || len([]rune(v.Note)) > 200) {
		fail(w, 400, "请填写 4 至 200 字的未通过原因")
		return
	}
	tx, err := a.db.Begin()
	if err != nil {
		fail(w, 500, "核款失败")
		return
	}
	defer tx.Rollback()
	var userID, channel, ref, status string
	var original, discount, payable int
	var promoID sql.NullString
	err = tx.QueryRow(`SELECT user_id,channel,transaction_ref,status,original_cents,discount_cents,payable_cents,promo_code_id FROM payment_orders WHERE id=?`, r.PathValue("id")).Scan(&userID, &channel, &ref, &status, &original, &discount, &payable, &promoID)
	if errors.Is(err, sql.ErrNoRows) {
		fail(w, 404, "订单不存在")
		return
	}
	if err != nil {
		fail(w, 500, "核款失败")
		return
	}
	if status != "submitted" {
		fail(w, 409, "该订单不是待核款状态")
		return
	}
	var paid int
	if err := tx.QueryRow(`SELECT paid FROM users WHERE id=?`, userID).Scan(&paid); err != nil {
		fail(w, 500, "核款失败")
		return
	}
	if paid != 0 {
		fail(w, 409, "该用户的会费已完成")
		return
	}
	now := time.Now().Unix()
	if v.Decision == "approved" {
		var duplicate int
		if err := tx.QueryRow(`SELECT EXISTS(SELECT 1 FROM payment_orders WHERE id<>? AND channel=? AND transaction_ref=? AND status='approved')`, r.PathValue("id"), channel, ref).Scan(&duplicate); err != nil {
			fail(w, 500, "核款失败")
			return
		}
		if duplicate != 0 {
			fail(w, 409, "该交易单号已用于其他通过的订单")
			return
		}
		if promoID.Valid {
			if _, err := tx.Exec(`INSERT INTO promo_uses(code_id,user_id,order_id,kind,original_cents,deducted_cents,payable_cents,created_at) VALUES(?,?,?,'discount',?,?,?,?)`, promoID.String, userID, r.PathValue("id"), original, discount, payable, now); err != nil {
				fail(w, 409, "折扣码使用记录冲突")
				return
			}
			if _, err := tx.Exec(`UPDATE promo_codes SET used_count=used_count+1 WHERE id=?`, promoID.String); err != nil {
				fail(w, 500, "核款失败")
				return
			}
		}
		result, err := tx.Exec(`UPDATE users SET paid=1 WHERE id=? AND paid=0`, userID)
		if err != nil {
			fail(w, 500, "核款失败")
			return
		}
		if n, _ := result.RowsAffected(); n != 1 {
			fail(w, 409, "该用户的会费已完成")
			return
		}
	}
	result, err := tx.Exec(`UPDATE payment_orders SET status=?,review_note=?,reviewed_at=?,reviewer_id=? WHERE id=? AND status='submitted'`, v.Decision, v.Note, now, admin.ID, r.PathValue("id"))
	if err != nil {
		fail(w, 500, "核款失败")
		return
	}
	if n, _ := result.RowsAffected(); n != 1 {
		fail(w, 409, "该订单已处理")
		return
	}
	if _, err := tx.Exec(`INSERT INTO payment_events(order_id,actor_id,event,transaction_ref,note,created_at) VALUES(?,?,?,?,?,?)`, r.PathValue("id"), admin.ID, v.Decision, ref, v.Note, now); err != nil {
		fail(w, 500, "核款失败")
		return
	}
	noticeTitle, noticeBody := "会费核款已通过", "付款已核实，会费步骤已完成。"
	if v.Decision == "rejected" {
		noticeTitle, noticeBody = "会费核款未通过", "付款信息未通过核对："+v.Note+"。请修改交易单号后重新提交。"
	}
	if err := addNotification(tx, userID, "payment_"+v.Decision, randomHex(8), noticeTitle, noticeBody, "payment", now); err != nil {
		fail(w, 500, "核款失败")
		return
	}
	if _, err := tx.Exec(`DELETE FROM notifications WHERE user_id=? AND kind='admin_payment_pending' AND entity_id=?`, admin.ID, r.PathValue("id")); err != nil {
		fail(w, 500, "核款失败")
		return
	}
	if err := tx.Commit(); err != nil {
		fail(w, 500, "核款失败")
		return
	}
	send(w, 200, map[string]any{"status": v.Decision})
}
