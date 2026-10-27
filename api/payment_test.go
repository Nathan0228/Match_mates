package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func paymentTestApp(t *testing.T) *App {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	db.SetMaxOpenConns(1)
	if err := migrate(db); err != nil {
		t.Fatal(err)
	}
	a := &App{db: db, adminEmail: "admin@mails.ucas.ac.cn", uploadDir: t.TempDir(), dev: true}
	for _, id := range []string{"admin", "alice", "bob"} {
		_, err := db.Exec(`INSERT INTO users(id,email,password_hash,nickname,email_verified,created_at) VALUES(?,?,?,?,1,?)`, id, id+"@mails.ucas.ac.cn", "hash", id, time.Now().Unix())
		if err != nil {
			t.Fatal(err)
		}
		_, err = db.Exec(`INSERT INTO sessions(token_hash,user_id,expires_at) VALUES(?,?,?)`, digest(id+"-token"), id, time.Now().Add(time.Hour).Unix())
		if err != nil {
			t.Fatal(err)
		}
	}
	return a
}

func paymentCall(handler http.HandlerFunc, method, path, user, id, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if user != "" {
		req.AddCookie(&http.Cookie{Name: "mm_session", Value: user + "-token"})
	}
	if id != "" {
		req.SetPathValue("id", id)
	}
	w := httptest.NewRecorder()
	handler(w, req)
	return w
}

