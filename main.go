package main

import (
	"bufio"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"golang.org/x/crypto/bcrypt"
	_ "modernc.org/sqlite"
)

// Dynamische Variablen (werden aus .env oder Env-Vars gelesen)
var (
	mediaRoot string
	port      string
	dataDir   string
	db        *sql.DB
)

var translations = map[string]map[string]string{
	"de": {
		"nav.shows":                 "Serien",
		"nav.profile":               "Profil",
		"nav.admin":                 "Admin",
		"nav.logout":                "Logout",
		"home.title":                "Meine Serien",
		"home.empty":                "Keine Serien im Ordner gefunden.",
		"home.no_cover":             "Kein Cover",
		"show.back":                 "Zurück zur Übersicht",
		"show.no_cover":             "Kein Cover",
		"show.no_overview":          "Keine Beschreibung verfügbar.",
		"show.watched":              "Gesehen",
		"show.play":                 "Abspielen",
		"play.back":                 "Zurück zur Serie",
		"play.unsupported":          "Dein Browser unterstützt dieses Videoformat nicht.",
		"login.title":               "Anmeldung",
		"login.username":            "Benutzername",
		"login.password":            "Passwort",
		"login.submit":              "Einloggen",
		"login.error":               "Falscher Benutzername oder Passwort.",
		"profile.title":             "Mein Profil",
		"profile.logged_in_as":      "Angemeldet als",
		"profile.change_password":   "Passwort ändern",
		"profile.old_password":      "Aktuelles Passwort",
		"profile.new_password":      "Neues Passwort",
		"profile.confirm_password":  "Neues Passwort bestätigen",
		"profile.save_password":     "Passwort speichern",
		"profile.password_changed":  "Passwort erfolgreich geändert.",
		"profile.password_wrong":    "Aktuelles Passwort ist falsch.",
		"profile.password_mismatch": "Neue Passwörter stimmen nicht überein.",
		"profile.password_short":    "Neues Passwort muss mindestens 4 Zeichen haben.",
		"admin.title":               "Admin Panel",
		"admin.settings":            "Metadaten Einstellungen",
		"admin.language":            "Sprache der Oberfläche & Metadaten",
		"admin.tmdb_key":            "TMDB API-Key (optional)",
		"admin.tmdb_placeholder":    "Leer = TVMaze (kein Key nötig)",
		"admin.tmdb_help":           "Ohne Key: TVMaze (meist Englisch). Mit Key: u.a. deutsche Beschreibungen.",
		"admin.save_settings":       "Einstellungen speichern",
		"admin.create_user":         "Neuen Nutzer anlegen",
		"admin.create_user_btn":     "User erstellen",
		"admin.user_exists":         "Benutzername existiert bereits!",
		"admin.users":               "Registrierte Nutzer",
		"admin.role_admin":          "Admin",
	},
	"en": {
		"nav.shows":                 "Shows",
		"nav.profile":               "Profile",
		"nav.admin":                 "Admin",
		"nav.logout":                "Logout",
		"home.title":                "My Shows",
		"home.empty":                "No shows found in media folder.",
		"home.no_cover":             "No cover",
		"show.back":                 "Back to overview",
		"show.no_cover":             "No cover",
		"show.no_overview":          "No description available.",
		"show.watched":              "Watched",
		"show.play":                 "Play",
		"play.back":                 "Back to show",
		"play.unsupported":          "Your browser does not support this video format.",
		"login.title":               "Sign in",
		"login.username":            "Username",
		"login.password":            "Password",
		"login.submit":              "Log in",
		"login.error":               "Invalid username or password.",
		"profile.title":             "My Profile",
		"profile.logged_in_as":      "Signed in as",
		"profile.change_password":   "Change password",
		"profile.old_password":      "Current password",
		"profile.new_password":      "New password",
		"profile.confirm_password":  "Confirm new password",
		"profile.save_password":     "Save password",
		"profile.password_changed":  "Password changed successfully.",
		"profile.password_wrong":    "Current password is incorrect.",
		"profile.password_mismatch": "New passwords do not match.",
		"profile.password_short":    "New password must be at least 4 characters.",
		"admin.title":               "Admin Panel",
		"admin.settings":            "Metadata settings",
		"admin.language":            "UI & metadata language",
		"admin.tmdb_key":            "TMDB API key (optional)",
		"admin.tmdb_placeholder":    "Empty = TVMaze (no key needed)",
		"admin.tmdb_help":           "Without key: TVMaze (mostly English). With key: better localized descriptions.",
		"admin.save_settings":       "Save settings",
		"admin.create_user":         "Create new user",
		"admin.create_user_btn":     "Create user",
		"admin.user_exists":         "Username already exists!",
		"admin.users":               "Registered users",
		"admin.role_admin":          "Admin",
	},
}

