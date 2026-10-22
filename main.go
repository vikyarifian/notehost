package main

import (
	"bytes"
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/yuin/goldmark"
)

type Note struct {
	ID              int
	Title           string
	Departemen      string
	Tags            string
	ContentMarkdown string
	ContentHTML     template.HTML
	EditPIN         string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func (n Note) FormattedDate() string {
	return n.UpdatedAt.Format("02 Jan 2006 15:04")
}

func (n Note) TagList() []string {
	if strings.TrimSpace(n.Tags) == "" {
		return nil
	}
	parts := strings.Split(n.Tags, ",")
	var result []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

func (n Note) IsPINProtected() bool {
	return strings.TrimSpace(n.EditPIN) != ""
}

var db *sql.DB
var tmpl *template.Template

func renderMarkdown(md string) template.HTML {
	var buf bytes.Buffer
	if err := goldmark.Convert([]byte(md), &buf); err != nil {
		return template.HTML(template.HTMLEscapeString(md))
	}
	// html/template limitation: raw markdown output requires explicit conversion to template.HTML
	return template.HTML(buf.String())
}

func main() {
	var err error
	dbPath := os.Getenv("DATABASE_PATH")
	if dbPath == "" {
		dbPath = "./notehost.db"
	}

	db, err = initDB(dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	initTemplates()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /", handleHome)
	mux.HandleFunc("GET /notes", handleNotesSearch)
	mux.HandleFunc("GET /notes/new", handleNoteNew)
	mux.HandleFunc("POST /notes", handleNoteCreate)
	mux.HandleFunc("POST /notes/preview", handlePreview)
	mux.HandleFunc("GET /notes/{id}", handleNoteView)
	mux.HandleFunc("GET /notes/{id}/edit", handleNoteEdit)
	mux.HandleFunc("POST /notes/{id}/verify-pin", handleVerifyPIN)
	mux.HandleFunc("POST /notes/{id}", handleNoteUpdate)
	mux.HandleFunc("DELETE /notes/{id}", handleNoteDelete)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("NoteHost listening on port %s...", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatal(err)
	}
}

func initDB(dbPath string) (*sql.DB, error) {
	database, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL")
	if err != nil {
		return nil, err
	}

	database.SetMaxOpenConns(1)

	schema := `
	PRAGMA journal_mode=WAL;

	CREATE TABLE IF NOT EXISTS notes (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		title TEXT NOT NULL,
		departemen TEXT NOT NULL,
		tags TEXT NOT NULL DEFAULT '',
		content_markdown TEXT NOT NULL,
		edit_pin TEXT NOT NULL DEFAULT '',
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

	CREATE VIRTUAL TABLE IF NOT EXISTS notes_fts USING fts5(
		title,
		tags,
		departemen,
		content_markdown,
		content='notes',
		content_rowid='id'
	);

	CREATE TRIGGER IF NOT EXISTS notes_ai AFTER INSERT ON notes BEGIN
		INSERT INTO notes_fts(rowid, title, tags, departemen, content_markdown)
		VALUES (new.id, new.title, new.tags, new.departemen, new.content_markdown);
	END;

	CREATE TRIGGER IF NOT EXISTS notes_ad AFTER DELETE ON notes BEGIN
		INSERT INTO notes_fts(notes_fts, rowid, title, tags, departemen, content_markdown)
		VALUES('delete', old.id, old.title, old.tags, old.departemen, old.content_markdown);
	END;

	CREATE TRIGGER IF NOT EXISTS notes_au AFTER UPDATE ON notes BEGIN
		INSERT INTO notes_fts(notes_fts, rowid, title, tags, departemen, content_markdown)
		VALUES('delete', old.id, old.title, old.tags, old.departemen, old.content_markdown);
		INSERT INTO notes_fts(rowid, title, tags, departemen, content_markdown)
		VALUES (new.id, new.title, new.tags, new.departemen, new.content_markdown);
	END;
	`

	_, err = database.Exec(schema)
	if err != nil {
		return nil, fmt.Errorf("exec schema: %w", err)
	}

	seedInitialData(database)
	return database, nil
}

func seedInitialData(database *sql.DB) {
	var count int
	_ = database.QueryRow("SELECT COUNT(*) FROM notes").Scan(&count)
	if count > 0 {
		return
	}

	seeds := []struct {
		title, dept, tags, content, pin string
	}{
		{
			title:      "Panduan Setup Laptop Staf Baru IT",
			dept:       "IT",
			tags:       "onboarding, laptop, setup",
			content:    "# Onboarding Staf IT\n\nInstruksi setup laptop staf baru:\n\n1. Install Ubuntu 24.04 LTS / Windows 11\n2. Konfigurasi VPN Kantor & Email Google Workspace\n3. Install Go, SQLite, Docker, & VSCode\n4. Daftarkan IP ke firewall server utama\n\nHubungi Tim IT Infra jika ada masalah access permission.",
			pin:        "1234",
		},
		{
			title:      "Prosedur Pengajuan Surat Jalan Operasional",
			dept:       "Operasional",
			tags:       "surat_jalan, logistik, pengiriman",
			content:    "# Prosedur Surat Jalan Operasional\n\nSetiap pengiriman unit kerja ke cabang wajib melampirkan **Surat Jalan** resmi.\n\n### Langkah Pembuatan:\n- Isi formulir barang & nomor serial\n- Minta verifikasi dari Kepala Operasional\n- Cetak 3 rangkap (Driver, Penerima, Arsip)",
			pin:        "",
		},
		{
			title:      "Ketentuan Lembur & Perhitungan PPh21 HR",
			dept:       "HR",
			tags:       "lembur, pph21, absensi, gaji",
			content:    "# Ketentuan Lembur & PPh21 HR\n\nPengajuan lembur wajib dicatat pada sistem absensi sebelum jam 17:00 WIB.\n\n- Batas maksimal lembur: 4 jam / hari kerja\n- Perhitungan pph21 mengacu pada aturan TER terbaru\n- Lampirkan bukti foto presensi fingerprint atau aplikasi mobile",
			pin:        "5678",
		},
	}

	for _, s := range seeds {
		_, _ = database.Exec(`INSERT INTO notes (title, departemen, tags, content_markdown, edit_pin) VALUES (?, ?, ?, ?, ?)`, s.title, s.dept, s.tags, s.content, s.pin)
	}
}

func fetchNotes(q, departemen, tag string) ([]Note, error) {
	var query string
	var args []interface{}

	q = strings.TrimSpace(q)
	departemen = strings.TrimSpace(departemen)
	tag = strings.TrimSpace(tag)

	if q != "" {
		ftsQuery := strings.ReplaceAll(q, "\"", "\"\"")
		ftsQuery = "\"" + ftsQuery + "\"*"

		query = `SELECT n.id, n.title, n.departemen, n.tags, n.content_markdown, n.edit_pin, n.created_at, n.updated_at
				 FROM notes n
				 JOIN notes_fts fts ON n.id = fts.rowid
				 WHERE fts MATCH ?`
		args = append(args, ftsQuery)

		if departemen != "" {
			query += " AND n.departemen = ?"
			args = append(args, departemen)
		}
		if tag != "" {
			query += " AND n.tags LIKE ?"
			args = append(args, "%"+tag+"%")
		}
		query += " ORDER BY n.updated_at DESC"
	} else {
		query = `SELECT id, title, departemen, tags, content_markdown, edit_pin, created_at, updated_at
				 FROM notes WHERE 1=1`
		if departemen != "" {
			query += " AND departemen = ?"
			args = append(args, departemen)
		}
		if tag != "" {
			query += " AND tags LIKE ?"
			args = append(args, "%"+tag+"%")
		}
		query += " ORDER BY updated_at DESC"
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []Note
	for rows.Next() {
		var n Note
		err := rows.Scan(&n.ID, &n.Title, &n.Departemen, &n.Tags, &n.ContentMarkdown, &n.EditPIN, &n.CreatedAt, &n.UpdatedAt)
		if err != nil {
			return nil, err
		}
		n.ContentHTML = renderMarkdown(n.ContentMarkdown)
		list = append(list, n)
	}
	return list, nil
}

func getNoteByID(id int) (*Note, error) {
	row := db.QueryRow("SELECT id, title, departemen, tags, content_markdown, edit_pin, created_at, updated_at FROM notes WHERE id = ?", id)
	var n Note
	err := row.Scan(&n.ID, &n.Title, &n.Departemen, &n.Tags, &n.ContentMarkdown, &n.EditPIN, &n.CreatedAt, &n.UpdatedAt)
	if err != nil {
		return nil, err
	}
	n.ContentHTML = renderMarkdown(n.ContentMarkdown)
	return &n, nil
}

func checkPINAuth(r *http.Request, note *Note) bool {
	if !note.IsPINProtected() {
		return true
	}
	pinParam := r.FormValue("pin")
	if pinParam != "" && pinParam == note.EditPIN {
		return true
	}
	cookie, err := r.Cookie(fmt.Sprintf("note_pin_%d", note.ID))
	if err == nil && cookie.Value == note.EditPIN {
		return true
	}
	return false
}

func handleHome(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	q := r.URL.Query().Get("q")
	dept := r.URL.Query().Get("departemen")
	tag := r.URL.Query().Get("tag")

	notes, err := fetchNotes(q, dept, tag)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Notes":      notes,
		"Query":      q,
		"Departemen": dept,
		"Tag":        tag,
	}
	tmpl.ExecuteTemplate(w, "index", data)
}

func handleNotesSearch(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	dept := r.URL.Query().Get("departemen")
	tag := r.URL.Query().Get("tag")

	notes, err := fetchNotes(q, dept, tag)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Notes":      notes,
		"Query":      q,
		"Departemen": dept,
		"Tag":        tag,
	}

	if r.Header.Get("HX-Request") == "true" {
		tmpl.ExecuteTemplate(w, "_note_list.html", data)
		return
	}
	tmpl.ExecuteTemplate(w, "index", data)
}

func handleNoteNew(w http.ResponseWriter, r *http.Request) {
	tmpl.ExecuteTemplate(w, "new_note", nil)
}

func handleNoteCreate(w http.ResponseWriter, r *http.Request) {
	title := r.FormValue("title")
	dept := r.FormValue("departemen")
	tags := r.FormValue("tags")
	content := r.FormValue("content_markdown")
	pin := strings.TrimSpace(r.FormValue("edit_pin"))

	res, err := db.Exec("INSERT INTO notes (title, departemen, tags, content_markdown, edit_pin) VALUES (?, ?, ?, ?, ?)",
		title, dept, tags, content, pin)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	id, _ := res.LastInsertId()
	redirectURL := fmt.Sprintf("/notes/%d", id)

	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Redirect", redirectURL)
		w.WriteHeader(http.StatusOK)
		return
	}
	http.Redirect(w, r, redirectURL, http.StatusSeeOther)
}

func handlePreview(w http.ResponseWriter, r *http.Request) {
	content := r.FormValue("content_markdown")
	html := renderMarkdown(content)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(html))
}

