package handlers

import (
	"allshop-api/cache"
	"allshop-api/storage"
	"database/sql"
	"encoding/json"
	"net/http"
)

type Handler struct {
	db        *sql.DB
	jwtSecret string
	cache     *cache.Cache
	storage   *storage.Storage
}

func New(db *sql.DB, jwtSecret string, cache *cache.Cache, storage *storage.Storage) *Handler {
	return &Handler{db: db, jwtSecret: jwtSecret, cache: cache, storage: storage}
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