func Translate(lang, key string) string {
	if m, ok := translations[lang]; ok {
		if v, ok := m[key]; ok {
			return v
		}
	}
	if v, ok := translations["en"][key]; ok {
		return v
	}
	return key
}

type User struct {
	Username string
	IsAdmin  bool
}

type Metadata struct {
	IMDBID    string `json:"imdb_id,omitempty"`
	TMDBID    int    `json:"tmdb_id,omitempty"`
	Name      string `json:"name"`
	Overview  string `json:"overview"`
	PosterURL string `json:"poster_url"`
	Language  string `json:"language,omitempty"`
}

type TVMazeImage struct {
	Original string `json:"original"`
}

type TVMazeExternals struct {
	IMDB string `json:"imdb"`
}

type TVMazeShow struct {
	Name      string          `json:"name"`
	Summary   string          `json:"summary"`
	Image     TVMazeImage     `json:"image"`
	Externals TVMazeExternals `json:"externals"`
}

type TMDBResult struct {
	ID         int    `json:"id"`
	Name       string `json:"name"`
	Overview   string `json:"overview"`
	PosterPath string `json:"poster_path"`
}

type TMDBSearchResponse struct {
	Results []TMDBResult `json:"results"`
}

type ShowItem struct {
	Name      string
	PosterURL string
}

type Episode struct {
	Name      string
	PlayPath  string
	IsWatched bool
}

type Season struct {
	Name     string
	Episodes []Episode
}

type ShowDetail struct {
	Name      string
	Overview  string
	PosterURL string
	Seasons   []Season
}

type PageData struct {
	Shows          []ShowItem
	User           *User
	AllUsers       []User
	Error, Success string
	ShowDetail     *ShowDetail
	PlayPath       string
	Language       string
	TMDBKey        string
}

func (d PageData) T(key string) string {
	return Translate(d.Language, key)
}

// Hilfsfunktion: Lädt eine .env Datei, falls vorhanden
func loadDotEnv() {
	file, err := os.Open(".env")
	if err != nil {
		return // Keine .env Datei vorhanden -> einfach überspringen
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			os.Setenv(strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]))
		}
	}
}

func getEnvOrDefault(key, defaultValue string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultValue
}

func main() {
	loadDotEnv()

	mediaRoot = getEnvOrDefault("WALLYFIN_MEDIA_ROOT", "./media")
	port = getEnvOrDefault("WALLYFIN_PORT", "8088")
	dataDir = getEnvOrDefault("WALLYFIN_DATA_DIR", "data")

	if !strings.HasPrefix(port, ":") {
		port = ":" + port
	}

	initDB()

	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	http.Handle("/media/", http.StripPrefix("/media/", http.FileServer(http.Dir(mediaRoot))))

	http.HandleFunc("/", handleIndex)
	http.HandleFunc("/login", handleLogin)
	http.HandleFunc("/logout", handleLogout)
	http.HandleFunc("/profile", handleProfile)
	http.HandleFunc("/profile/change-password", handleChangePassword)
	http.HandleFunc("/admin", handleAdmin)
	http.HandleFunc("/admin/settings", handleAdminSettings)
	http.HandleFunc("/show", handleShow)
	http.HandleFunc("/play", handlePlay)

	fmt.Printf("🚀 Wallyfin startet auf http://localhost%s\n", port)
	fmt.Printf("📁 Medienordner: %s\n", mediaRoot)
	log.Fatal(http.ListenAndServe(port, nil))
}