func handleNoteView(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	note, err := getNoteByID(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	data := map[string]interface{}{
		"Note": note,
	}
	tmpl.ExecuteTemplate(w, "view_note", data)
}

func handleNoteEdit(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	note, err := getNoteByID(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	if !checkPINAuth(r, note) {
		tmpl.ExecuteTemplate(w, "pin_prompt", map[string]interface{}{
			"NoteID": note.ID,
			"Error":  "",
		})
		return
	}

	tmpl.ExecuteTemplate(w, "edit_note", map[string]interface{}{
		"Note":  note,
		"Error": "",
	})
}

func handleVerifyPIN(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	note, err := getNoteByID(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	inputPIN := r.FormValue("pin")
	if inputPIN == note.EditPIN {
		http.SetCookie(w, &http.Cookie{
			Name:     fmt.Sprintf("note_pin_%d", note.ID),
			Value:    inputPIN,
			Path:     "/",
			HttpOnly: true,
			MaxAge:   3600,
		})

		if r.Header.Get("HX-Request") == "true" {
			tmpl.ExecuteTemplate(w, "edit_note", map[string]interface{}{
				"Note":  note,
				"Error": "",
			})
			return
		}
		http.Redirect(w, r, fmt.Sprintf("/notes/%d/edit", note.ID), http.StatusSeeOther)
		return
	}

	tmpl.ExecuteTemplate(w, "pin_prompt", map[string]interface{}{
		"NoteID": note.ID,
		"Error":  "PIN salah, silakan coba lagi.",
	})
}

func handleNoteUpdate(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	note, err := getNoteByID(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	if !checkPINAuth(r, note) {
		http.Error(w, "Unauthorized PIN", http.StatusForbidden)
		return
	}

	title := r.FormValue("title")
	dept := r.FormValue("departemen")
	tags := r.FormValue("tags")
	content := r.FormValue("content_markdown")
	newPIN := strings.TrimSpace(r.FormValue("edit_pin"))

	_, err = db.Exec(`UPDATE notes SET title = ?, departemen = ?, tags = ?, content_markdown = ?, edit_pin = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		title, dept, tags, content, newPIN, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	redirectURL := fmt.Sprintf("/notes/%d", id)
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Redirect", redirectURL)
		w.WriteHeader(http.StatusOK)
		return
	}
	http.Redirect(w, r, redirectURL, http.StatusSeeOther)
}

func handleNoteDelete(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	note, err := getNoteByID(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	if !checkPINAuth(r, note) {
		http.Error(w, "PIN required to delete this note", http.StatusForbidden)
		return
	}

	_, err = db.Exec("DELETE FROM notes WHERE id = ?", id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Redirect", "/")
		w.WriteHeader(http.StatusOK)
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func initTemplates() {
	funcMap := template.FuncMap{
		"renderMD": renderMarkdown,
	}
	tmpl = template.Must(template.New("root").Funcs(funcMap).Parse(templateSource))
}

const templateSource = `
{{define "header"}}
<!DOCTYPE html>
<html lang="id">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>NoteHost - Internal Docs & Guides</title>
    <script src="https://unpkg.com/htmx.org@1.9.10"></script>
    <style>
        :root {
            --bg-color: #f8fafc;
            --text-color: #0f172a;
            --card-bg: #ffffff;
            --border-color: #e2e8f0;
            --primary: #2563eb;
            --primary-hover: #1d4ed8;
            --danger: #dc2626;
            --dept-it: #dbeafe;
            --dept-it-text: #1e40af;
            --dept-hr: #f3e8ff;
            --dept-hr-text: #6b21a8;
            --dept-op: #dcfce7;
            --dept-op-text: #166534;
            --dept-keu: #fef3c7;
            --dept-keu-text: #92400e;
        }
        * { box-sizing: border-box; margin: 0; padding: 0; }
        body { font-family: system-ui, -apple-system, sans-serif; background: var(--bg-color); color: var(--text-color); line-height: 1.5; }
        header { background: #1e293b; color: white; padding: 1rem 2rem; display: flex; justify-content: space-between; align-items: center; }
        header h1 { font-size: 1.3rem; font-weight: 600; }
        header h1 a { color: white; text-decoration: none; }
        .container { max-width: 1100px; margin: 2rem auto; padding: 0 1rem; }
        .nav-btn { background: var(--primary); color: white; padding: 0.5rem 1rem; border-radius: 6px; text-decoration: none; font-size: 0.9rem; border: none; cursor: pointer; display: inline-block; }
        .nav-btn:hover { background: var(--primary-hover); }
        .btn-danger { background: var(--danger); }
        .btn-secondary { background: #64748b; }
        .search-panel { background: var(--card-bg); padding: 1rem; border-radius: 8px; border: 1px solid var(--border-color); margin-bottom: 1.5rem; display: flex; gap: 0.75rem; flex-wrap: wrap; }
        .search-panel input, .search-panel select { padding: 0.6rem; border: 1px solid var(--border-color); border-radius: 6px; font-size: 0.9rem; }
        .search-panel input[type="text"] { flex: 1; min-width: 180px; }
        .notes-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(320px, 1fr)); gap: 1.25rem; }
        .note-card { background: var(--card-bg); border: 1px solid var(--border-color); border-radius: 8px; padding: 1.25rem; display: flex; flex-direction: column; justify-content: space-between; }
        .note-card h3 { font-size: 1.1rem; margin-bottom: 0.5rem; }
        .note-card h3 a { color: var(--text-color); text-decoration: none; }
        .note-card h3 a:hover { color: var(--primary); }
        .meta-tags { display: flex; gap: 0.4rem; flex-wrap: wrap; margin-bottom: 0.75rem; align-items: center; }
        .badge { font-size: 0.7rem; font-weight: 700; padding: 0.2rem 0.5rem; border-radius: 4px; text-transform: uppercase; }
        .badge-IT { background: var(--dept-it); color: var(--dept-it-text); }
        .badge-HR { background: var(--dept-hr); color: var(--dept-hr-text); }
        .badge-Operasional { background: var(--dept-op); color: var(--dept-op-text); }
        .badge-Keuangan { background: var(--dept-keu); color: var(--dept-keu-text); }
        .tag-pill { background: #f1f5f9; color: #475569; font-size: 0.75rem; padding: 0.15rem 0.4rem; border-radius: 4px; }
        .card-footer { font-size: 0.8rem; color: #64748b; display: flex; justify-content: space-between; margin-top: 1rem; padding-top: 0.75rem; border-top: 1px solid var(--border-color); }
        .editor-container { display: grid; grid-template-columns: 1fr 1fr; gap: 1.5rem; margin-top: 1rem; }
        @media (max-width: 768px) { .editor-container { grid-template-columns: 1fr; } }
        .form-group { margin-bottom: 1rem; }
        .form-group label { display: block; font-weight: 600; font-size: 0.85rem; margin-bottom: 0.3rem; }
        .form-group input, .form-group select, .form-group textarea { width: 100%; padding: 0.6rem; border: 1px solid var(--border-color); border-radius: 6px; font-size: 0.95rem; }
        .form-group textarea { height: 350px; font-family: monospace; }
        .markdown-body { background: white; border: 1px solid var(--border-color); border-radius: 6px; padding: 1.25rem; min-height: 350px; }
        .markdown-body h1, .markdown-body h2, .markdown-body h3 { margin-bottom: 0.5rem; margin-top: 1rem; }
        .markdown-body ul, .markdown-body ol { margin-left: 1.5rem; margin-bottom: 0.5rem; }
        .markdown-body p { margin-bottom: 0.75rem; }
        .markdown-body code { background: #f1f5f9; padding: 0.1rem 0.3rem; border-radius: 3px; font-family: monospace; font-size: 0.9em; }
        .alert { background: #fee2e2; border: 1px solid #fca5a5; color: #991b1b; padding: 0.75rem 1rem; border-radius: 6px; margin-bottom: 1rem; }
        .lock-icon { font-size: 0.8rem; color: #b45309; margin-left: auto; }
        .pin-card { max-width: 400px; margin: 3rem auto; background: white; padding: 2rem; border-radius: 8px; border: 1px solid var(--border-color); box-shadow: 0 4px 12px rgba(0,0,0,0.05); }
    </style>
</head>
<body>
    <header>
        <h1><a href="/">NoteHost Docs Internal</a></h1>
        <div>
            <a href="/notes/new" class="nav-btn">+ Tambah Panduan Baru</a>
        </div>
    </header>
    <div class="container" id="page-body">
{{end}}

{{define "footer"}}
    </div>
</body>
</html>
{{end}}

{{define "index"}}
{{template "header" .}}
<div class="search-panel">
    <input type="text" name="q" placeholder="Cari judul, kata kunci, atau isi panduan..." value="{{.Query}}"
           hx-get="/notes" hx-target="#search-results" hx-trigger="keyup changed delay:200ms" hx-include="[name='departemen'], [name='tag']">
    <select name="departemen" hx-get="/notes" hx-target="#search-results" hx-trigger="change" hx-include="[name='q'], [name='tag']">
        <option value="">Semua Departemen</option>
        <option value="IT" {{if eq .Departemen "IT"}}selected{{end}}>IT</option>
        <option value="HR" {{if eq .Departemen "HR"}}selected{{end}}>HR</option>
        <option value="Operasional" {{if eq .Departemen "Operasional"}}selected{{end}}>Operasional</option>
        <option value="Keuangan" {{if eq .Departemen "Keuangan"}}selected{{end}}>Keuangan</option>
    </select>
    <input type="text" name="tag" placeholder="Filter tag (misal: absensi)" value="{{.Tag}}"
           hx-get="/notes" hx-target="#search-results" hx-trigger="keyup changed delay:200ms" hx-include="[name='q'], [name='departemen']">
</div>

<div id="search-results">
    {{template "_note_list.html" .}}
</div>
{{template "footer" .}}
{{end}}

{{define "_note_list.html"}}
{{if .Notes}}
<div class="notes-grid">
    {{range .Notes}}
    <div class="note-card">
        <div>
            <div class="meta-tags">
                <span class="badge badge-{{.Departemen}}">{{.Departemen}}</span>
                {{range .TagList}}
                <span class="tag-pill">#{{.}}</span>
                {{end}}
                {{if .IsPINProtected}}
                <span class="lock-icon" title="Dilindungi PIN">🔒 PIN</span>
                {{end}}
            </div>
            <h3><a href="/notes/{{.ID}}">{{.Title}}</a></h3>
        </div>
        <div class="card-footer">
            <span>Update: {{.FormattedDate}}</span>
            <a href="/notes/{{.ID}}" style="color: var(--primary); text-decoration: none; font-weight: 600;">Lihat &rarr;</a>
        </div>
    </div>
    {{end}}
</div>
{{else}}
<div style="text-align: center; padding: 3rem; background: white; border-radius: 8px; color: #64748b;">
    <p>Tidak ada panduan atau dokumentasi yang ditemukan.</p>
</div>
{{end}}
{{end}}

{{define "new_note"}}
{{template "header" .}}
<h2 style="margin-bottom: 1rem;">Buat Operational Guide Baru</h2>
<form action="/notes" method="POST">
    <div style="display: grid; grid-template-columns: 2fr 1fr 1fr; gap: 1rem;">
        <div class="form-group">
            <label>Judul Dokumentasi / Panduan</label>
            <input type="text" name="title" required placeholder="Contoh: Setup Laptop & VPN Staf Baru">
        </div>
        <div class="form-group">
            <label>Departemen</label>
            <select name="departemen" required>
                <option value="IT">IT</option>
                <option value="HR">HR</option>
                <option value="Operasional">Operasional</option>
                <option value="Keuangan">Keuangan</option>
            </select>
        </div>
        <div class="form-group">
            <label>Edit PIN (Opsional)</label>
            <input type="password" name="edit_pin" placeholder="4-6 angka" pattern="[0-9]*" maxlength="6">
        </div>
    </div>
    <div class="form-group">
        <label>Tags (pisahkan dengan koma)</label>
        <input type="text" name="tags" placeholder="onboarding, surat_jalan, absensi">
    </div>

    <div class="editor-container">
        <div class="form-group">
            <label>Konten (Markdown Format)</label>
            <textarea name="content_markdown" required placeholder="# Judul&#10;&#10;Tulis instruksi detail di sini..."
                      hx-post="/notes/preview" hx-target="#preview" hx-trigger="keyup changed delay:200ms, load"></textarea>
        </div>
        <div>
            <label style="display: block; font-weight: 600; font-size: 0.85rem; margin-bottom: 0.3rem;">Live Render Preview</label>
            <div id="preview" class="markdown-body">
                <em>Hasil preview markdown akan tampil di sini secara realtime...</em>
            </div>
        </div>
    </div>

    <div style="margin-top: 1.5rem; display: flex; gap: 1rem;">
        <button type="submit" class="nav-btn">Simpan Dokumentasi</button>
        <a href="/" class="nav-btn btn-secondary">Batal</a>
    </div>
</form>
{{template "footer" .}}
{{end}}

{{define "view_note"}}
{{template "header" .}}
<div style="background: white; border: 1px solid var(--border-color); border-radius: 8px; padding: 2rem;">
    <div style="display: flex; justify-content: space-between; align-items: flex-start; margin-bottom: 1.5rem; border-bottom: 1px solid var(--border-color); padding-bottom: 1rem;">
        <div>
            <div class="meta-tags" style="margin-bottom: 0.5rem;">
                <span class="badge badge-{{.Note.Departemen}}">{{.Note.Departemen}}</span>
                {{range .Note.TagList}}
                <span class="tag-pill">#{{.}}</span>
                {{end}}
                {{if .Note.IsPINProtected}}
                <span class="lock-icon">🔒 Protected PIN</span>
                {{end}}
            </div>
            <h1 style="font-size: 1.75rem;">{{.Note.Title}}</h1>
            <p style="font-size: 0.85rem; color: #64748b; margin-top: 0.25rem;">Terakhir diperbarui: {{.Note.FormattedDate}}</p>
        </div>
        <div style="display: flex; gap: 0.5rem;">
            <a href="/notes/{{.Note.ID}}/edit" class="nav-btn btn-secondary">Edit</a>
            <button class="nav-btn btn-danger" hx-delete="/notes/{{.Note.ID}}" hx-confirm="Yakin ingin menghapus dokumentasi ini?" hx-target="body">Hapus</button>
        </div>
    </div>

    <div class="markdown-body" style="border: none; padding: 0;">
        {{.Note.ContentHTML}}
    </div>
</div>
{{template "footer" .}}
{{end}}

{{define "edit_note"}}
{{template "header" .}}
<h2 style="margin-bottom: 1rem;">Edit Panduan / Catatan</h2>
{{if .Error}}
<div class="alert">{{.Error}}</div>
{{end}}
<form action="/notes/{{.Note.ID}}" method="POST">
    <div style="display: grid; grid-template-columns: 2fr 1fr 1fr; gap: 1rem;">
        <div class="form-group">
            <label>Judul Dokumentasi / Panduan</label>
            <input type="text" name="title" value="{{.Note.Title}}" required>
        </div>
        <div class="form-group">
            <label>Departemen</label>
            <select name="departemen" required>
                <option value="IT" {{if eq .Note.Departemen "IT"}}selected{{end}}>IT</option>
                <option value="HR" {{if eq .Note.Departemen "HR"}}selected{{end}}>HR</option>
                <option value="Operasional" {{if eq .Note.Departemen "Operasional"}}selected{{end}}>Operasional</option>
                <option value="Keuangan" {{if eq .Note.Departemen "Keuangan"}}selected{{end}}>Keuangan</option>
            </select>
        </div>
        <div class="form-group">
            <label>Edit PIN (Kosongkan jika ingin menghapus)</label>
            <input type="password" name="edit_pin" value="{{.Note.EditPIN}}" pattern="[0-9]*" maxlength="6">
        </div>
    </div>
    <div class="form-group">
        <label>Tags (pisahkan dengan koma)</label>
        <input type="text" name="tags" value="{{.Note.Tags}}">
    </div>

    <div class="editor-container">
        <div class="form-group">
            <label>Konten (Markdown Format)</label>
            <textarea name="content_markdown" required
                      hx-post="/notes/preview" hx-target="#preview" hx-trigger="keyup changed delay:200ms, load">{{.Note.ContentMarkdown}}</textarea>
        </div>
        <div>
            <label style="display: block; font-weight: 600; font-size: 0.85rem; margin-bottom: 0.3rem;">Live Render Preview</label>
            <div id="preview" class="markdown-body">
                {{.Note.ContentHTML}}
            </div>
        </div>
    </div>

    <div style="margin-top: 1.5rem; display: flex; gap: 1rem;">
        <button type="submit" class="nav-btn">Simpan Perubahan</button>
        <a href="/notes/{{.Note.ID}}" class="nav-btn btn-secondary">Batal</a>
    </div>
</form>
{{template "footer" .}}
{{end}}

{{define "pin_prompt"}}
{{template "header" .}}
<div class="pin-card">
    <h3 style="margin-bottom: 0.5rem; text-align: center;">🔒 Otentikasi PIN Edit</h3>
    <p style="font-size: 0.85rem; color: #64748b; text-align: center; margin-bottom: 1.5rem;">
        Catatan ini dilindungi oleh PIN. Masukkan PIN untuk melanjutkan pengeditan atau penghapusan.
    </p>

    {{if .Error}}
    <div class="alert" style="font-size: 0.85rem;">{{.Error}}</div>
    {{end}}

    <form action="/notes/{{.NoteID}}/verify-pin" method="POST">
        <div class="form-group">
            <label style="text-align: center;">Masukkan 4-6 Digit PIN</label>
            <input type="password" name="pin" required autofocus pattern="[0-9]*" maxlength="6" style="text-align: center; letter-spacing: 0.3em; font-size: 1.2rem;">
        </div>
        <div style="margin-top: 1.5rem; display: flex; gap: 0.5rem;">
            <button type="submit" class="nav-btn" style="width: 100%;">Verifikasi</button>
            <a href="/notes/{{.NoteID}}" class="nav-btn btn-secondary" style="text-align: center; width: 100%;">Batal</a>
        </div>
    </form>
</div>
{{template "footer" .}}
{{end}}
`
