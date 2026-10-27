package main

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"time"
)

const membershipCents = 990

type promoCode struct {
	ID         string
	Kind       string
	PayPercent int
	MaxUses    int
	UsedCount  int
	ValidFrom  int64
	ValidUntil int64
	Active     bool
}

type paymentOrder struct {
	ID             string `json:"id"`
	Channel        string `json:"channel"`
	PayeeName      string `json:"payee_name"`
	QRVersion      string `json:"qr_version"`
	OriginalCents  int    `json:"original_cents"`
	DiscountCents  int    `json:"discount_cents"`
	PayableCents   int    `json:"payable_cents"`
	CodeTail       string `json:"code_tail"`
	Status         string `json:"status"`
	TransactionRef string `json:"transaction_ref"`
	ReviewNote     string `json:"review_note"`
	CreatedAt      int64  `json:"created_at"`
	SubmittedAt    int64  `json:"submitted_at"`
	ReviewedAt     int64  `json:"reviewed_at"`
}

func migratePayments(db *sql.DB) error {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS promo_codes (
		id TEXT PRIMARY KEY, code_hash TEXT NOT NULL UNIQUE, code_tail TEXT NOT NULL, code_ciphertext TEXT NOT NULL DEFAULT '',
		kind TEXT NOT NULL CHECK(kind IN ('exchange','discount')),
		pay_percent INTEGER NOT NULL DEFAULT 100,
		max_uses INTEGER NOT NULL DEFAULT 0, used_count INTEGER NOT NULL DEFAULT 0,
		valid_from INTEGER NOT NULL, valid_until INTEGER NOT NULL,
		active INTEGER NOT NULL DEFAULT 1, created_at INTEGER NOT NULL,
		created_by TEXT NOT NULL REFERENCES users(id)
	);
	CREATE TABLE IF NOT EXISTS payment_qr (
		channel TEXT PRIMARY KEY CHECK(channel IN ('wechat','alipay')),
		image_path TEXT NOT NULL, payee_name TEXT NOT NULL DEFAULT '', version TEXT NOT NULL DEFAULT '', active INTEGER NOT NULL DEFAULT 1, updated_at INTEGER NOT NULL
	);
	CREATE TABLE IF NOT EXISTS payment_orders (
		id TEXT PRIMARY KEY, user_id TEXT NOT NULL REFERENCES users(id),
		channel TEXT NOT NULL CHECK(channel IN ('wechat','alipay')),
		payee_name TEXT NOT NULL DEFAULT '', qr_version TEXT NOT NULL DEFAULT '',
		original_cents INTEGER NOT NULL, discount_cents INTEGER NOT NULL,
		payable_cents INTEGER NOT NULL, promo_code_id TEXT REFERENCES promo_codes(id),
		code_tail TEXT NOT NULL DEFAULT '', status TEXT NOT NULL,
		transaction_ref TEXT NOT NULL DEFAULT '', review_note TEXT NOT NULL DEFAULT '',
		created_at INTEGER NOT NULL, submitted_at INTEGER NOT NULL DEFAULT 0,
		reviewed_at INTEGER NOT NULL DEFAULT 0, reviewer_id TEXT REFERENCES users(id)
	);
	CREATE UNIQUE INDEX IF NOT EXISTS payment_open_order ON payment_orders(user_id) WHERE status IN ('awaiting_payment','submitted','rejected');
	CREATE UNIQUE INDEX IF NOT EXISTS payment_approved_receipt ON payment_orders(channel,transaction_ref) WHERE status='approved' AND transaction_ref<>'';
	CREATE INDEX IF NOT EXISTS payment_submitted ON payment_orders(status,submitted_at);
	CREATE TABLE IF NOT EXISTS promo_uses (
		id INTEGER PRIMARY KEY AUTOINCREMENT, code_id TEXT NOT NULL REFERENCES promo_codes(id),
		user_id TEXT NOT NULL REFERENCES users(id), order_id TEXT REFERENCES payment_orders(id),
		kind TEXT NOT NULL, original_cents INTEGER NOT NULL, deducted_cents INTEGER NOT NULL,
		payable_cents INTEGER NOT NULL, created_at INTEGER NOT NULL,
		UNIQUE(code_id,user_id)
	);
	CREATE TABLE IF NOT EXISTS payment_events (
		id INTEGER PRIMARY KEY AUTOINCREMENT, order_id TEXT NOT NULL REFERENCES payment_orders(id),
		actor_id TEXT NOT NULL REFERENCES users(id), event TEXT NOT NULL,
		transaction_ref TEXT NOT NULL DEFAULT '', note TEXT NOT NULL DEFAULT '', created_at INTEGER NOT NULL
	);`)
	if err != nil {
		return err
	}
	for _, column := range []struct{ table, name, definition string }{
		{"payment_qr", "payee_name", "TEXT NOT NULL DEFAULT ''"},
		{"payment_qr", "version", "TEXT NOT NULL DEFAULT ''"},
		{"payment_orders", "payee_name", "TEXT NOT NULL DEFAULT ''"},
		{"payment_orders", "qr_version", "TEXT NOT NULL DEFAULT ''"},
		{"promo_codes", "code_ciphertext", "TEXT NOT NULL DEFAULT ''"},
	} {
		if err := ensureColumn(db, column.table, column.name, column.definition); err != nil {
			return err
		}
	}
	return nil
}

func ensureColumn(db *sql.DB, table, column, definition string) error {
	rows, err := db.Query(fmt.Sprintf(`PRAGMA table_info(%s)`, table))
	if err != nil {
		return err
	}
	found := false
	for rows.Next() {
		var cid, notnull, pk int
		var name, kind string
		var defaultValue sql.NullString
		if err := rows.Scan(&cid, &name, &kind, &notnull, &defaultValue, &pk); err != nil {
			rows.Close()
			return err
		}
		if name == column {
			found = true
		}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	if err := rows.Close(); err != nil {
		return err
	}
	if !found {
		_, err = db.Exec(fmt.Sprintf(`ALTER TABLE %s ADD COLUMN %s %s`, table, column, definition))
	}
	return err
}

func (a *App) registerPaymentRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/payment/create", func(w http.ResponseWriter, r *http.Request) {
		fail(w, 410, "模拟支付已停用，请刷新页面使用收款码流程")
	})
	mux.HandleFunc("GET /api/payment/status", a.paymentStatus)
	mux.HandleFunc("POST /api/payment/quote", a.paymentQuote)
	mux.HandleFunc("POST /api/payment/redeem", a.redeemCode)
	mux.HandleFunc("POST /api/payment/orders", a.createPaymentOrder)
	mux.HandleFunc("POST /api/payment/orders/{id}/submit", a.submitPaymentOrder)
	mux.HandleFunc("POST /api/payment/orders/{id}/cancel", a.cancelPaymentOrder)
	mux.HandleFunc("GET /api/payment/qr/{channel}", a.paymentQR)
	mux.HandleFunc("GET /api/admin/payment/orders", a.adminPaymentOrders)
	mux.HandleFunc("POST /api/admin/payment/orders/{id}", a.adminPaymentDecision)
	mux.HandleFunc("GET /api/admin/payment/qr", a.adminPaymentQRList)
	mux.HandleFunc("POST /api/admin/payment/qr/{channel}", a.adminUploadPaymentQR)
	mux.HandleFunc("POST /api/admin/payment/qr/{channel}/status", a.adminPaymentQRStatus)
	mux.HandleFunc("POST /api/admin/payment/qr/{channel}/settings", a.adminPaymentQRSettings)
	mux.HandleFunc("GET /api/admin/promo-codes", a.adminPromoCodes)
	mux.HandleFunc("POST /api/admin/promo-codes", a.adminCreatePromoCode)
	mux.HandleFunc("GET /api/admin/promo-codes/{id}/code", a.adminPromoCodeValue)
	mux.HandleFunc("POST /api/admin/promo-codes/{id}/status", a.adminPromoCodeStatus)
	mux.HandleFunc("GET /api/admin/promo-uses", a.adminPromoUses)
}

func normalizeCode(raw string) string { return strings.ToUpper(strings.TrimSpace(raw)) }

func payableForPercent(percent int) int { return (membershipCents*percent + 50) / 100 }

func (a *App) lookupCode(q interface{ QueryRow(string, ...any) *sql.Row }, raw string) (promoCode, error) {
	var code promoCode
	var active int
	err := q.QueryRow(`SELECT id,kind,pay_percent,max_uses,used_count,valid_from,valid_until,active FROM promo_codes WHERE code_hash=?`, digest(normalizeCode(raw))).Scan(&code.ID, &code.Kind, &code.PayPercent, &code.MaxUses, &code.UsedCount, &code.ValidFrom, &code.ValidUntil, &active)
	code.Active = active != 0
	return code, err
}

func validCodeAt(code promoCode, now int64) bool {
	return code.Active && now >= code.ValidFrom && now <= code.ValidUntil && (code.Kind != "exchange" || code.UsedCount < code.MaxUses)
}

func (a *App) codeUsedBy(q interface{ QueryRow(string, ...any) *sql.Row }, codeID, userID string) (bool, error) {
	var used int
	err := q.QueryRow(`SELECT EXISTS(SELECT 1 FROM promo_uses WHERE code_id=? AND user_id=?)`, codeID, userID).Scan(&used)
	return used != 0, err
}

func (a *App) paymentQuote(w http.ResponseWriter, r *http.Request) {
	u, ok := a.mustAuth(w, r)
	if !ok {
		return
	}
	if u.Paid {
		fail(w, 409, "会费已完成")
		return
	}
	var input struct {
		Code string `json:"code"`
	}
	if !decode(w, r, &input) {
		return
	}
	if len(normalizeCode(input.Code)) > 80 || normalizeCode(input.Code) == "" {
		fail(w, 400, "请输入有效代码")
		return
	}
	code, err := a.lookupCode(a.db, input.Code)
	if err != nil || !validCodeAt(code, time.Now().Unix()) {
		fail(w, 400, "代码无效、已停用或不在使用时间内")
		return
	}
	used, err := a.codeUsedBy(a.db, code.ID, u.ID)
	if err != nil {
		fail(w, 500, "验证失败")
		return
	}
	if used {
		fail(w, 409, "该代码已使用过")
		return
	}
	payable := 0
	if code.Kind == "discount" {
		payable = payableForPercent(code.PayPercent)
	}
	send(w, 200, map[string]any{"kind": code.Kind, "pay_percent": code.PayPercent, "original_cents": membershipCents, "discount_cents": membershipCents - payable, "payable_cents": payable})
}

func (a *App) paymentStatus(w http.ResponseWriter, r *http.Request) {
	u, ok := a.mustAuth(w, r)
	if !ok {
		return
	}
	channels := []map[string]string{}
	rows, err := a.db.Query(`SELECT channel,payee_name,version,image_path FROM payment_qr WHERE active=1 ORDER BY channel`)
	if err != nil {
		fail(w, 500, "获取收款方式失败")
		return
	}
	for rows.Next() {
		var channel, payeeName, version, imagePath string
		if err := rows.Scan(&channel, &payeeName, &version, &imagePath); err != nil {
			rows.Close()
			fail(w, 500, "获取收款方式失败")
			return
		}
		if version == "" {
			version = filepath.Base(imagePath)
		}
		channels = append(channels, map[string]string{"channel": channel, "qr_url": "/api/payment/qr/" + channel, "payee_name": payeeName, "version": version})
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		fail(w, 500, "获取收款方式失败")
		return
	}
	rows.Close()
	var order *paymentOrder
	var current paymentOrder
	err = a.db.QueryRow(`SELECT id,channel,payee_name,qr_version,original_cents,discount_cents,payable_cents,code_tail,status,transaction_ref,review_note,created_at,submitted_at,reviewed_at FROM payment_orders WHERE user_id=? ORDER BY created_at DESC,id DESC LIMIT 1`, u.ID).Scan(&current.ID, &current.Channel, &current.PayeeName, &current.QRVersion, &current.OriginalCents, &current.DiscountCents, &current.PayableCents, &current.CodeTail, &current.Status, &current.TransactionRef, &current.ReviewNote, &current.CreatedAt, &current.SubmittedAt, &current.ReviewedAt)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		fail(w, 500, "获取订单失败")
		return
	}
	if err == nil {
		order = &current
	}
	uses := []map[string]any{}
	rows, err = a.db.Query(`SELECT p.kind,p.code_tail,v.original_cents,v.deducted_cents,v.payable_cents,v.created_at FROM promo_uses v JOIN promo_codes p ON p.id=v.code_id WHERE v.user_id=? ORDER BY v.created_at DESC,v.id DESC`, u.ID)
	if err != nil {
		fail(w, 500, "获取使用记录失败")
		return
	}
	for rows.Next() {
		var kind, tail string
		var original, deducted, payable int
		var created int64
		if err := rows.Scan(&kind, &tail, &original, &deducted, &payable, &created); err != nil {
			rows.Close()
			fail(w, 500, "获取使用记录失败")
			return
		}
		uses = append(uses, map[string]any{"kind": kind, "code_tail": tail, "original_cents": original, "discount_cents": deducted, "payable_cents": payable, "created_at": created})
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		fail(w, 500, "获取使用记录失败")
		return
	}
	rows.Close()
	send(w, 200, map[string]any{"paid": u.Paid, "original_cents": membershipCents, "channels": channels, "order": order, "uses": uses})
}

func (a *App) redeemCode(w http.ResponseWriter, r *http.Request) {
	u, ok := a.mustAuth(w, r)
	if !ok {
		return
	}
	var input struct {
		Code string `json:"code"`
	}
	if !decode(w, r, &input) {
		return
	}
	if len(normalizeCode(input.Code)) > 80 || normalizeCode(input.Code) == "" {
		fail(w, 400, "请输入有效兑换码")
		return
	}
	tx, err := a.db.Begin()
	if err != nil {
		fail(w, 500, "兑换失败")
		return
	}
	defer tx.Rollback()
	var paid, submitted int
	if err := tx.QueryRow(`SELECT paid FROM users WHERE id=?`, u.ID).Scan(&paid); err != nil {
		fail(w, 500, "兑换失败")
		return
	}
	if paid != 0 {
		fail(w, 409, "会费已完成")
		return
	}
	if err := tx.QueryRow(`SELECT EXISTS(SELECT 1 FROM payment_orders WHERE user_id=? AND status='submitted')`, u.ID).Scan(&submitted); err != nil {
		fail(w, 500, "兑换失败")
		return
	}
	if submitted != 0 {
		fail(w, 409, "已有待核款订单，请先完成核款")
		return
	}
	code, err := a.lookupCode(tx, input.Code)
	if err != nil || code.Kind != "exchange" || !validCodeAt(code, time.Now().Unix()) {
		fail(w, 400, "兑换码无效、已停用或次数已用完")
		return
	}
	used, err := a.codeUsedBy(tx, code.ID, u.ID)
	if err != nil || used {
		fail(w, 409, "该兑换码已使用过")
		return
	}
	result, err := tx.Exec(`UPDATE promo_codes SET used_count=used_count+1 WHERE id=? AND active=1 AND used_count<max_uses AND valid_from<=? AND valid_until>=?`, code.ID, time.Now().Unix(), time.Now().Unix())
	if err != nil {
		fail(w, 500, "兑换失败")
		return
	}
	if n, _ := result.RowsAffected(); n != 1 {
		fail(w, 409, "兑换码次数已用完")
		return
	}
	if _, err := tx.Exec(`INSERT INTO promo_uses(code_id,user_id,kind,original_cents,deducted_cents,payable_cents,created_at) VALUES(?,?,'exchange',?,?,0,?)`, code.ID, u.ID, membershipCents, membershipCents, time.Now().Unix()); err != nil {
		fail(w, 409, "该兑换码已使用过")
		return
	}
	if _, err := tx.Exec(`UPDATE payment_orders SET status='cancelled' WHERE user_id=? AND status IN ('awaiting_payment','rejected')`, u.ID); err != nil {
		fail(w, 500, "兑换失败")
		return
	}
	result, err = tx.Exec(`UPDATE users SET paid=1 WHERE id=? AND paid=0`, u.ID)
	if err != nil {
		fail(w, 500, "兑换失败")
		return
	}
	if n, _ := result.RowsAffected(); n != 1 {
		fail(w, 409, "会费已完成")
		return
	}
	if err := addNotification(tx, u.ID, "membership_redeemed", code.ID, "会费已完成", "兑换码使用成功，会费步骤已完成。", "payment", time.Now().Unix()); err != nil {
		fail(w, 500, "兑换失败")
		return
	}
	if err := tx.Commit(); err != nil {
		fail(w, 500, "兑换失败")
		return
	}
	send(w, 200, map[string]any{"paid": true, "method": "exchange"})
}

func (a *App) createPaymentOrder(w http.ResponseWriter, r *http.Request) {
	u, ok := a.mustAuth(w, r)
	if !ok {
		return
	}
	var input struct {
		Channel string `json:"channel"`
		Code    string `json:"code"`
	}
	if !decode(w, r, &input) {
		return
	}
	if input.Channel != "wechat" && input.Channel != "alipay" {
		fail(w, 400, "请选择收款方式")
		return
	}
	if len(normalizeCode(input.Code)) > 80 {
		fail(w, 400, "折扣码格式不正确")
		return
	}
	tx, err := a.db.Begin()
	if err != nil {
		fail(w, 500, "创建订单失败")
		return
	}
	defer tx.Rollback()
	var paid, enabled int
	if err := tx.QueryRow(`SELECT paid FROM users WHERE id=?`, u.ID).Scan(&paid); err != nil {
		fail(w, 500, "创建订单失败")
		return
	}
	if paid != 0 {
		fail(w, 409, "会费已完成")
		return
	}
	if err := tx.QueryRow(`SELECT EXISTS(SELECT 1 FROM payment_orders WHERE user_id=? AND status IN ('awaiting_payment','submitted','rejected'))`, u.ID).Scan(&enabled); err != nil {
		fail(w, 500, "创建订单失败")
		return
	}
	if enabled != 0 {
		fail(w, 409, "已有未完成订单，请先处理或取消")
		return
	}
	var qrPath, payeeName, qrVersion string
	err = tx.QueryRow(`SELECT image_path,payee_name,version FROM payment_qr WHERE channel=? AND active=1`, input.Channel).Scan(&qrPath, &payeeName, &qrVersion)
	if errors.Is(err, sql.ErrNoRows) {
		fail(w, 409, "该收款方式暂不可用")
		return
	}
	if err != nil {
		fail(w, 500, "创建订单失败")
		return
	}
	if payeeName == "" {
		fail(w, 409, "该收款方式尚未设置收款方名称")
		return
	}
	if qrVersion == "" {
		qrVersion = filepath.Base(qrPath)
	}
	payable, discount, codeID, codeTail := membershipCents, 0, "", ""
	if normalizeCode(input.Code) != "" {
		code, err := a.lookupCode(tx, input.Code)
		if err != nil || code.Kind != "discount" || !validCodeAt(code, time.Now().Unix()) {
			fail(w, 400, "折扣码无效或已过期")
			return
		}
		used, err := a.codeUsedBy(tx, code.ID, u.ID)
		if err != nil {
			fail(w, 500, "创建订单失败")
			return
		}
		if used {
			fail(w, 409, "该折扣码已使用过")
			return
		}
		payable = payableForPercent(code.PayPercent)
		discount = membershipCents - payable
		codeID = code.ID
		codeTail = normalizeCode(input.Code)
		if len(codeTail) > 4 {
			codeTail = codeTail[len(codeTail)-4:]
		}
	}
	id := "MM" + strings.ToUpper(randomHex(12))
	now := time.Now().Unix()
	var promoID any
	if codeID != "" {
		promoID = codeID
	}
	if _, err := tx.Exec(`INSERT INTO payment_orders(id,user_id,channel,payee_name,qr_version,original_cents,discount_cents,payable_cents,promo_code_id,code_tail,status,created_at) VALUES(?,?,?,?,?,?,?,?,?,?, 'awaiting_payment',?)`, id, u.ID, input.Channel, payeeName, qrVersion, membershipCents, discount, payable, promoID, codeTail, now); err != nil {
		fail(w, 409, "已有未完成订单，请刷新页面")
		return
	}
	if _, err := tx.Exec(`INSERT INTO payment_events(order_id,actor_id,event,created_at) VALUES(?,?,'created',?)`, id, u.ID, now); err != nil {
		fail(w, 500, "创建订单失败")
		return
	}
	if err := tx.Commit(); err != nil {
		fail(w, 500, "创建订单失败")
		return
	}
	send(w, 200, map[string]any{"order_id": id, "payable_cents": payable, "discount_cents": discount, "status": "awaiting_payment"})
}

func validTransactionRef(ref string) bool {
	if len(ref) < 6 || len(ref) > 64 {
		return false
	}
	for _, char := range ref {
		if !(char >= '0' && char <= '9' || char >= 'A' && char <= 'Z' || char >= 'a' && char <= 'z' || char == '-') {
			return false
		}
	}
	return true
}

func (a *App) submitPaymentOrder(w http.ResponseWriter, r *http.Request) {
	u, ok := a.mustAuth(w, r)
	if !ok {
		return
	}
	var input struct {
		TransactionRef string `json:"transaction_ref"`
	}
	if !decode(w, r, &input) {
		return
	}
	ref := strings.TrimSpace(input.TransactionRef)
	if !validTransactionRef(ref) {
		fail(w, 400, "请输入 6 至 64 位交易单号，仅可含字母、数字或连字符")
		return
	}
	tx, err := a.db.Begin()
	if err != nil {
		fail(w, 500, "提交失败")
		return
	}
	defer tx.Rollback()
	var status string
	if err := tx.QueryRow(`SELECT status FROM payment_orders WHERE id=? AND user_id=?`, r.PathValue("id"), u.ID).Scan(&status); err != nil {
		fail(w, 404, "订单不存在")
		return
	}
	if status != "awaiting_payment" && status != "rejected" {
		fail(w, 409, "该订单不能重复提交")
		return
	}
	var paid int
	if err := tx.QueryRow(`SELECT paid FROM users WHERE id=?`, u.ID).Scan(&paid); err != nil || paid != 0 {
		fail(w, 409, "会费已完成")
		return
	}
	now := time.Now().Unix()
	if _, err := tx.Exec(`UPDATE payment_orders SET status='submitted',transaction_ref=?,submitted_at=?,review_note='' WHERE id=?`, ref, now, r.PathValue("id")); err != nil {
		fail(w, 500, "提交失败")
		return
	}
	if _, err := tx.Exec(`INSERT INTO payment_events(order_id,actor_id,event,transaction_ref,created_at) VALUES(?,?,'submitted',?,?)`, r.PathValue("id"), u.ID, ref, now); err != nil {
		fail(w, 500, "提交失败")
		return
	}
	if err := a.addAdminNotification(tx, "admin_payment_pending", r.PathValue("id"), "会费待核款", u.Nickname+"提交了付款信息，等待核对。", "admin:payments", now); err != nil {
		fail(w, 500, "提交失败")
		return
	}
	if err := tx.Commit(); err != nil {
		fail(w, 500, "提交失败")
		return
	}
	send(w, 200, map[string]any{"status": "submitted"})
}

func (a *App) cancelPaymentOrder(w http.ResponseWriter, r *http.Request) {
	u, ok := a.mustAuth(w, r)
	if !ok {
		return
	}
	tx, err := a.db.Begin()
	if err != nil {
		fail(w, 500, "取消失败")
		return
	}
	defer tx.Rollback()
	result, err := tx.Exec(`UPDATE payment_orders SET status='cancelled' WHERE id=? AND user_id=? AND status IN ('awaiting_payment','rejected')`, r.PathValue("id"), u.ID)
	if err != nil {
		fail(w, 500, "取消失败")
		return
	}
	if n, _ := result.RowsAffected(); n != 1 {
		fail(w, 409, "订单待核款期间不能取消")
		return
	}
	if _, err := tx.Exec(`INSERT INTO payment_events(order_id,actor_id,event,created_at) VALUES(?,?,'cancelled',?)`, r.PathValue("id"), u.ID, time.Now().Unix()); err != nil {
		fail(w, 500, "取消失败")
		return
	}
	if err := tx.Commit(); err != nil {
		fail(w, 500, "取消失败")
		return
	}
	send(w, 200, map[string]any{"status": "cancelled"})
}

func paymentLabel(channel string) string {
	if channel == "wechat" {
		return "微信"
	}
	if channel == "alipay" {
		return "支付宝"
	}
	return channel
}

func validatePromoCreate(kind string, percent, maxUses int, start, end int64) error {
	if kind != "exchange" && kind != "discount" {
		return fmt.Errorf("请选择代码类型")
	}
	if kind == "exchange" && (maxUses < 1 || maxUses > 100000) {
		return fmt.Errorf("兑换码使用次数须在 1 至 100000 之间")
	}
	if kind == "discount" && (percent < 1 || percent > 99) {
		return fmt.Errorf("支付比例须在 1%% 至 99%% 之间")
	}
	if start < 0 || end <= start || end-start > 366*24*3600 {
		return fmt.Errorf("有效时间范围须在一年以内")
	}
	return nil
}
