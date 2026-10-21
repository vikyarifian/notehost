package handler

import (
	"bytes"
	"database/sql"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"

	"github.com/yuin/goldmark"

	"notehost/internal/db"
)

type Handler struct {
	DB   *sql.DB
	Tmpl *template.Template
}

type NoteView struct {
	db.Note
	ContentHTML template.HTML
}

func NewHandler(database *sql.DB, tmpl *template.Template) *Handler {
	return &Handler{
		DB:   database,
		Tmpl: tmpl,
	}
}

func RenderMarkdown(md string) template.HTML {
	var buf bytes.Buffer
	if err := goldmark.Convert([]byte(md), &buf); err != nil {
		return template.HTML(template.HTMLEscapeString(md))
	}
	return template.HTML(buf.String())
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /", h.HandleHome)
	mux.HandleFunc("GET /notes", h.HandleNotesSearch)
	mux.HandleFunc("GET /notes/new", h.HandleNoteNew)
	mux.HandleFunc("POST /notes", h.HandleNoteCreate)
	mux.HandleFunc("POST /notes/preview", h.HandlePreview)
	mux.HandleFunc("GET /notes/{id}", h.HandleNoteView)
	mux.HandleFunc("GET /notes/{id}/edit", h.HandleNoteEdit)
	mux.HandleFunc("POST /notes/{id}/verify-pin", h.HandleVerifyPIN)
	mux.HandleFunc("POST /notes/{id}", h.HandleNoteUpdate)
	mux.HandleFunc("DELETE /notes/{id}", h.HandleNoteDelete)
}

func (h *Handler) checkPINAuth(r *http.Request, note *db.Note) bool {
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

func (h *Handler) HandleHome(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	q := r.URL.Query().Get("q")
	dept := r.URL.Query().Get("departemen")
	tag := r.URL.Query().Get("tag")

	notes, err := db.FetchNotes(h.DB, q, dept, tag)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var noteViews []NoteView
	for _, n := range notes {
		noteViews = append(noteViews, NoteView{
			Note:        n,
			ContentHTML: RenderMarkdown(n.ContentMarkdown),
		})
	}

	data := map[string]interface{}{
		"Notes":      noteViews,
		"Query":      q,
		"Departemen": dept,
		"Tag":        tag,
	}
	h.Tmpl.ExecuteTemplate(w, "index", data)
}

func (h *Handler) HandleNotesSearch(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	dept := r.URL.Query().Get("departemen")
	tag := r.URL.Query().Get("tag")

	notes, err := db.FetchNotes(h.DB, q, dept, tag)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var noteViews []NoteView
	for _, n := range notes {
		noteViews = append(noteViews, NoteView{
			Note:        n,
			ContentHTML: RenderMarkdown(n.ContentMarkdown),
		})
	}

	data := map[string]interface{}{
		"Notes":      noteViews,
		"Query":      q,
		"Departemen": dept,
		"Tag":        tag,
	}

	if r.Header.Get("HX-Request") == "true" {
		h.Tmpl.ExecuteTemplate(w, "_note_list.html", data)
		return
	}
	h.Tmpl.ExecuteTemplate(w, "index", data)
}

