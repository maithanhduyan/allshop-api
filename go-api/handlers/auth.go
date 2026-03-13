package handlers

import (
	"allshop-api/middleware"
	"allshop-api/models"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req models.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" || req.Email == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "name, email and password are required")
		return
	}

	if len(req.Password) < 6 {
		writeError(w, http.StatusBadRequest, "password must be at least 6 characters")
		return
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to hash password")
		return
	}

	var user models.User
	err = h.db.QueryRow(
		`INSERT INTO users (name, email, password) VALUES ($1, $2, $3)
		 RETURNING id, name, email, phone, avatar, created_at`,
		req.Name, req.Email, string(hashed),
	).Scan(&user.ID, &user.Name, &user.Email, &user.Phone, &user.Avatar, &user.CreatedAt)
	if err != nil {
		writeError(w, http.StatusConflict, "email already registered")
		return
	}

	token, err := h.generateToken(user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate token")
		return
	}

	writeJSON(w, http.StatusCreated, models.AuthResponse{Token: token, User: user})
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Email == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "email and password are required")
		return
	}

	var user models.User
	err := h.db.QueryRow(
		`SELECT id, name, email, phone, avatar, password, created_at FROM users WHERE email = $1`,
		req.Email,
	).Scan(&user.ID, &user.Name, &user.Email, &user.Phone, &user.Avatar, &user.Password, &user.CreatedAt)
	if err == sql.ErrNoRows {
		writeError(w, http.StatusUnauthorized, "invalid email or password")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to query user")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		writeError(w, http.StatusUnauthorized, "invalid email or password")
		return
	}

	token, err := h.generateToken(user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate token")
		return
	}

	user.Password = ""
	writeJSON(w, http.StatusOK, models.AuthResponse{Token: token, User: user})
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"message": "logged out successfully"})
}

func (h *Handler) generateToken(userID string) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(72 * time.Hour).Unix(),
		"iat":     time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(h.jwtSecret))
}

func getUserID(r *http.Request) string {
	if v := r.Context().Value(middleware.UserIDKey); v != nil {
		return v.(string)
	}
	return ""
}

func (h *Handler) GetProfile(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	var user models.User
	err := h.db.QueryRow(
		`SELECT id, name, email, phone, avatar, created_at FROM users WHERE id = $1`, userID,
	).Scan(&user.ID, &user.Name, &user.Email, &user.Phone, &user.Avatar, &user.CreatedAt)
	if err != nil {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}
	writeJSON(w, http.StatusOK, user)
}

func (h *Handler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	var req models.UpdateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	var user models.User
	err := h.db.QueryRow(
		`UPDATE users SET name = COALESCE(NULLIF($1, ''), name), phone = COALESCE($2, phone)
		 WHERE id = $3 RETURNING id, name, email, phone, avatar, created_at`,
		req.Name, req.Phone, userID,
	).Scan(&user.ID, &user.Name, &user.Email, &user.Phone, &user.Avatar, &user.CreatedAt)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to update profile: %v", err))
		return
	}

	writeJSON(w, http.StatusOK, user)
}
