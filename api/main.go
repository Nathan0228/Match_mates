package main

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/smtp"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
	_ "modernc.org/sqlite"
)

type App struct {
	db                            *sql.DB
	dev                           bool
	origin, uploadDir, adminEmail string
	promoCodeKey                  []byte
	tmdbToken, tmdbAPIKey         string
	tmdbBaseURL                   string
	tmdbImageBaseURL              string
	tmdbHTTPClient                *http.Client
}

var errAccountSuspended = errors.New("account suspended")

type User struct {
	ID              string           `json:"id"`
	Email           string           `json:"email"`
	Nickname        string           `json:"nickname"`
	Gender          string           `json:"gender"`
	Seeking         string           `json:"seeking"`
	Major           string           `json:"major"`
	Campus          string           `json:"campus"`
	Intro           string           `json:"intro"`
	Hobbies         []string         `json:"hobbies"`
	Media           MediaShelf       `json:"media"`
	Preferences     MatchPreferences `json:"preferences"`
	EmailVerified   bool             `json:"email_verified"`
	AvatarURL       string           `json:"avatar_url"`
	Paid            bool             `json:"paid"`
	CardStatus      string           `json:"card_status"`
	Status          string           `json:"status"`
	IsAdmin         bool             `json:"is_admin"`
	MatchingEnabled bool             `json:"matching_enabled"`
	SuspendedAt     int64            `json:"-"`
}
type Candidate struct {
	ID          string           `json:"id"`
	Nickname    string           `json:"nickname"`
	Major       string           `json:"major"`
	Campus      string           `json:"campus"`
	Intro       string           `json:"intro"`
	Hobbies     []string         `json:"hobbies"`
	Media       MediaShelf       `json:"media"`
	Preferences MatchPreferences `json:"preferences"`
	Score       float64          `json:"score"`
	Reasons     []string         `json:"reasons"`
	Liked       bool             `json:"liked"`
	AvatarURL   string           `json:"avatar_url"`
}
type input struct {
	Email       string            `json:"email"`
	Password    string            `json:"password"`
	Nickname    string            `json:"nickname"`
	Code        string            `json:"code"`
	Gender      string            `json:"gender"`
	Seeking     string            `json:"seeking"`
	Major       string            `json:"major"`
	Campus      string            `json:"campus"`
	Intro       string            `json:"intro"`
	Hobbies     []string          `json:"hobbies"`
	Media       *MediaShelf       `json:"media"`
	Preferences *MatchPreferences `json:"preferences"`
	Decision    string            `json:"decision"`
}

func main() {
	if err := loadDotEnv(".env"); err != nil {
		log.Fatal(err)
	}
	path := env("DB_PATH", "data/matchmates.db")
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		log.Fatal(err)
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	if err := migrate(db); err != nil {
		log.Fatal(err)
	}
	dev := os.Getenv("APP_ENV") == "development"
	promoCodeKey, err := loadPromoCodeKey(os.Getenv("PROMO_CODE_KEY"), dev)
	if err != nil {
		log.Fatal(err)
	}
	a := &App{db: db, dev: dev, origin: os.Getenv("FRONTEND_ORIGIN"), uploadDir: env("UPLOAD_DIR", "uploads"), adminEmail: strings.ToLower(os.Getenv("ADMIN_EMAIL")), promoCodeKey: promoCodeKey, tmdbToken: os.Getenv("TMDB_READ_ACCESS_TOKEN"), tmdbAPIKey: os.Getenv("TMDB_API_KEY")}
	if !a.dev && (a.origin == "" || a.adminEmail == "" || os.Getenv("SMTP_HOST") == "") {
		log.Fatal("production requires FRONTEND_ORIGIN, ADMIN_EMAIL and SMTP_HOST")
	}
	if err := os.MkdirAll(a.uploadDir, 0700); err != nil {
		log.Fatal(err)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) { send(w, 200, map[string]any{"ok": true}) })
	mux.HandleFunc("POST /api/register", a.register)
	mux.HandleFunc("POST /api/resend-code", a.resendCode)
	mux.HandleFunc("POST /api/verify-email", a.verifyEmail)
	mux.HandleFunc("POST /api/login", a.login)
	mux.HandleFunc("POST /api/password-reset/request", a.requestPasswordReset)
	mux.HandleFunc("POST /api/password-reset/confirm", a.confirmPasswordReset)
	mux.HandleFunc("POST /api/logout", a.logout)
	mux.HandleFunc("GET /api/me", a.me)
	mux.HandleFunc("GET /api/media/search", a.searchMedia)
	mux.HandleFunc("GET /api/media/poster/{size}/{file}", a.mediaPoster)
	a.registerNotificationRoutes(mux)
	mux.HandleFunc("PUT /api/profile", a.profile)
	mux.HandleFunc("PUT /api/settings/matching", a.updateMatchingSetting)
	mux.HandleFunc("POST /api/settings/password", a.changePassword)
	mux.HandleFunc("POST /api/avatar", a.uploadAvatar)
	mux.HandleFunc("GET /api/avatars/{id}", a.avatar)
	a.registerPaymentRoutes(mux)
	mux.HandleFunc("POST /api/student-card", a.studentCard)
	mux.HandleFunc("GET /api/candidates", a.candidates)
	mux.HandleFunc("POST /api/likes/{id}", a.like)
	mux.HandleFunc("GET /api/matches", a.matches)
	mux.HandleFunc("GET /api/admin/reviews", a.adminReviews)
	mux.HandleFunc("GET /api/admin/summary", a.adminSummary)
	mux.HandleFunc("GET /api/admin/history", a.adminHistory)
	mux.HandleFunc("POST /api/admin/reviews/{id}", a.adminDecision)
	mux.HandleFunc("GET /api/admin/cards/{id}", a.adminCard)
	a.registerUserManagementRoutes(mux)
	log.Printf("API listening on %s", env("PORT", "8080"))
	log.Fatal(http.ListenAndServe(":"+env("PORT", "8080"), a.headers(mux)))
}

