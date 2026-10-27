package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestAdminCanCopyNewPromoCodeWithoutExposingItInList(t *testing.T) {
	a := paymentTestApp(t)
	code := issueCode(t, a, "exchange", 0, 2)
	var id string
	if err := a.db.QueryRow(`SELECT id FROM promo_codes WHERE code_hash=?`, digest(code)).Scan(&id); err != nil {
		t.Fatal(err)
	}
	list := paymentCall(a.adminPromoCodes, "GET", "/api/admin/promo-codes", "admin", "", "")
	if list.Code != 200 || strings.Contains(list.Body.String(), code) || !strings.Contains(list.Body.String(), `"copy_available":true`) {
		t.Fatalf("admin list: %d %s", list.Code, list.Body.String())
	}
	if got := paymentCall(a.adminPromoCodeValue, "GET", "/api/admin/promo-codes/"+id+"/code", "alice", id, ""); got.Code != 403 {
		t.Fatalf("non-admin reveal: %d", got.Code)
	}
	revealed := paymentCall(a.adminPromoCodeValue, "GET", "/api/admin/promo-codes/"+id+"/code", "admin", id, "")
	var result struct {
		Code string `json:"code"`
	}
	if revealed.Code != 200 || json.Unmarshal(revealed.Body.Bytes(), &result) != nil || result.Code != code || revealed.Header().Get("Cache-Control") != "private, no-store" {
		t.Fatalf("admin reveal: %d %s", revealed.Code, revealed.Body.String())
	}
	if got := paymentCall(a.adminPromoCodeValue, "GET", "/api/admin/promo-codes/missing/code", "admin", "missing", ""); got.Code != 404 {
		t.Fatalf("unknown code: %d", got.Code)
	}
	if _, err := a.db.Exec(`UPDATE promo_codes SET code_ciphertext='' WHERE id=?`, id); err != nil {
		t.Fatal(err)
	}
	if got := paymentCall(a.adminPromoCodeValue, "GET", "/api/admin/promo-codes/"+id+"/code", "admin", id, ""); got.Code != 409 {
		t.Fatalf("legacy code reveal: %d", got.Code)
	}
	if got := paymentCall(a.paymentQuote, "POST", "/api/payment/quote", "alice", "", `{"code":"`+code+`"}`); got.Code != 200 {
		t.Fatalf("legacy code still redeemable: %d %s", got.Code, got.Body.String())
	}
}

func TestPromoCodeKeyConfiguration(t *testing.T) {
	if _, err := loadPromoCodeKey("", false); err == nil {
		t.Fatal("production accepted missing promo code key")
	}
	if _, err := loadPromoCodeKey("too-short", false); err == nil {
		t.Fatal("production accepted invalid promo code key")
	}
	if key, err := loadPromoCodeKey(strings.Repeat("ab", 32), false); err != nil || len(key) != 32 {
		t.Fatalf("valid promo code key: %d %v", len(key), err)
	}
}
