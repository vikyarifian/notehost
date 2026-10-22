package templates

import (
	"bytes"
	"fmt"
	"html/template"

	"github.com/yuin/goldmark"
)

func RenderMarkdown(md string) template.HTML {
	var buf bytes.Buffer
	if err := goldmark.Convert([]byte(md), &buf); err != nil {
		return template.HTML(template.HTMLEscapeString(md))
	}
	return template.HTML(buf.String())
}

// html/template limitation: custom functions must be added to FuncMap before parsing template strings
func InitTemplates() (*template.Template, error) {
	funcMap := template.FuncMap{
		"renderMD": RenderMarkdown,
	}

	tmpl, err := template.New("root").Funcs(funcMap).Parse(HTMLTemplates)
	if err != nil {
		return nil, fmt.Errorf("failed to parse templates: %w", err)
	}
	return tmpl, nil
}

func Parse() (*template.Template, error) {
	return InitTemplates()
}

func New() (*template.Template, error) {
	return InitTemplates()
}

const HTMLTemplates = `
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