func migrate(db *sql.DB) error {
	_, err := db.Exec(`PRAGMA foreign_keys=ON;
	CREATE TABLE IF NOT EXISTS users (
	 id TEXT PRIMARY KEY, email TEXT NOT NULL UNIQUE, password_hash TEXT NOT NULL, nickname TEXT NOT NULL,
	 gender TEXT NOT NULL DEFAULT '', seeking TEXT NOT NULL DEFAULT '', major TEXT NOT NULL DEFAULT '', campus TEXT NOT NULL DEFAULT '雁栖湖',
	 intro TEXT NOT NULL DEFAULT '', hobbies TEXT NOT NULL DEFAULT '[]', media_shelf TEXT NOT NULL DEFAULT '{}', match_preferences TEXT NOT NULL DEFAULT '{}', email_verified INTEGER NOT NULL DEFAULT 0,
	 paid INTEGER NOT NULL DEFAULT 0, card_status TEXT NOT NULL DEFAULT 'none', card_path TEXT NOT NULL DEFAULT '', card_submitted_at INTEGER NOT NULL DEFAULT 0, avatar_path TEXT NOT NULL DEFAULT '', matching_enabled INTEGER NOT NULL DEFAULT 1,
	 created_at INTEGER NOT NULL
	);
	CREATE INDEX IF NOT EXISTS users_matching ON users(gender,seeking,email_verified,paid,card_status);
	CREATE TABLE IF NOT EXISTS email_codes (user_id TEXT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE, code_hash TEXT NOT NULL, expires_at INTEGER NOT NULL);
	CREATE TABLE IF NOT EXISTS password_reset_codes (user_id TEXT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE, code_hash TEXT NOT NULL, expires_at INTEGER NOT NULL, requested_at INTEGER NOT NULL, attempts INTEGER NOT NULL DEFAULT 0);
	CREATE TABLE IF NOT EXISTS sessions (token_hash TEXT PRIMARY KEY, user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE, expires_at INTEGER NOT NULL);
	CREATE INDEX IF NOT EXISTS sessions_user ON sessions(user_id);
	CREATE TABLE IF NOT EXISTS likes (from_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE, to_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE, created_at INTEGER NOT NULL, PRIMARY KEY(from_id,to_id));
	CREATE TABLE IF NOT EXISTS review_events (id INTEGER PRIMARY KEY AUTOINCREMENT, user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE, reviewer_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE, decision TEXT NOT NULL, created_at INTEGER NOT NULL);
	CREATE INDEX IF NOT EXISTS review_events_recent ON review_events(created_at DESC, id DESC);`)
	if err != nil {
		return err
	}
	// Existing SQLite databases may predate profile pictures and matching prompts.
	rows, err := db.Query(`PRAGMA table_info(users)`)
	if err != nil {
		return err
	}
	hasAvatar, hasPreferences, hasCardSubmittedAt, hasMedia := false, false, false, false
	for rows.Next() {
		var cid, notnull, pk int
		var name, kind string
		var defaultValue sql.NullString
		if err := rows.Scan(&cid, &name, &kind, &notnull, &defaultValue, &pk); err != nil {
			rows.Close()
			return err
		}
		if name == "avatar_path" {
			hasAvatar = true
		}
		if name == "match_preferences" {
			hasPreferences = true
		}
		if name == "card_submitted_at" {
			hasCardSubmittedAt = true
		}
		if name == "media_shelf" {
			hasMedia = true
		}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	if err := rows.Close(); err != nil {
		return err
	}
	if !hasAvatar {
		if _, err = db.Exec(`ALTER TABLE users ADD COLUMN avatar_path TEXT NOT NULL DEFAULT ''`); err != nil {
			return err
		}
	}
	if !hasPreferences {
		if _, err = db.Exec(`ALTER TABLE users ADD COLUMN match_preferences TEXT NOT NULL DEFAULT '{}'`); err != nil {
			return err
		}
	}
	if !hasCardSubmittedAt {
		if _, err = db.Exec(`ALTER TABLE users ADD COLUMN card_submitted_at INTEGER NOT NULL DEFAULT 0`); err != nil {
			return err
		}
	}
	if !hasMedia {
		if _, err = db.Exec(`ALTER TABLE users ADD COLUMN media_shelf TEXT NOT NULL DEFAULT '{}'`); err != nil {
			return err
		}
	}
	if err := migratePayments(db); err != nil {
		return err
	}
	if err := migrateUserManagement(db); err != nil {
		return err
	}
	return migrateNotifications(db)
}
func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
func send(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func fail(w http.ResponseWriter, status int, message string) {
	send(w, status, map[string]string{"error": message})
}
func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	if err := d.Decode(v); err != nil {
		fail(w, 400, "请求格式不正确")
		return false
	}
	return true
}
func randomHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}
func digest(s string) string { h := sha256.Sum256([]byte(s)); return hex.EncodeToString(h[:]) }
func validEmail(email string) bool {
	email = strings.ToLower(strings.TrimSpace(email))
	return strings.HasSuffix(email, "@mails.ucas.ac.cn") && len(strings.Split(email, "@")) == 2 && len(strings.Split(email, "@")[0]) > 0 && !strings.ContainsAny(email, " \t\n")
}
func validGender(value string) bool { return value == "male" || value == "female" }
func truth(n int) bool              { return n != 0 }
func (u User) active() bool {
	return u.SuspendedAt == 0 && u.EmailVerified && u.Paid && u.CardStatus == "approved" && u.Gender != "" && u.Seeking != "" && len([]rune(u.Intro)) >= 20 && len(u.Hobbies) >= 2
}
func (a *App) headers(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if a.origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", a.origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,OPTIONS")
			w.Header().Set("Vary", "Origin")
		}
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		if r.Method == "OPTIONS" {
			w.WriteHeader(204)
			return
		}
		if r.Method != "GET" && r.Method != "HEAD" && a.origin != "" {
			o := r.Header.Get("Origin")
			if o != "" && o != a.origin {
				fail(w, 403, "来源不允许")
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}
func (a *App) getUser(id string) (User, error) {
	var u User
	var hs, prefs, media string
	var verified, paid, matchingEnabled int
	var avatarPath string
	err := a.db.QueryRow(`SELECT id,email,nickname,gender,seeking,major,campus,intro,hobbies,media_shelf,match_preferences,email_verified,paid,card_status,avatar_path,suspended_at,matching_enabled FROM users WHERE id=?`, id).Scan(&u.ID, &u.Email, &u.Nickname, &u.Gender, &u.Seeking, &u.Major, &u.Campus, &u.Intro, &hs, &media, &prefs, &verified, &paid, &u.CardStatus, &avatarPath, &u.SuspendedAt, &matchingEnabled)
	if err != nil {
		return u, err
	}
	_ = json.Unmarshal([]byte(hs), &u.Hobbies)
	_ = json.Unmarshal([]byte(media), &u.Media)
	u.Media.normalize()
	_ = json.Unmarshal([]byte(prefs), &u.Preferences)
	if u.Hobbies == nil {
		u.Hobbies = []string{}
	}
	if u.Preferences.Values == nil {
		u.Preferences.Values = []string{}
	}
	u.EmailVerified = truth(verified)
	if avatarPath != "" {
		u.AvatarURL = avatarURL(u.ID, avatarPath)
	}
	u.Paid = truth(paid)
	u.MatchingEnabled = truth(matchingEnabled)
	u.IsAdmin = a.adminEmail != "" && u.Email == a.adminEmail
	if u.SuspendedAt != 0 {
		u.Status = "suspended"
	} else if u.active() {
		u.Status = "active"
	} else {
		u.Status = "pending"
	}
	return u, nil
}
func (a *App) auth(r *http.Request) (User, error) {
	c, err := r.Cookie("mm_session")
	if err != nil {
		return User{}, err
	}
	var id string
	err = a.db.QueryRow(`SELECT user_id FROM sessions WHERE token_hash=? AND expires_at>?`, digest(c.Value), time.Now().Unix()).Scan(&id)
	if err != nil {
		return User{}, err
	}
	u, err := a.getUser(id)
	if err != nil || u.SuspendedAt != 0 {
		return User{}, errors.New("session unavailable")
	}
	return u, nil
}
func (a *App) mustAuth(w http.ResponseWriter, r *http.Request) (User, bool) {
	u, err := a.auth(r)
	if err != nil {
		fail(w, 401, "请先登录")
		return u, false
	}
	return u, true
}
func (a *App) mustActive(w http.ResponseWriter, r *http.Request) (User, bool) {
	u, ok := a.mustAuth(w, r)
	if !ok {
		return u, false
	}
	if !u.active() {
		fail(w, 403, "完成资料、会费和学生证审核后才能使用匹配")
		return u, false
	}
	return u, true
}
func (a *App) session(w http.ResponseWriter, r *http.Request, id string) error {
	token := randomHex(32)
	result, err := a.db.Exec(`INSERT INTO sessions(token_hash,user_id,expires_at) SELECT ?,id,? FROM users WHERE id=? AND suspended_at=0`, digest(token), time.Now().Add(30*24*time.Hour).Unix(), id)
	if err != nil {
		return err
	}
	if count, err := result.RowsAffected(); err != nil || count != 1 {
		return errAccountSuspended
	}
	http.SetCookie(w, &http.Cookie{Name: "mm_session", Value: token, Path: "/", HttpOnly: true, Secure: r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https", SameSite: http.SameSiteLaxMode, MaxAge: 30 * 24 * 3600})
	return nil
}

func (a *App) register(w http.ResponseWriter, r *http.Request) {
	var v input
	if !decode(w, r, &v) {
		return
	}
	v.Email = strings.ToLower(strings.TrimSpace(v.Email))
	v.Nickname = strings.TrimSpace(v.Nickname)
	if !validEmail(v.Email) {
		fail(w, 400, "仅接受 @mails.ucas.ac.cn 校园邮箱")
		return
	}
	if len(v.Password) < 8 || len(v.Password) > 128 {
		fail(w, 400, "密码需为 8–128 位")
		return
	}
	if len([]rune(v.Nickname)) < 1 || len([]rune(v.Nickname)) > 24 {
		fail(w, 400, "昵称需为 1–24 字")
		return
	}
	if !validGender(v.Gender) || !validGender(v.Seeking) {
		fail(w, 400, "请选择性别和希望认识的性别")
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(v.Password), bcrypt.DefaultCost)
	if err != nil {
		fail(w, 500, "注册失败")
		return
	}
	id := randomHex(16)
	code := fmt.Sprintf("%06d", randomNumber(1000000))
	tx, err := a.db.Begin()
	if err != nil {
		fail(w, 500, "注册失败")
		return
	}
	defer tx.Rollback()
	_, err = tx.Exec(`INSERT INTO users(id,email,password_hash,nickname,gender,seeking,created_at) VALUES(?,?,?,?,?,?,?)`, id, v.Email, string(hash), v.Nickname, v.Gender, v.Seeking, time.Now().Unix())
	if err != nil {
		fail(w, 409, "该邮箱已注册")
		return
	}
	_, err = tx.Exec(`INSERT INTO email_codes(user_id,code_hash,expires_at) VALUES(?,?,?)`, id, digest(code), time.Now().Add(10*time.Minute).Unix())
	if err != nil {
		fail(w, 500, "注册失败")
		return
	}
	if err = sendCode(v.Email, code, a.dev); err != nil {
		fail(w, 503, "邮件发送失败，请检查邮件服务配置")
		return
	}
	if err = tx.Commit(); err != nil {
		fail(w, 500, "注册失败")
		return
	}
	out := map[string]any{"message": "验证码已发送，有效期 10 分钟"}
	if a.dev && os.Getenv("SMTP_HOST") == "" {
		out["dev_code"] = code
	}
	send(w, 201, out)
}
func (a *App) resendCode(w http.ResponseWriter, r *http.Request) {
	var v input
	if !decode(w, r, &v) {
		return
	}
	var id, hash string
	var verified int
	err := a.db.QueryRow(`SELECT id,password_hash,email_verified FROM users WHERE email=?`, strings.ToLower(strings.TrimSpace(v.Email))).Scan(&id, &hash, &verified)
	if err != nil || bcrypt.CompareHashAndPassword([]byte(hash), []byte(v.Password)) != nil {
		fail(w, 401, "邮箱或密码不正确")
		return
	}
	if truth(verified) {
		fail(w, 400, "该邮箱已完成验证，请直接登录")
		return
	}
	code := fmt.Sprintf("%06d", randomNumber(1000000))
	if err := sendCode(strings.ToLower(strings.TrimSpace(v.Email)), code, a.dev); err != nil {
		fail(w, 503, "邮件发送失败")
		return
	}
	_, err = a.db.Exec(`INSERT INTO email_codes(user_id,code_hash,expires_at) VALUES(?,?,?) ON CONFLICT(user_id) DO UPDATE SET code_hash=excluded.code_hash,expires_at=excluded.expires_at`, id, digest(code), time.Now().Add(10*time.Minute).Unix())
	if err != nil {
		fail(w, 500, "重发失败")
		return
	}
	out := map[string]any{"message": "验证码已重新发送"}
	if a.dev && os.Getenv("SMTP_HOST") == "" {
		out["dev_code"] = code
	}
	send(w, 200, out)
}

func randomNumber(max int) int {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return int((uint32(b[0])<<24 | uint32(b[1])<<16 | uint32(b[2])<<8 | uint32(b[3])) % uint32(max))
}
func sendCode(email, code string, dev bool) error {
	return sendEmail(email, "Jianshan email verification", "您的见山验证码是 "+code+"，10 分钟内有效。", dev)
}
func sendEmail(email, subject, body string, dev bool) error {
	host := os.Getenv("SMTP_HOST")
	if host == "" {
		if dev {
			return nil
		}
		return errors.New("SMTP_HOST missing")
	}
	port := env("SMTP_PORT", "587")
	user := os.Getenv("SMTP_USER")
	pass := os.Getenv("SMTP_PASS")
	from := env("SMTP_FROM", user)
	if user == "" || pass == "" || from == "" {
		return errors.New("SMTP credentials missing")
	}
	msg := []byte("To: " + email + "\r\nSubject: " + subject + "\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n" + body + "\r\n")
	return smtp.SendMail(host+":"+port, smtp.PlainAuth("", user, pass, host), from, []string{email}, msg)
}
func (a *App) verifyEmail(w http.ResponseWriter, r *http.Request) {
	var v input
	if !decode(w, r, &v) {
		return
	}
	var id, hash string
	var exp int64
	var suspendedAt int64
	err := a.db.QueryRow(`SELECT u.id,c.code_hash,c.expires_at,u.suspended_at FROM users u JOIN email_codes c ON c.user_id=u.id WHERE u.email=?`, strings.ToLower(strings.TrimSpace(v.Email))).Scan(&id, &hash, &exp, &suspendedAt)
	if err != nil || exp < time.Now().Unix() || digest(v.Code) != hash {
		fail(w, 400, "验证码无效或已过期")
		return
	}
	if suspendedAt != 0 {
		fail(w, 403, "账号已停用，请联系管理员")
		return
	}
	tx, err := a.db.Begin()
	if err != nil {
		fail(w, 500, "验证失败")
		return
	}
	defer tx.Rollback()
	if _, err = tx.Exec(`UPDATE users SET email_verified=1 WHERE id=?`, id); err != nil {
		fail(w, 500, "验证失败")
		return
	}
	if _, err = tx.Exec(`DELETE FROM email_codes WHERE user_id=?`, id); err != nil {
		fail(w, 500, "验证失败")
		return
	}
	if err = tx.Commit(); err != nil {
		fail(w, 500, "验证失败")
		return
	}
	if err = a.session(w, r, id); err != nil {
		if errors.Is(err, errAccountSuspended) {
			fail(w, 403, "账号已停用，请联系管理员")
			return
		}
		fail(w, 500, "登录失败")
		return
	}
	u, _ := a.getUser(id)
	send(w, 200, map[string]any{"user": u})
}
func (a *App) login(w http.ResponseWriter, r *http.Request) {
	var v input
	if !decode(w, r, &v) {
		return
	}
	var id, hash string
	var verified int
	var suspendedAt int64
	err := a.db.QueryRow(`SELECT id,password_hash,email_verified,suspended_at FROM users WHERE email=?`, strings.ToLower(strings.TrimSpace(v.Email))).Scan(&id, &hash, &verified, &suspendedAt)
	if err != nil || bcrypt.CompareHashAndPassword([]byte(hash), []byte(v.Password)) != nil {
		fail(w, 401, "邮箱或密码不正确")
		return
	}
	if !truth(verified) {
		fail(w, 403, "请先完成邮箱验证")
		return
	}
	if suspendedAt != 0 {
		fail(w, 403, "账号已停用，请联系管理员")
		return
	}
	if err = a.session(w, r, id); err != nil {
		if errors.Is(err, errAccountSuspended) {
			fail(w, 403, "账号已停用，请联系管理员")
			return
		}
		fail(w, 500, "登录失败")
		return
	}
	u, _ := a.getUser(id)
	send(w, 200, map[string]any{"user": u})
}
func (a *App) logout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie("mm_session"); err == nil {
		_, _ = a.db.Exec(`DELETE FROM sessions WHERE token_hash=?`, digest(c.Value))
	}
	http.SetCookie(w, &http.Cookie{Name: "mm_session", Path: "/", MaxAge: -1, HttpOnly: true, SameSite: http.SameSiteLaxMode})
	send(w, 200, map[string]any{"ok": true})
}
func (a *App) me(w http.ResponseWriter, r *http.Request) {
	u, ok := a.mustAuth(w, r)
	if ok {
		send(w, 200, map[string]any{"user": u})
	}
}
func (a *App) profile(w http.ResponseWriter, r *http.Request) {
	u, ok := a.mustAuth(w, r)
	if !ok {
		return
	}
	var v input
	if !decode(w, r, &v) {
		return
	}
	v.Nickname = strings.TrimSpace(v.Nickname)
	v.Intro = strings.TrimSpace(v.Intro)
	if len([]rune(v.Nickname)) < 1 || len([]rune(v.Nickname)) > 24 || len([]rune(v.Intro)) < 20 || len([]rune(v.Intro)) > 1000 || len([]rune(v.Major)) > 60 || len([]rune(v.Campus)) > 40 || !validGender(v.Gender) || !validGender(v.Seeking) {
		fail(w, 400, "请完整填写有效的个人资料（介绍至少 20 字）")
		return
	}
	if (u.Gender != "" && v.Gender != u.Gender) || (u.Seeking != "" && v.Seeking != u.Seeking) {
		fail(w, 409, "性别和交友方向确定后不可修改")
		return
	}
	seen := map[string]bool{}
	for _, h := range v.Hobbies {
		if len([]rune(h)) > 30 || strings.TrimSpace(h) == "" {
			fail(w, 400, "兴趣标签不合法")
			return
		}
		seen[h] = true
	}
	if len(seen) < 2 || len(seen) > 20 {
		fail(w, 400, "请选择 2–20 项兴趣")
		return
	}
	prefs := u.Preferences
	if v.Preferences != nil {
		prefs = *v.Preferences
	}
	if !validPreferences(prefs) {
		fail(w, 400, "相处偏好或简短介绍填写不合法")
		return
	}
	media := u.Media
	if v.Media != nil {
		media = *v.Media
	}
	if !media.valid() {
		fail(w, 400, "音乐、影视、书籍或游戏条目不合法（每类最多 5 项）")
		return
	}
	hs, _ := json.Marshal(v.Hobbies)
	ms, _ := json.Marshal(media)
	ps, _ := json.Marshal(prefs)
	result, err := a.db.Exec(`UPDATE users SET nickname=?,gender=?,seeking=?,major=?,campus=?,intro=?,hobbies=?,media_shelf=?,match_preferences=? WHERE id=? AND (gender='' OR gender=?) AND (seeking='' OR seeking=?)`, v.Nickname, v.Gender, v.Seeking, strings.TrimSpace(v.Major), strings.TrimSpace(v.Campus), v.Intro, string(hs), string(ms), string(ps), u.ID, v.Gender, v.Seeking)
	if err != nil {
		fail(w, 500, "保存失败")
		return
	}
	if affected, err := result.RowsAffected(); err != nil || affected == 0 {
		fail(w, 409, "性别和交友方向确定后不可修改")
		return
	}
	nu, _ := a.getUser(u.ID)
	send(w, 200, map[string]any{"user": nu})
}
func validateFile(f multipart.File) (string, error) {
	b := make([]byte, 512)
	n, _ := f.Read(b)
	_, _ = f.Seek(0, 0)
	kind := http.DetectContentType(b[:n])
	switch kind {
	case "image/jpeg":
		return ".jpg", nil
	case "image/png":
		return ".png", nil
	case "image/webp":
		return ".webp", nil
	case "application/pdf":
		return ".pdf", nil
	}
	return "", errors.New("unsupported file")
}
func (a *App) studentCard(w http.ResponseWriter, r *http.Request) {
	u, ok := a.mustAuth(w, r)
	if !ok {
		return
	}
	if u.CardStatus == "approved" {
		fail(w, 409, "学生证已审核通过")
		return
	}
	if !u.Paid {
		fail(w, 403, "请先完成会费步骤")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 6<<20)
	if err := r.ParseMultipartForm(6 << 20); err != nil {
		fail(w, 400, "文件不得超过 5 MB")
		return
	}
	f, _, err := r.FormFile("student_card")
	if err != nil {
		fail(w, 400, "请选择学生证图片或 PDF")
		return
	}
	defer f.Close()
	ext, err := validateFile(f)
	if err != nil {
		fail(w, 400, "仅支持 JPG、PNG、WebP 或 PDF")
		return
	}
	temp, err := os.CreateTemp(a.uploadDir, "card-*")
	if err != nil {
		fail(w, 500, "保存失败")
		return
	}
	defer os.Remove(temp.Name())
	defer temp.Close()
	n, err := io.Copy(temp, io.LimitReader(f, (5<<20)+1))
	if err != nil || n > 5<<20 {
		fail(w, 400, "文件不得超过 5 MB")
		return
	}
	var oldPath string
	_ = a.db.QueryRow(`SELECT card_path FROM users WHERE id=?`, u.ID).Scan(&oldPath)
	target := filepath.Join(a.uploadDir, randomHex(16)+ext)
	if err := temp.Close(); err != nil {
		fail(w, 500, "保存失败")
		return
	}
	if err := os.Rename(temp.Name(), target); err != nil {
		fail(w, 500, "保存失败")
		return
	}
	now := time.Now().Unix()
	tx, err := a.db.Begin()
	if err != nil {
		_ = os.Remove(target)
		fail(w, 500, "保存失败")
		return
	}
	defer tx.Rollback()
	_, err = tx.Exec(`UPDATE users SET card_path=?,card_status='pending',card_submitted_at=? WHERE id=?`, target, now, u.ID)
	if err != nil {
		_ = os.Remove(target)
		fail(w, 500, "保存失败")
		return
	}
	if err := a.addAdminNotification(tx, "admin_card_pending", u.ID, "学生证待审核", u.Nickname+"提交了学生证，等待审核。", "admin:queue", now); err != nil {
		_ = os.Remove(target)
		fail(w, 500, "保存失败")
		return
	}
	if err := tx.Commit(); err != nil {
		_ = os.Remove(target)
		fail(w, 500, "保存失败")
		return
	}
	if oldPath != "" {
		_ = os.Remove(oldPath)
	}
	send(w, 200, map[string]any{"status": "pending"})
}
func (a *App) candidates(w http.ResponseWriter, r *http.Request) {
	u, ok := a.mustActive(w, r)
	if !ok {
		return
	}
	if !u.MatchingEnabled {
		send(w, 200, map[string]any{"candidates": []Candidate{}})
		return
	}
	rows, err := a.db.Query(`SELECT id,nickname,major,campus,intro,hobbies,media_shelf,match_preferences,avatar_path,EXISTS(SELECT 1 FROM likes WHERE from_id=? AND to_id=users.id) FROM users WHERE id<>? AND gender=? AND seeking=? AND email_verified=1 AND paid=1 AND card_status='approved' AND suspended_at=0 AND matching_enabled=1 AND length(intro)>=20`, u.ID, u.ID, u.Seeking, u.Gender)
	if err != nil {
		fail(w, 500, "获取匹配失败")
		return
	}
	defer rows.Close()
	result := []Candidate{}
	for rows.Next() {
		var c Candidate
		var hs, prefs, media string
		var avatarPath string
		var liked int
		if rows.Scan(&c.ID, &c.Nickname, &c.Major, &c.Campus, &c.Intro, &hs, &media, &prefs, &avatarPath, &liked) != nil {
			continue
		}
		_ = json.Unmarshal([]byte(hs), &c.Hobbies)
		_ = json.Unmarshal([]byte(media), &c.Media)
		c.Media.normalize()
		_ = json.Unmarshal([]byte(prefs), &c.Preferences)
		if len(c.Hobbies) < 2 {
			continue
		}
		c.Score = similarity(u, c)
		c.Reasons = matchReasons(u, c)
		c.Liked = truth(liked)
		if avatarPath != "" {
			c.AvatarURL = avatarURL(c.ID, avatarPath)
		}
		result = append(result, c)
	}
	if err := rows.Err(); err != nil {
		fail(w, 500, "获取匹配失败")
		return
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Score == result[j].Score {
			return result[i].ID < result[j].ID
		}
		return result[i].Score > result[j].Score
	})
	if len(result) > 30 {
		result = result[:30]
	}
	send(w, 200, map[string]any{"candidates": result})
}
func (a *App) like(w http.ResponseWriter, r *http.Request) {
	u, ok := a.mustActive(w, r)
	if !ok {
		return
	}
	id := r.PathValue("id")
	if id == u.ID {
		fail(w, 400, "不能选择自己")
		return
	}
	target, err := a.getUser(id)
	if !u.MatchingEnabled || err != nil || !target.active() || !target.MatchingEnabled || target.Gender != u.Seeking || target.Seeking != u.Gender {
		fail(w, 403, "该用户不在你的匹配范围")
		return
	}
	tx, err := a.db.Begin()
	if err != nil {
		fail(w, 500, "操作失败")
		return
	}
	defer tx.Rollback()
	_, err = tx.Exec(`INSERT OR IGNORE INTO likes(from_id,to_id,created_at) VALUES(?,?,?)`, u.ID, id, time.Now().Unix())
	if err != nil {
		fail(w, 500, "操作失败")
		return
	}
	var mutual int
	if err := tx.QueryRow(`SELECT EXISTS(SELECT 1 FROM likes WHERE from_id=? AND to_id=?)`, id, u.ID).Scan(&mutual); err != nil {
		fail(w, 500, "操作失败")
		return
	}
	if truth(mutual) {
		if err := addMutualMatchNotifications(tx, u, target); err != nil {
			fail(w, 500, "操作失败")
			return
		}
	}
	if err := tx.Commit(); err != nil {
		fail(w, 500, "操作失败")
		return
	}
	send(w, 200, map[string]any{"mutual": truth(mutual)})
}
func (a *App) matches(w http.ResponseWriter, r *http.Request) {
	u, ok := a.mustActive(w, r)
	if !ok {
		return
	}
	rows, err := a.db.Query(`SELECT t.id,t.nickname,t.major,t.email,t.avatar_path FROM users t JOIN likes mine ON mine.to_id=t.id AND mine.from_id=? JOIN likes theirs ON theirs.from_id=t.id AND theirs.to_id=? WHERE t.email_verified=1 AND t.paid=1 AND t.card_status='approved' AND t.suspended_at=0 AND t.gender=? AND t.seeking=?`, u.ID, u.ID, u.Seeking, u.Gender)
	if err != nil {
		fail(w, 500, "获取失败")
		return
	}
	defer rows.Close()
	result := []map[string]string{}
	for rows.Next() {
		var id, nickname, major, email, avatarPath string
		if rows.Scan(&id, &nickname, &major, &email, &avatarPath) == nil {
			entry := map[string]string{"id": id, "nickname": nickname, "major": major, "email": email}
			if avatarPath != "" {
				entry["avatar_url"] = avatarURL(id, avatarPath)
			}
			result = append(result, entry)
		}
	}
	send(w, 200, map[string]any{"matches": result})
}
