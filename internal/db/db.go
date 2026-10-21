package db

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type Note struct {
	ID              int
	Title           string
	Departemen      string
	Tags            string
	ContentMarkdown string
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

func InitDB(dbPath string) (*sql.DB, error) {
	database, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL")
	if err != nil {
		return nil, err
	}

	database.SetMaxOpenConns(1)

	// sqlite3 limitation: external content tables require manual triggers for full-text search sync
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

	SeedInitialData(database)
	return database, nil
}

func SeedInitialData(database *sql.DB) {
	var count int
	_ = database.QueryRow("SELECT COUNT(*) FROM notes").Scan(&count)
	if count > 0 {
		return
	}

	seeds := []struct {
		title, dept, tags, content, pin string
	}{
		{
			title:   "Panduan Setup Laptop Staf Baru IT",
			dept:    "IT",
			tags:    "onboarding, laptop, setup",
			content: "# Onboarding Staf IT\n\nInstruksi setup laptop staf baru:\n\n1. Install Ubuntu 24.04 LTS / Windows 11\n2. Konfigurasi VPN Kantor & Email Google Workspace\n3. Install Go, SQLite, Docker, & VSCode\n4. Daftarkan IP ke firewall server utama\n\nHubungi Tim IT Infra jika ada masalah access permission.",
			pin:     "1234",
		},
		{
			title:   "Prosedur Pengajuan Surat Jalan Operasional",
			dept:    "Operasional",
			tags:    "surat_jalan, logistik, pengiriman",
			content: "# Prosedur Surat Jalan Operasional\n\nSetiap pengiriman unit kerja ke cabang wajib melampirkan **Surat Jalan** resmi.\n\n### Langkah Pembuatan:\n- Isi formulir barang & nomor serial\n- Minta verifikasi dari Kepala Operasional\n- Cetak 3 rangkap (Driver, Penerima, Arsip)",
			pin:     "",
		},
		{
			title:   "Ketentuan Lembur & Perhitungan PPh21 HR",
			dept:    "HR",
			tags:    "lembur, pph21, absensi, gaji",
			content: "# Ketentuan Lembur & PPh21 HR\n\nPengajuan lembur wajib dicatat pada sistem absensi sebelum jam 17:00 WIB.\n\n- Batas maksimal lembur: 4 jam / hari kerja\n- Perhitungan pph21 mengacu pada aturan TER terbaru\n- Lampirkan bukti foto presensi fingerprint atau aplikasi mobile",
			pin:     "5678",
		},
	}

	for _, s := range seeds {
		_, _ = database.Exec(`INSERT INTO notes (title, departemen, tags, content_markdown, edit_pin) VALUES (?, ?, ?, ?, ?)`, s.title, s.dept, s.tags, s.content, s.pin)
	}
}

func FetchNotes(database *sql.DB, q, departemen, tag string) ([]Note, error) {
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

	rows, err := database.Query(query, args...)
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
		list = append(list, n)
	}
	return list, nil
}

func GetNoteByID(database *sql.DB, id int) (*Note, error) {
	row := database.QueryRow("SELECT id, title, departemen, tags, content_markdown, edit_pin, created_at, updated_at FROM notes WHERE id = ?", id)
	var n Note
	err := row.Scan(&n.ID, &n.Title, &n.Departemen, &n.Tags, &n.ContentMarkdown, &n.EditPIN, &n.CreatedAt, &n.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &n, nil
}

func CreateNote(database *sql.DB, title, departemen, tags, content, pin string) (int64, error) {
	res, err := database.Exec("INSERT INTO notes (title, departemen, tags, content_markdown, edit_pin) VALUES (?, ?, ?, ?, ?)",
		title, departemen, tags, content, pin)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func UpdateNote(database *sql.DB, id int, title, departemen, tags, content, pin string) error {
	_, err := database.Exec(`UPDATE notes SET title = ?, departemen = ?, tags = ?, content_markdown = ?, edit_pin = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		title, departemen, tags, content, pin, id)
	return err
}

func DeleteNote(database *sql.DB, id int) error {
	_, err := database.Exec("DELETE FROM notes WHERE id = ?", id)
	return err
}
