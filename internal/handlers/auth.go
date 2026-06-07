package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"smart-chapa/internal/models"
)

type AuthHandler struct {
	db *sql.DB
}

func NewAuthHandler(db *sql.DB) *AuthHandler {
	return &AuthHandler{db: db}
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Name == "" || body.Email == "" || body.Password == "" {
		http.Error(w, "nombre, email y password requeridos", http.StatusBadRequest)
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(body.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "error interno", http.StatusInternalServerError)
		return
	}

	res, err := h.db.Exec(
		"INSERT INTO users (name, email, password_hash) VALUES (?, ?, ?)",
		body.Name, body.Email, string(hash),
	)
	if err != nil {
		http.Error(w, "email ya registrado", http.StatusConflict)
		return
	}

	id, err := res.LastInsertId()
	if err != nil {
		http.Error(w, "error interno", http.StatusInternalServerError)
		return
	}
	user := models.User{ID: id, Name: body.Name, Email: body.Email}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(user)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Email == "" || body.Password == "" {
		http.Error(w, "email y password requeridos", http.StatusBadRequest)
		return
	}

	var user models.User
	var passwordHash string
	err := h.db.QueryRow(
		"SELECT id, name, email, password_hash FROM users WHERE email = ?",
		body.Email,
	).Scan(&user.ID, &user.Name, &user.Email, &passwordHash)
	if err != nil {
		http.Error(w, "email o password incorrectos", http.StatusUnauthorized)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(body.Password)); err != nil {
		http.Error(w, "email o password incorrectos", http.StatusUnauthorized)
		return
	}

	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		http.Error(w, "error interno", http.StatusInternalServerError)
		return
	}

	claims := jwt.MapClaims{
		"user_id": user.ID,
		"exp":     time.Now().Add(7 * 24 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		http.Error(w, "error interno", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"token": signed})
}
