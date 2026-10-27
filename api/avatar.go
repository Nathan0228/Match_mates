package main

import (
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	_ "image/png"
	"net/http"
	"os"
	"path/filepath"
)

const maxAvatarBytes = 2 << 20

func avatarURL(id, path string) string { return "/api/avatars/" + id + "?v=" + filepath.Base(path) }

// uploadAvatar decodes and re-encodes the image, removing EXIF and other
// uploaded metadata before it can be shown to another student.
func (a *App) uploadAvatar(w http.ResponseWriter, r *http.Request) {
	u, ok := a.mustAuth(w, r)
	if !ok {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxAvatarBytes+(64<<10))
	if err := r.ParseMultipartForm(maxAvatarBytes + (64 << 10)); err != nil {
		fail(w, 400, "头像文件不得超过 2 MB")
		return
	}
	f, _, err := r.FormFile("avatar")
	if err != nil {
		fail(w, 400, "请选择头像图片")
		return
	}
	defer f.Close()
	config, format, err := image.DecodeConfig(f)
	if err != nil || (format != "jpeg" && format != "png") {
		fail(w, 400, "头像仅支持 JPG 或 PNG")
		return
	}
	if config.Width < 1 || config.Height < 1 || config.Width > 4096 || config.Height > 4096 || int64(config.Width)*int64(config.Height) > 8_000_000 {
		fail(w, 400, "图片尺寸过大")
		return
	}
	if _, err := f.Seek(0, 0); err != nil {
		fail(w, 400, "读取图片失败")
		return
	}
	decoded, _, err := image.Decode(f)
	if err != nil {
		fail(w, 400, "图片无法解码")
		return
	}
	canvas := image.NewRGBA(decoded.Bounds())
	draw.Draw(canvas, canvas.Bounds(), image.NewUniform(color.White), image.Point{}, draw.Src)
	draw.Draw(canvas, canvas.Bounds(), decoded, decoded.Bounds().Min, draw.Over)
	dir := filepath.Join(a.uploadDir, "avatars")
	if err := os.MkdirAll(dir, 0700); err != nil {
		fail(w, 500, "保存头像失败")
		return
	}
	temp, err := os.CreateTemp(dir, "avatar-*.jpg")
	if err != nil {
		fail(w, 500, "保存头像失败")
		return
	}
	defer os.Remove(temp.Name())
	if err := jpeg.Encode(temp, canvas, &jpeg.Options{Quality: 85}); err != nil {
		temp.Close()
		fail(w, 500, "保存头像失败")
		return
	}
	if err := temp.Close(); err != nil {
		fail(w, 500, "保存头像失败")
		return
	}
	var oldPath string
	_ = a.db.QueryRow(`SELECT avatar_path FROM users WHERE id=?`, u.ID).Scan(&oldPath)
	path := filepath.Join(dir, randomHex(16)+".jpg")
	if err := os.Rename(temp.Name(), path); err != nil {
		fail(w, 500, "保存头像失败")
		return
	}
	if _, err := a.db.Exec(`UPDATE users SET avatar_path=? WHERE id=?`, path, u.ID); err != nil {
		_ = os.Remove(path)
		fail(w, 500, "保存头像失败")
		return
	}
	if oldPath != "" {
		_ = os.Remove(oldPath)
	}
	send(w, 200, map[string]string{"avatar_url": avatarURL(u.ID, path)})
}

func (a *App) avatar(w http.ResponseWriter, r *http.Request) {
	viewer, ok := a.mustAuth(w, r)
	if !ok {
		return
	}
	id := r.PathValue("id")
	if id != viewer.ID && !viewer.IsAdmin {
		if !viewer.active() {
			fail(w, 403, "无权查看此头像")
			return
		}
		target, err := a.getUser(id)
		if err != nil || !target.active() || target.Gender != viewer.Seeking || target.Seeking != viewer.Gender {
			fail(w, 404, "头像不存在")
			return
		}
		if !target.MatchingEnabled || !viewer.MatchingEnabled {
			var mutual int
			err = a.db.QueryRow(`SELECT EXISTS(SELECT 1 FROM likes WHERE from_id=? AND to_id=?) AND EXISTS(SELECT 1 FROM likes WHERE from_id=? AND to_id=?)`, viewer.ID, id, id, viewer.ID).Scan(&mutual)
			if err != nil || !truth(mutual) {
				fail(w, 404, "头像不存在")
				return
			}
		}
	}
	var path string
	if err := a.db.QueryRow(`SELECT avatar_path FROM users WHERE id=?`, id).Scan(&path); err != nil || path == "" {
		fail(w, 404, "头像不存在")
		return
	}
	w.Header().Set("Content-Type", "image/jpeg")
	w.Header().Set("Cache-Control", "private, no-store")
	http.ServeFile(w, r, path)
}
