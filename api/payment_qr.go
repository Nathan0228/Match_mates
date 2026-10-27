package main

import (
	"bytes"
	"image"
	_ "image/jpeg"
	"image/png"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func validPaymentChannel(channel string) bool { return channel == "wechat" || channel == "alipay" }

func (a *App) adminPaymentQRList(w http.ResponseWriter, r *http.Request) {
	if _, ok := a.admin(w, r); !ok {
		return
	}
	rows, err := a.db.Query(`SELECT channel,payee_name,version,active,updated_at FROM payment_qr ORDER BY channel`)
	if err != nil {
		fail(w, 500, "获取收款码失败")
		return
	}
	defer rows.Close()
	items := []map[string]any{}
	for rows.Next() {
		var channel, payeeName, version string
		var active int
		var updated int64
		if err := rows.Scan(&channel, &payeeName, &version, &active, &updated); err != nil {
			fail(w, 500, "获取收款码失败")
			return
		}
		items = append(items, map[string]any{"channel": channel, "payee_name": payeeName, "version": version, "active": active != 0, "updated_at": updated, "qr_url": "/api/payment/qr/" + channel})
	}
	if err := rows.Err(); err != nil {
		fail(w, 500, "获取收款码失败")
		return
	}
	send(w, 200, map[string]any{"codes": items})
}

func (a *App) adminUploadPaymentQR(w http.ResponseWriter, r *http.Request) {
	if _, ok := a.admin(w, r); !ok {
		return
	}
	channel := r.PathValue("channel")
	if !validPaymentChannel(channel) {
		fail(w, 400, "不支持的收款方式")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 2<<20+(64<<10))
	if err := r.ParseMultipartForm(2<<20 + (64 << 10)); err != nil {
		fail(w, 400, "收款码图片不得超过 2 MB")
		return
	}
	payeeName := strings.TrimSpace(r.FormValue("payee_name"))
	if len([]rune(payeeName)) < 2 || len([]rune(payeeName)) > 60 {
		fail(w, 400, "请填写 2 至 60 字的收款方名称")
		return
	}
	f, _, err := r.FormFile("qr")
	if err != nil {
		fail(w, 400, "请选择收款码图片")
		return
	}
	defer f.Close()
	config, format, err := image.DecodeConfig(f)
	if err != nil || (format != "png" && format != "jpeg") {
		fail(w, 400, "仅支持 PNG 或 JPG 收款码")
		return
	}
	if config.Width < 1 || config.Height < 1 || config.Width > 3000 || config.Height > 3000 || int64(config.Width)*int64(config.Height) > 4_000_000 {
		fail(w, 400, "图片尺寸过大")
		return
	}
	if _, err := f.Seek(0, 0); err != nil {
		fail(w, 400, "读取图片失败")
		return
	}
	img, _, err := image.Decode(f)
	if err != nil {
		fail(w, 400, "图片无法解码")
		return
	}
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, img); err != nil || encoded.Len() > 4<<20 {
		fail(w, 400, "收款码图片过大")
		return
	}
	dir := filepath.Join(a.uploadDir, "payment-qr")
	if err := os.MkdirAll(dir, 0700); err != nil {
		fail(w, 500, "保存收款码失败")
		return
	}
	path := filepath.Join(dir, randomHex(16)+".png")
	if err := os.WriteFile(path, encoded.Bytes(), 0600); err != nil {
		fail(w, 500, "保存收款码失败")
		return
	}
	var oldPath string
	_ = a.db.QueryRow(`SELECT image_path FROM payment_qr WHERE channel=?`, channel).Scan(&oldPath)
	_, err = a.db.Exec(`INSERT INTO payment_qr(channel,image_path,payee_name,version,active,updated_at) VALUES(?,?,?,?,1,?) ON CONFLICT(channel) DO UPDATE SET image_path=excluded.image_path,payee_name=excluded.payee_name,version=excluded.version,updated_at=excluded.updated_at`, channel, path, payeeName, randomHex(12), time.Now().Unix())
	if err != nil {
		_ = os.Remove(path)
		fail(w, 500, "保存收款码失败")
		return
	}
	if oldPath != "" {
		_ = os.Remove(oldPath)
	}
	send(w, 200, map[string]any{"qr_url": "/api/payment/qr/" + channel})
}

func (a *App) adminPaymentQRSettings(w http.ResponseWriter, r *http.Request) {
	if _, ok := a.admin(w, r); !ok {
		return
	}
	channel := r.PathValue("channel")
	if !validPaymentChannel(channel) {
		fail(w, 400, "不支持的收款方式")
		return
	}
	var v struct {
		PayeeName string `json:"payee_name"`
	}
	if !decode(w, r, &v) {
		return
	}
	name := strings.TrimSpace(v.PayeeName)
	if len([]rune(name)) < 2 || len([]rune(name)) > 60 {
		fail(w, 400, "请填写 2 至 60 字的收款方名称")
		return
	}
	result, err := a.db.Exec(`UPDATE payment_qr SET payee_name=?,version=?,updated_at=? WHERE channel=?`, name, randomHex(12), time.Now().Unix(), channel)
	if err != nil {
		fail(w, 500, "更新失败")
		return
	}
	if n, _ := result.RowsAffected(); n != 1 {
		fail(w, 404, "请先上传收款码")
		return
	}
	send(w, 200, map[string]any{"payee_name": name})
}

func (a *App) adminPaymentQRStatus(w http.ResponseWriter, r *http.Request) {
	if _, ok := a.admin(w, r); !ok {
		return
	}
	channel := r.PathValue("channel")
	if !validPaymentChannel(channel) {
		fail(w, 400, "不支持的收款方式")
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
	result, err := a.db.Exec(`UPDATE payment_qr SET active=? WHERE channel=?`, active, channel)
	if err != nil {
		fail(w, 500, "更新失败")
		return
	}
	if n, _ := result.RowsAffected(); n != 1 {
		fail(w, 404, "请先上传收款码")
		return
	}
	send(w, 200, map[string]any{"active": *v.Active})
}

func (a *App) paymentQR(w http.ResponseWriter, r *http.Request) {
	u, ok := a.mustAuth(w, r)
	if !ok {
		return
	}
	channel := r.PathValue("channel")
	if !validPaymentChannel(channel) {
		fail(w, 404, "收款码不存在")
		return
	}
	var path string
	var active int
	if err := a.db.QueryRow(`SELECT image_path,active FROM payment_qr WHERE channel=?`, channel).Scan(&path, &active); err != nil || path == "" || (active == 0 && !u.IsAdmin) {
		fail(w, 404, "收款码不存在")
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "private, no-store")
	http.ServeFile(w, r, path)
}