func initDB() {
	os.MkdirAll(dataDir, 0755)
	dbPath := filepath.Join(dataDir, "wallyfin.db")

	var err error
	db, err = sql.Open("sqlite", dbPath)
	if err != nil {
		log.Fatal(err)
	}

	db.Exec(`CREATE TABLE IF NOT EXISTS users (username TEXT PRIMARY KEY, password_hash TEXT, is_admin BOOLEAN)`)
	db.Exec(`CREATE TABLE IF NOT EXISTS sessions (token TEXT PRIMARY KEY, username TEXT)`)
	db.Exec(`CREATE TABLE IF NOT EXISTS watch_history (username TEXT, episode_path TEXT, PRIMARY KEY (username, episode_path))`)
	db.Exec(`CREATE TABLE IF NOT EXISTS settings (key TEXT PRIMARY KEY, value TEXT)`)

	setSettingIfMissing("language", "de")
	setSettingIfMissing("tmdb_key", "")

	var count int
	db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&count)
	if count == 0 {
		adminUser := getEnvOrDefault("WALLYFIN_ADMIN_USER", "admin")
		adminPass := getEnvOrDefault("WALLYFIN_ADMIN_PASSWORD", "admin")
		hash, _ := bcrypt.GenerateFromPassword([]byte(adminPass), bcrypt.DefaultCost)
		db.Exec(`INSERT INTO users (username, password_hash, is_admin) VALUES (?, ?, ?)`, adminUser, string(hash), true)
		fmt.Printf("⚠️ Admin erstellt: %s / %s\n", adminUser, adminPass)
	}
}

func setSettingIfMissing(key, value string) {
	var existing string
	if db.QueryRow(`SELECT value FROM settings WHERE key = ?`, key).Scan(&existing) == sql.ErrNoRows {
		db.Exec(`INSERT INTO settings (key, value) VALUES (?, ?)`, key, value)
	}
}

func getSetting(key string) string {
	var value string
	db.QueryRow(`SELECT value FROM settings WHERE key = ?`, key).Scan(&value)
	return value
}

func setSetting(key, value string) {
	db.Exec(`INSERT INTO settings (key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value`, key, value)
}

func getLang() string {
	lang := getSetting("language")
	if lang != "en" && lang != "de" {
		return "de"
	}
	return lang
}

func getUser(r *http.Request) *User {
	cookie, err := r.Cookie("session_token")
	if err != nil {
		return nil
	}
	var user User
	err = db.QueryRow(`SELECT u.username, u.is_admin FROM users u JOIN sessions s ON u.username = s.username WHERE s.token = ?`, cookie.Value).Scan(&user.Username, &user.IsAdmin)
	if err != nil {
		return nil
	}
	return &user
}

func hasWatched(username, path string) bool {
	var count int
	db.QueryRow(`SELECT COUNT(*) FROM watch_history WHERE username = ? AND episode_path = ?`, username, path).Scan(&count)
	return count > 0
}

func stripHTML(s string) string {
	return regexp.MustCompile(`<[^>]*>`).ReplaceAllString(s, "")
}

func cleanShowQuery(name string) string {
	return strings.TrimSpace(regexp.MustCompile(`\s*\(\d{4}\)`).ReplaceAllString(name, ""))
}

func baseData(user *User) PageData {
	return PageData{
		User:     user,
		Language: getLang(),
		TMDBKey:  getSetting("tmdb_key"),
	}
}

func autoScrapeShow(showName string) {
	showPath := filepath.Join(mediaRoot, showName)
	metaPath := filepath.Join(showPath, "metadata.json")
	posterPath := filepath.Join(showPath, "poster.jpg")
	lang := getLang()

	if metaBytes, err := os.ReadFile(metaPath); err == nil {
		var meta Metadata
		if json.Unmarshal(metaBytes, &meta) == nil {
			if _, err := os.Stat(posterPath); err == nil && meta.Overview != "" && (meta.Language == lang || meta.Language == "") {
				return
			}
		}
	}

	tmdbKey := strings.TrimSpace(getSetting("tmdb_key"))
	query := cleanShowQuery(showName)
	meta := Metadata{Language: lang}

	if tmdbKey != "" {
		tmdbLang := "en-US"
		if lang == "de" {
			tmdbLang = "de-DE"
		}
		searchURL := fmt.Sprintf("https://api.themoviedb.org/3/search/tv?api_key=%s&query=%s&language=%s", tmdbKey, url.QueryEscape(query), tmdbLang)
		if resp, err := http.Get(searchURL); err == nil && resp.StatusCode == 200 {
			var search TMDBSearchResponse
			json.NewDecoder(resp.Body).Decode(&search)
			resp.Body.Close()
			if len(search.Results) > 0 {
				item := search.Results[0]
				meta.TMDBID, meta.Name, meta.Overview = item.ID, item.Name, item.Overview
				if item.PosterPath != "" {
					imgURL := "https://image.tmdb.org/t/p/w500" + item.PosterPath
					if downloadPoster(imgURL, posterPath) {
						meta.PosterURL = "/media/" + url.PathEscape(showName) + "/poster.jpg"
					} else {
						meta.PosterURL = imgURL
					}
				}
				saveMetadata(metaPath, meta)
				return
			}
		}
	}

	tvURL := fmt.Sprintf("https://api.tvmaze.com/singlesearch/shows?q=%s", url.QueryEscape(query))
	if resp, err := http.Get(tvURL); err == nil && resp.StatusCode == 200 {
		var show TVMazeShow
		json.NewDecoder(resp.Body).Decode(&show)
		resp.Body.Close()
		meta.IMDBID, meta.Name, meta.Overview = show.Externals.IMDB, show.Name, stripHTML(show.Summary)
		if show.Image.Original != "" {
			if downloadPoster(show.Image.Original, posterPath) {
				meta.PosterURL = "/media/" + url.PathEscape(showName) + "/poster.jpg"
			} else {
				meta.PosterURL = show.Image.Original
			}
		}
		saveMetadata(metaPath, meta)
	}
}

