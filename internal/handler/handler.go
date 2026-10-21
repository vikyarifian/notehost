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

func (h *Handler) HandleNoteNew(w http.ResponseWriter, r *http.Request) {
	h.Tmpl.ExecuteTemplate(w, "new_note", nil)
}

func (h *Handler) HandleNoteCreate(w http.ResponseWriter, r *http.Request) {
	title := r.FormValue("title")
	dept := r.FormValue("departemen")
	tags := r.FormValue("tags")
	content := r.FormValue("content_markdown")
	pin := strings.TrimSpace(r.FormValue("edit_pin"))

	id, err := db.CreateNote(h.DB, title, dept, tags, content, pin)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	redirectURL := fmt.Sprintf("/notes/%d", id)
	if r.Header.Get("HX-Request") == "true" {
		// htmx limitation: redirect response in HTMX request requires custom HX-Redirect header
		w.Header().Set("HX-Redirect", redirectURL)
		w.WriteHeader(http.StatusOK)
		return
	}
	http.Redirect(w, r, redirectURL, http.StatusSeeOther)
}

func (h *Handler) HandlePreview(w http.ResponseWriter, r *http.Request) {
	content := r.FormValue("content_markdown")
	html := RenderMarkdown(content)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(html))
}

func (h *Handler) HandleNoteView(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	note, err := db.GetNoteByID(h.DB, id)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	noteView := NoteView{
		Note:        *note,
		ContentHTML: RenderMarkdown(note.ContentMarkdown),
	}

	data := map[string]interface{}{
		"Note": noteView,
	}
	h.Tmpl.ExecuteTemplate(w, "view_note", data)
}

func (h *Handler) HandleNoteEdit(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	note, err := db.GetNoteByID(h.DB, id)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	if !h.checkPINAuth(r, note) {
		h.Tmpl.ExecuteTemplate(w, "pin_prompt", map[string]interface{}{
			"NoteID": note.ID,
			"Error":  "",
		})
		return
	}

	noteView := NoteView{
		Note:        *note,
		ContentHTML: RenderMarkdown(note.ContentMarkdown),
	}

	h.Tmpl.ExecuteTemplate(w, "edit_note", map[string]interface{}{
		"Note":  noteView,
		"Error": "",
	})
}

func (h *Handler) HandleVerifyPIN(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	note, err := db.GetNoteByID(h.DB, id)
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

		noteView := NoteView{
			Note:        *note,
			ContentHTML: RenderMarkdown(note.ContentMarkdown),
		}

		if r.Header.Get("HX-Request") == "true" {
			h.Tmpl.ExecuteTemplate(w, "edit_note", map[string]interface{}{
				"Note":  noteView,
				"Error": "",
			})
			return
		}
		http.Redirect(w, r, fmt.Sprintf("/notes/%d/edit", note.ID), http.StatusSeeOther)
		return
	}

	h.Tmpl.ExecuteTemplate(w, "pin_prompt", map[string]interface{}{
		"NoteID": note.ID,
		"Error":  "PIN salah, silakan coba lagi.",
	})
}

func (h *Handler) HandleNoteUpdate(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	note, err := db.GetNoteByID(h.DB, id)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	if !h.checkPINAuth(r, note) {
		http.Error(w, "Unauthorized PIN", http.StatusForbidden)
		return
	}

	title := r.FormValue("title")
	dept := r.FormValue("departemen")
	tags := r.FormValue("tags")
	content := r.FormValue("content_markdown")
	newPIN := strings.TrimSpace(r.FormValue("edit_pin"))

	err = db.UpdateNote(h.DB, id, title, dept, tags, content, newPIN)
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

func (h *Handler) HandleNoteDelete(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	note, err := db.GetNoteByID(h.DB, id)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	if !h.checkPINAuth(r, note) {
		http.Error(w, "PIN required to delete this note", http.StatusForbidden)
		return
	}

	err = db.DeleteNote(h.DB, id)
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
