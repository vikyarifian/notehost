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