func downloadPoster(imgURL, posterPath string) bool {
	resp, err := http.Get(imgURL)
	if err != nil || resp.StatusCode != 200 {
		if resp != nil {
			resp.Body.Close()
		}
		return false
	}
	defer resp.Body.Close()
	out, err := os.Create(posterPath)
	if err != nil {
		return false
	}
	defer out.Close()
	_, err = io.Copy(out, resp.Body)
	return err == nil
}

func saveMetadata(path string, meta Metadata) {
	b, _ := json.MarshalIndent(meta, "", "  ")
	_ = os.WriteFile(path, b, 0644)
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	user := getUser(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	data := baseData(user)
	entries, _ := os.ReadDir(mediaRoot)
	for _, e := range entries {
		if e.IsDir() && !strings.HasPrefix(e.Name(), ".") {
			name := e.Name()
			autoScrapeShow(name)
			posterURL := ""
			if _, err := os.Stat(filepath.Join(mediaRoot, name, "poster.jpg")); err == nil {
				posterURL = "/media/" + url.PathEscape(name) + "/poster.jpg"
			}
			data.Shows = append(data.Shows, ShowItem{Name: name, PosterURL: posterURL})
		}
	}
	render(w, "index.html", data)
}

func handleShow(w http.ResponseWriter, r *http.Request) {
	user := getUser(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	showName := r.URL.Query().Get("name")
	if showName == "" || strings.Contains(showName, "..") {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	autoScrapeShow(showName)
	showPath := filepath.Join(mediaRoot, showName)
	detail := &ShowDetail{Name: showName}
	if _, err := os.Stat(filepath.Join(showPath, "poster.jpg")); err == nil {
		detail.PosterURL = "/media/" + url.PathEscape(showName) + "/poster.jpg"
	}
	if b, err := os.ReadFile(filepath.Join(showPath, "metadata.json")); err == nil {
		var meta Metadata
		if json.Unmarshal(b, &meta) == nil {
			detail.Overview = meta.Overview
			if detail.PosterURL == "" {
				detail.PosterURL = meta.PosterURL
			}
			if meta.Name != "" {
				detail.Name = meta.Name
			}
		}
	}
	entries, _ := os.ReadDir(showPath)
	var rootEpisodes []Episode
	for _, e := range entries {
		if !e.IsDir() && (strings.HasSuffix(e.Name(), ".mkv") || strings.HasSuffix(e.Name(), ".mp4")) {
			epPath := showName + "/" + e.Name()
			rootEpisodes = append(rootEpisodes, Episode{Name: e.Name(), PlayPath: epPath, IsWatched: hasWatched(user.Username, epPath)})
		}
		if e.IsDir() && !strings.HasPrefix(e.Name(), ".") {
			season := Season{Name: e.Name()}
			eps, _ := os.ReadDir(filepath.Join(showPath, e.Name()))
			for _, ep := range eps {
				if !ep.IsDir() && (strings.HasSuffix(ep.Name(), ".mkv") || strings.HasSuffix(ep.Name(), ".mp4")) {
					epPath := showName + "/" + e.Name() + "/" + ep.Name()
					season.Episodes = append(season.Episodes, Episode{Name: ep.Name(), PlayPath: epPath, IsWatched: hasWatched(user.Username, epPath)})
				}
			}
			if len(season.Episodes) > 0 {
				detail.Seasons = append(detail.Seasons, season)
			}
		}
	}
	if len(rootEpisodes) > 0 {
		detail.Seasons = append([]Season{{Name: "Extra", Episodes: rootEpisodes}}, detail.Seasons...)
	}
	data := baseData(user)
	data.ShowDetail = detail
	render(w, "show.html", data)
}

func handlePlay(w http.ResponseWriter, r *http.Request) {
	user := getUser(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	file := r.URL.Query().Get("file")
	if file != "" {
		db.Exec(`INSERT OR IGNORE INTO watch_history (username, episode_path) VALUES (?, ?)`, user.Username, file)
	}
	data := baseData(user)
	data.PlayPath = file
	render(w, "play.html", data)
}

func handleProfile(w http.ResponseWriter, r *http.Request) {
	user := getUser(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	render(w, "profile.html", baseData(user))
}

func handleChangePassword(w http.ResponseWriter, r *http.Request) {
	user := getUser(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	data := baseData(user)
	if r.Method != "POST" {
		http.Redirect(w, r, "/profile", http.StatusSeeOther)
		return
	}

	oldPass := r.FormValue("old_password")
	newPass := r.FormValue("new_password")
	confirm := r.FormValue("confirm_password")

	if len(newPass) < 4 {
		data.Error = data.T("profile.password_short")
		render(w, "profile.html", data)
		return
	}
	if newPass != confirm {
		data.Error = data.T("profile.password_mismatch")
		render(w, "profile.html", data)
		return
	}

	var currentHash string
	err := db.QueryRow(`SELECT password_hash FROM users WHERE username = ?`, user.Username).Scan(&currentHash)
	if err != nil || bcrypt.CompareHashAndPassword([]byte(currentHash), []byte(oldPass)) != nil {
		data.Error = data.T("profile.password_wrong")
		render(w, "profile.html", data)
		return
	}

	newHash, _ := bcrypt.GenerateFromPassword([]byte(newPass), bcrypt.DefaultCost)
	db.Exec(`UPDATE users SET password_hash = ? WHERE username = ?`, string(newHash), user.Username)

	data.Success = data.T("profile.password_changed")
	render(w, "profile.html", data)
}

func handleAdmin(w http.ResponseWriter, r *http.Request) {
	user := getUser(r)
	if user == nil || !user.IsAdmin {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	data := baseData(user)
	if r.Method == "POST" {
		hash, _ := bcrypt.GenerateFromPassword([]byte(r.FormValue("password")), bcrypt.DefaultCost)
		if _, err := db.Exec(`INSERT INTO users (username, password_hash, is_admin) VALUES (?, ?, ?)`, r.FormValue("username"), string(hash), false); err != nil {
			data.Error = data.T("admin.user_exists")
		}
	}
	rows, _ := db.Query(`SELECT username, is_admin FROM users`)
	defer rows.Close()
	for rows.Next() {
		var u User
		rows.Scan(&u.Username, &u.IsAdmin)
		data.AllUsers = append(data.AllUsers, u)
	}
	render(w, "admin.html", data)
}

func handleAdminSettings(w http.ResponseWriter, r *http.Request) {
	user := getUser(r)
	if user == nil || !user.IsAdmin {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	if r.Method == "POST" {
		lang := r.FormValue("language")
		if lang != "de" && lang != "en" {
			lang = "de"
		}
		setSetting("language", lang)
		setSetting("tmdb_key", strings.TrimSpace(r.FormValue("tmdb_key")))
	}
	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

func handleLogin(w http.ResponseWriter, r *http.Request) {
	data := baseData(nil)
	if r.Method == "POST" {
		username, password := r.FormValue("username"), r.FormValue("password")
		var hash string
		err := db.QueryRow(`SELECT password_hash FROM users WHERE username = ?`, username).Scan(&hash)
		if err == nil && bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil {
			b := make([]byte, 32)
			rand.Read(b)
			token := hex.EncodeToString(b)
			db.Exec(`INSERT INTO sessions (token, username) VALUES (?, ?)`, token, username)
			http.SetCookie(w, &http.Cookie{Name: "session_token", Value: token, Path: "/"})
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		data.Error = data.T("login.error")
	}
	render(w, "login.html", data)
}

func handleLogout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie("session_token"); err == nil {
		db.Exec(`DELETE FROM sessions WHERE token = ?`, cookie.Value)
		http.SetCookie(w, &http.Cookie{Name: "session_token", Value: "", MaxAge: -1, Path: "/"})
	}
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func render(w http.ResponseWriter, templateName string, data PageData) {
	tmpl, err := template.ParseFiles("templates/layout.html", "templates/"+templateName)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	if err := tmpl.ExecuteTemplate(w, "layout.html", data); err != nil {
		http.Error(w, err.Error(), 500)
	}
}