func issueCode(t *testing.T, a *App, kind string, percent, maxUses int) string {
	t.Helper()
	body, err := json.Marshal(map[string]any{"kind": kind, "pay_percent": percent, "max_uses": maxUses, "valid_from": time.Now().Add(-time.Minute).Unix(), "valid_until": time.Now().Add(time.Hour).Unix()})
	if err != nil {
		t.Fatal(err)
	}
	w := paymentCall(a.adminCreatePromoCode, "POST", "/api/admin/promo-codes", "admin", "", string(body))
	if w.Code != 201 {
		t.Fatalf("issue code: %d %s", w.Code, w.Body.String())
	}
	var result struct {
		Code string `json:"code"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	return result.Code
}

func TestExchangeCodeUsageAndLegacyPaymentClosed(t *testing.T) {
	a := paymentTestApp(t)
	code := issueCode(t, a, "exchange", 0, 1)
	if got := paymentCall(a.adminCreatePromoCode, "POST", "/api/admin/promo-codes", "alice", "", `{}`).Code; got != 403 {
		t.Fatalf("non-admin code create = %d", got)
	}
	quote := paymentCall(a.paymentQuote, "POST", "/api/payment/quote", "alice", "", `{"code":"`+code+`"}`)
	if quote.Code != 200 || !strings.Contains(quote.Body.String(), `"payable_cents":0`) {
		t.Fatalf("exchange quote: %d %s", quote.Code, quote.Body.String())
	}
	redeem := paymentCall(a.redeemCode, "POST", "/api/payment/redeem", "alice", "", `{"code":"`+code+`"}`)
	if redeem.Code != 200 {
		t.Fatalf("redeem: %d %s", redeem.Code, redeem.Body.String())
	}
	var paid, uses int
	if err := a.db.QueryRow(`SELECT paid FROM users WHERE id='alice'`).Scan(&paid); err != nil || paid != 1 {
		t.Fatalf("alice paid = %d %v", paid, err)
	}
	if err := a.db.QueryRow(`SELECT used_count FROM promo_codes LIMIT 1`).Scan(&uses); err != nil || uses != 1 {
		t.Fatalf("code uses = %d %v", uses, err)
	}
	if got := paymentCall(a.redeemCode, "POST", "/api/payment/redeem", "bob", "", `{"code":"`+code+`"}`).Code; got != 400 {
		t.Fatalf("exhausted code status = %d", got)
	}
	if err := a.db.QueryRow(`SELECT paid FROM users WHERE id='bob'`).Scan(&paid); err != nil || paid != 0 {
		t.Fatalf("bob paid = %d %v", paid, err)
	}
	status := paymentCall(a.paymentStatus, "GET", "/api/payment/status", "alice", "", "")
	if status.Code != 200 || !strings.Contains(status.Body.String(), `"kind":"exchange"`) {
		t.Fatalf("usage history: %d %s", status.Code, status.Body.String())
	}
	mux := http.NewServeMux()
	a.registerPaymentRoutes(mux)
	legacy := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/payment/create", nil)
	req.AddCookie(&http.Cookie{Name: "mm_session", Value: "bob-token"})
	mux.ServeHTTP(legacy, req)
	if legacy.Code != 410 {
		t.Fatalf("legacy payment status = %d", legacy.Code)
	}
}

func TestDiscountOrderRequiresManualApprovalAndLocksQuote(t *testing.T) {
	a := paymentTestApp(t)
	_, err := a.db.Exec(`INSERT INTO payment_qr(channel,image_path,payee_name,active,updated_at) VALUES('wechat','test.png','测试收款方',1,?)`, time.Now().Unix())
	if err != nil {
		t.Fatal(err)
	}
	code := issueCode(t, a, "discount", 80, 0)
	quote := paymentCall(a.paymentQuote, "POST", "/api/payment/quote", "alice", "", `{"code":"`+code+`"}`)
	if quote.Code != 200 || !strings.Contains(quote.Body.String(), `"payable_cents":792`) || !strings.Contains(quote.Body.String(), `"discount_cents":198`) {
		t.Fatalf("discount quote: %d %s", quote.Code, quote.Body.String())
	}
	created := paymentCall(a.createPaymentOrder, "POST", "/api/payment/orders", "alice", "", `{"channel":"wechat","code":"`+code+`"}`)
	if created.Code != 200 {
		t.Fatalf("create order: %d %s", created.Code, created.Body.String())
	}
	var order struct {
		ID      string `json:"order_id"`
		Payable int    `json:"payable_cents"`
	}
	if err := json.Unmarshal(created.Body.Bytes(), &order); err != nil || order.Payable != 792 {
		t.Fatalf("order: %v %s", err, created.Body.String())
	}
	if got := paymentCall(a.createPaymentOrder, "POST", "/api/payment/orders", "alice", "", `{"channel":"wechat"}`).Code; got != 409 {
		t.Fatalf("second open order = %d", got)
	}
	var paid int
	if err := a.db.QueryRow(`SELECT paid FROM users WHERE id='alice'`).Scan(&paid); err != nil || paid != 0 {
		t.Fatalf("paid before transfer = %d %v", paid, err)
	}
	submitted := paymentCall(a.submitPaymentOrder, "POST", "/api/payment/orders/"+order.ID+"/submit", "alice", order.ID, `{"transaction_ref":"WX123456789"}`)
	if submitted.Code != 200 {
		t.Fatalf("submit: %d %s", submitted.Code, submitted.Body.String())
	}
	if got := paymentCall(a.cancelPaymentOrder, "POST", "/api/payment/orders/"+order.ID+"/cancel", "alice", order.ID, "").Code; got != 409 {
		t.Fatalf("cancel submitted = %d", got)
	}
	if got := paymentCall(a.adminPaymentDecision, "POST", "/api/admin/payment/orders/"+order.ID, "alice", order.ID, `{"decision":"approved","verified":true}`).Code; got != 403 {
		t.Fatalf("non-admin approval = %d", got)
	}
	if got := paymentCall(a.adminPaymentDecision, "POST", "/api/admin/payment/orders/"+order.ID, "admin", order.ID, `{"decision":"approved"}`).Code; got != 400 {
		t.Fatalf("approval without verification = %d", got)
	}
	var codeID string
	if err := a.db.QueryRow(`SELECT id FROM promo_codes WHERE code_hash=?`, digest(code)).Scan(&codeID); err != nil {
		t.Fatal(err)
	}
	if got := paymentCall(a.adminPromoCodeStatus, "POST", "/api/admin/promo-codes/"+codeID+"/status", "admin", codeID, `{"active":false}`).Code; got != 200 {
		t.Fatalf("disable discount = %d", got)
	}
	if got := paymentCall(a.paymentQuote, "POST", "/api/payment/quote", "bob", "", `{"code":"`+code+`"}`).Code; got != 400 {
		t.Fatalf("disabled quote = %d", got)
	}
	approved := paymentCall(a.adminPaymentDecision, "POST", "/api/admin/payment/orders/"+order.ID, "admin", order.ID, `{"decision":"approved","verified":true}`)
	if approved.Code != 200 {
		t.Fatalf("approve locked quote: %d %s", approved.Code, approved.Body.String())
	}
	if err := a.db.QueryRow(`SELECT paid FROM users WHERE id='alice'`).Scan(&paid); err != nil || paid != 1 {
		t.Fatalf("paid after approval = %d %v", paid, err)
	}
	var uses, discount int
	if err := a.db.QueryRow(`SELECT used_count FROM promo_codes WHERE id=?`, codeID).Scan(&uses); err != nil || uses != 1 {
		t.Fatalf("discount uses = %d %v", uses, err)
	}
	if err := a.db.QueryRow(`SELECT deducted_cents FROM promo_uses WHERE user_id='alice'`).Scan(&discount); err != nil || discount != 198 {
		t.Fatalf("discount record = %d %v", discount, err)
	}
	if got := paymentCall(a.adminPaymentDecision, "POST", "/api/admin/payment/orders/"+order.ID, "admin", order.ID, `{"decision":"approved","verified":true}`).Code; got != 409 {
		t.Fatalf("repeat approval = %d", got)
	}
	if got := paymentCall(a.createPaymentOrder, "POST", "/api/payment/orders", "bob", "", `{"channel":"wechat"}`).Code; got != 200 {
		t.Fatalf("bob order = %d", got)
	}
	var bobID string
	if err := a.db.QueryRow(`SELECT id FROM payment_orders WHERE user_id='bob'`).Scan(&bobID); err != nil {
		t.Fatal(err)
	}
	if got := paymentCall(a.submitPaymentOrder, "POST", "/api/payment/orders/"+bobID+"/submit", "bob", bobID, `{"transaction_ref":"WX123456789"}`).Code; got != 200 {
		t.Fatalf("bob submit = %d", got)
	}
	if got := paymentCall(a.adminPaymentDecision, "POST", "/api/admin/payment/orders/"+bobID, "admin", bobID, `{"decision":"approved","verified":true}`).Code; got != 409 {
		t.Fatalf("duplicate receipt approval = %d", got)
	}
	if err := a.db.QueryRow(`SELECT paid FROM users WHERE id='bob'`).Scan(&paid); err != nil || paid != 0 {
		t.Fatalf("bob wrongly paid = %d %v", paid, err)
	}
}

func TestRejectedPaymentCanBeCorrected(t *testing.T) {
	a := paymentTestApp(t)
	_, err := a.db.Exec(`INSERT INTO payment_qr(channel,image_path,payee_name,active,updated_at) VALUES('alipay','test.png','测试收款方',1,?)`, time.Now().Unix())
	if err != nil {
		t.Fatal(err)
	}
	created := paymentCall(a.createPaymentOrder, "POST", "/api/payment/orders", "alice", "", `{"channel":"alipay"}`)
	if created.Code != 200 {
		t.Fatalf("order: %d %s", created.Code, created.Body.String())
	}
	var result struct {
		ID string `json:"order_id"`
	}
	_ = json.Unmarshal(created.Body.Bytes(), &result)
	if got := paymentCall(a.submitPaymentOrder, "POST", "/api/payment/orders/"+result.ID+"/submit", "alice", result.ID, `{"transaction_ref":"ALI111111"}`).Code; got != 200 {
		t.Fatalf("first submit = %d", got)
	}
	if got := paymentCall(a.adminPaymentDecision, "POST", "/api/admin/payment/orders/"+result.ID, "admin", result.ID, `{"decision":"rejected","note":"未查到对应交易"}`).Code; got != 200 {
		t.Fatalf("reject = %d", got)
	}
	var paid int
	_ = a.db.QueryRow(`SELECT paid FROM users WHERE id='alice'`).Scan(&paid)
	if paid != 0 {
		t.Fatal("rejected payment unlocked membership")
	}
	if got := paymentCall(a.submitPaymentOrder, "POST", "/api/payment/orders/"+result.ID+"/submit", "alice", result.ID, `{"transaction_ref":"ALI222222"}`).Code; got != 200 {
		t.Fatalf("resubmit = %d", got)
	}
	if got := paymentCall(a.adminPaymentDecision, "POST", "/api/admin/payment/orders/"+result.ID, "admin", result.ID, `{"decision":"approved","verified":true}`).Code; got != 200 {
		t.Fatalf("approve corrected = %d", got)
	}
	_ = a.db.QueryRow(`SELECT paid FROM users WHERE id='alice'`).Scan(&paid)
	if paid != 1 {
		t.Fatal("corrected payment not unlocked")
	}
	var events int
	_ = a.db.QueryRow(`SELECT COUNT(*) FROM payment_events WHERE order_id=?`, result.ID).Scan(&events)
	if events != 5 {
		t.Fatalf("payment audit events = %d", events)
	}
}

func TestAdminPaymentQRUploadAndStatus(t *testing.T) {
	a := paymentTestApp(t)
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	img.Set(0, 0, color.RGBA{R: 255, A: 255})
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, img); err != nil {
		t.Fatal(err)
	}
	makeUpload := func(user string) *httptest.ResponseRecorder {
		var body bytes.Buffer
		writer := multipart.NewWriter(&body)
		part, err := writer.CreateFormFile("qr", "qr.png")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := part.Write(encoded.Bytes()); err != nil {
			t.Fatal(err)
		}
		if err := writer.WriteField("payee_name", "见山收款账户"); err != nil {
			t.Fatal(err)
		}
		if err := writer.Close(); err != nil {
			t.Fatal(err)
		}
		req := httptest.NewRequest("POST", "/api/admin/payment/qr/wechat", &body)
		req.SetPathValue("channel", "wechat")
		req.Header.Set("Content-Type", writer.FormDataContentType())
		req.AddCookie(&http.Cookie{Name: "mm_session", Value: user + "-token"})
		response := httptest.NewRecorder()
		a.adminUploadPaymentQR(response, req)
		return response
	}
	if got := makeUpload("alice").Code; got != 403 {
		t.Fatalf("non-admin QR upload = %d", got)
	}
	if got := makeUpload("admin").Code; got != 200 {
		t.Fatalf("admin QR upload = %d", got)
	}
	view := func(user string) int {
		req := httptest.NewRequest("GET", "/api/payment/qr/wechat", nil)
		req.SetPathValue("channel", "wechat")
		req.AddCookie(&http.Cookie{Name: "mm_session", Value: user + "-token"})
		response := httptest.NewRecorder()
		a.paymentQR(response, req)
		return response.Code
	}
	if got := view("alice"); got != 200 {
		t.Fatalf("user QR view = %d", got)
	}
	status := paymentCall(a.paymentStatus, "GET", "/api/payment/status", "alice", "", "")
	if status.Code != 200 || !strings.Contains(status.Body.String(), "见山收款账户") {
		t.Fatalf("payee name missing from payment status: %d %s", status.Code, status.Body.String())
	}
	created := paymentCall(a.createPaymentOrder, "POST", "/api/payment/orders", "alice", "", `{"channel":"wechat"}`)
	if created.Code != 200 {
		t.Fatalf("create order with QR: %d %s", created.Code, created.Body.String())
	}
	var oldPayee, oldVersion string
	if err := a.db.QueryRow(`SELECT payee_name,qr_version FROM payment_orders WHERE user_id='alice'`).Scan(&oldPayee, &oldVersion); err != nil {
		t.Fatal(err)
	}
	if oldPayee != "见山收款账户" || oldVersion == "" {
		t.Fatalf("missing order QR snapshot: %q %q", oldPayee, oldVersion)
	}
	settingsRequest := httptest.NewRequest("POST", "/api/admin/payment/qr/wechat/settings", strings.NewReader(`{"payee_name":"见山新收款账户"}`))
	settingsRequest.SetPathValue("channel", "wechat")
	settingsRequest.AddCookie(&http.Cookie{Name: "mm_session", Value: "admin-token"})
	settings := httptest.NewRecorder()
	a.adminPaymentQRSettings(settings, settingsRequest)
	if settings.Code != 200 {
		t.Fatalf("update payee: %d %s", settings.Code, settings.Body.String())
	}
	var newVersion string
	if err := a.db.QueryRow(`SELECT version FROM payment_qr WHERE channel='wechat'`).Scan(&newVersion); err != nil || newVersion == oldVersion {
		t.Fatalf("QR version did not change: %q %q %v", oldVersion, newVersion, err)
	}
	if err := a.db.QueryRow(`SELECT payee_name FROM payment_orders WHERE user_id='alice'`).Scan(&oldPayee); err != nil || oldPayee != "见山收款账户" {
		t.Fatalf("order payee changed: %q %v", oldPayee, err)
	}
	request := httptest.NewRequest("POST", "/api/admin/payment/qr/wechat/status", strings.NewReader(`{"active":false}`))
	request.SetPathValue("channel", "wechat")
	request.AddCookie(&http.Cookie{Name: "mm_session", Value: "admin-token"})
	response := httptest.NewRecorder()
	a.adminPaymentQRStatus(response, request)
	if response.Code != 200 {
		t.Fatalf("disable QR = %d %s", response.Code, response.Body.String())
	}
	if got := view("alice"); got != 404 {
		t.Fatalf("disabled QR user view = %d", got)
	}
	if got := view("admin"); got != 200 {
		t.Fatalf("disabled QR admin view = %d", got)
	}
}
