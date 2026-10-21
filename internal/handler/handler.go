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

