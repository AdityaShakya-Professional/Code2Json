package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"code2json/internal/parser"
)

type Response struct {
	Success bool        `json:"success"`
	JSON    interface{} `json:"json,omitempty"`
	Error   string      `json:"error,omitempty"`
}

func GenerateJSON(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "only POST allowed")
		return
	}

	if err := r.ParseMultipartForm(10 << 20); err != nil {
		// Fall back to plain form for text-only (pasted code) submissions.
		if err2 := r.ParseForm(); err2 != nil {
			writeError(w, http.StatusBadRequest, "failed to parse form: "+err.Error())
			return
		}
	}

	lang := r.FormValue("language")
	if lang == "" {
		writeError(w, http.StatusBadRequest, "language field is required")
		return
	}

	// Prefer uploaded file; fall back to pasted code text.
	var content []byte
	if r.MultipartForm != nil {
		file, _, fileErr := r.FormFile("file")
		if fileErr == nil {
			defer file.Close()
			raw, readErr := io.ReadAll(file)
			if readErr != nil {
				writeError(w, http.StatusInternalServerError, "failed to read file: "+readErr.Error())
				return
			}
			content = raw
		}
	}
	if len(content) == 0 {
		code := r.FormValue("code")
		if strings.TrimSpace(code) == "" {
			writeError(w, http.StatusBadRequest, "either a file or pasted code is required")
			return
		}
		content = []byte(code)
	}

	var (
		result   interface{}
		parseErr error
	)
	switch lang {
	case "java":
		result, parseErr = parser.ParseJavaDTO(string(content))
	case "golang":
		result, parseErr = parser.ParseGoStruct(string(content))
	default:
		writeError(w, http.StatusBadRequest, fmt.Sprintf("unsupported language: %s", lang))
		return
	}

	if parseErr != nil {
		writeError(w, http.StatusUnprocessableEntity, parseErr.Error())
		return
	}

	json.NewEncoder(w).Encode(Response{Success: true, JSON: result})
}

func writeError(w http.ResponseWriter, status int, msg string) {
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(Response{Success: false, Error: msg})
}
