package main

import (
	"bytes"
	"database/sql"
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

func TestAvatarUploadAndAccess(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	if err := migrate(db); err != nil {
		t.Fatal(err)
	}
	a := &App{db: db, dev: true, uploadDir: t.TempDir()}
	_, err = db.Exec(`INSERT INTO users(id,email,password_hash,nickname,created_at) VALUES('one','one@mails.ucas.ac.cn','hash','甲',?)`, time.Now().Unix())
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`INSERT INTO sessions(token_hash,user_id,expires_at) VALUES(?,?,?)`, digest("test-token"), "one", time.Now().Add(time.Hour).Unix())
	if err != nil {
		t.Fatal(err)
	}

	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	img.Set(0, 0, color.RGBA{R: 250, A: 255})
	var pngBytes bytes.Buffer
	if err := png.Encode(&pngBytes, img); err != nil {
		t.Fatal(err)
	}
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	file, err := writer.CreateFormFile("avatar", "face.png")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.Write(pngBytes.Bytes()); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest("POST", "/api/avatar", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.AddCookie(&http.Cookie{Name: "mm_session", Value: "test-token"})
	w := httptest.NewRecorder()
	a.uploadAvatar(w, req)
	if w.Code != 200 {
		t.Fatalf("upload: %d %s", w.Code, w.Body.String())
	}
	u, err := a.getUser("one")
	if err != nil || !strings.Contains(u.AvatarURL, "/api/avatars/one?v=") {
		t.Fatalf("avatar URL: %q, err: %v", u.AvatarURL, err)
	}
	view := httptest.NewRequest("GET", u.AvatarURL, nil)
	view.SetPathValue("id", "one")
	view.AddCookie(&http.Cookie{Name: "mm_session", Value: "test-token"})
	viewResponse := httptest.NewRecorder()
	a.avatar(viewResponse, view)
	if viewResponse.Code != 200 || !strings.HasPrefix(viewResponse.Header().Get("Content-Type"), "image/jpeg") {
		t.Fatalf("avatar response: %d %s", viewResponse.Code, viewResponse.Header().Get("Content-Type"))
	}
	if _, _, err := image.Decode(bytes.NewReader(viewResponse.Body.Bytes())); err != nil {
		t.Fatalf("stored image is invalid: %v", err)
	}
	unauthorized := httptest.NewRecorder()
	request := httptest.NewRequest("GET", u.AvatarURL, nil)
	request.SetPathValue("id", "one")
	a.avatar(unauthorized, request)
	if unauthorized.Code != 401 {
		t.Fatalf("anonymous avatar access: %d", unauthorized.Code)
	}
	_, err = db.Exec(`UPDATE users SET gender='male',seeking='female',major='计算机',campus='雁栖湖',intro='喜欢徒步和电影，想认识愿意一起在湖边散步的人。',hobbies='["徒步","电影"]',email_verified=1,paid=1,card_status='approved' WHERE id='one'`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`INSERT INTO users(id,email,password_hash,nickname,gender,seeking,major,campus,intro,hobbies,email_verified,paid,card_status,created_at) VALUES('other','other@mails.ucas.ac.cn','hash','乙','female','female','计算机','雁栖湖','喜欢徒步和电影，想认识愿意一起在湖边散步的人。','["徒步","电影"]',1,1,'approved',?)`, time.Now().Unix())
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`INSERT INTO sessions(token_hash,user_id,expires_at) VALUES(?,?,?)`, digest("other-token"), "other", time.Now().Add(time.Hour).Unix())
	if err != nil {
		t.Fatal(err)
	}
	wrongGroup := httptest.NewRequest("GET", u.AvatarURL, nil)
	wrongGroup.SetPathValue("id", "one")
	wrongGroup.AddCookie(&http.Cookie{Name: "mm_session", Value: "other-token"})
	wrongGroupResponse := httptest.NewRecorder()
	a.avatar(wrongGroupResponse, wrongGroup)
	if wrongGroupResponse.Code != 404 {
		t.Fatalf("cross-group avatar access: %d", wrongGroupResponse.Code)
	}
}

func TestAvatarColumnMigratesExistingUsers(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	_, err = db.Exec(`CREATE TABLE users (
	 id TEXT PRIMARY KEY, email TEXT NOT NULL UNIQUE, password_hash TEXT NOT NULL, nickname TEXT NOT NULL,
	 gender TEXT NOT NULL DEFAULT '', seeking TEXT NOT NULL DEFAULT '', major TEXT NOT NULL DEFAULT '', campus TEXT NOT NULL DEFAULT '雁栖湖',
	 intro TEXT NOT NULL DEFAULT '', hobbies TEXT NOT NULL DEFAULT '[]', email_verified INTEGER NOT NULL DEFAULT 0,
	 paid INTEGER NOT NULL DEFAULT 0, card_status TEXT NOT NULL DEFAULT 'none', card_path TEXT NOT NULL DEFAULT '',
	 created_at INTEGER NOT NULL);`)
	if err != nil {
		t.Fatal(err)
	}
	if err := migrate(db); err != nil {
		t.Fatalf("migrate old database: %v", err)
	}
	if err := migrate(db); err != nil {
		t.Fatalf("repeat migration: %v", err)
	}
	_, err = db.Exec(`INSERT INTO users(id,email,password_hash,nickname,created_at) VALUES('old','old@mails.ucas.ac.cn','hash','旧同学',?)`, time.Now().Unix())
	if err != nil {
		t.Fatal(err)
	}
	a := &App{db: db}
	if _, err := a.getUser("old"); err != nil {
		t.Fatalf("read old user: %v", err)
	}
}
