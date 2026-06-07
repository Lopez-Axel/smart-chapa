package handlers

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"smart-chapa/internal/middleware"
	"smart-chapa/internal/models"
)

type DeviceHandler struct {
	db *sql.DB
}

func NewDeviceHandler(db *sql.DB) *DeviceHandler {
	return &DeviceHandler{db: db}
}

func (h *DeviceHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())

	var body struct {
		Name    string `json:"name"`
		HouseID int64  `json:"house_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Name == "" {
		http.Error(w, "nombre requerido", http.StatusBadRequest)
		return
	}

	if body.HouseID != 0 && !userHasAccess(h.db, userID, body.HouseID) {
		http.Error(w, "casa no encontrada", http.StatusNotFound)
		return
	}

	token, err := generateToken()
	if err != nil {
		http.Error(w, "error generando token", http.StatusInternalServerError)
		return
	}

	res, err := h.db.Exec(
		"INSERT INTO devices (name, token, user_id, house_id) VALUES (?, ?, ?, ?)",
		body.Name, token, userID, body.HouseID,
	)
	if err != nil {
		http.Error(w, "error creando dispositivo", http.StatusInternalServerError)
		return
	}

	id, _ := res.LastInsertId()
	device := models.Device{ID: id, Name: body.Name, Token: token, UserID: userID, HouseID: body.HouseID}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(device)
}

func (h *DeviceHandler) List(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())

	rows, err := h.db.Query(
		"SELECT id, name, token, user_id, house_id, created_at FROM devices WHERE user_id = ?",
		userID,
	)
	if err != nil {
		http.Error(w, "error consultando dispositivos", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	devices := []models.Device{}
	for rows.Next() {
		var d models.Device
		rows.Scan(&d.ID, &d.Name, &d.Token, &d.UserID, &d.HouseID, &d.CreatedAt)
		devices = append(devices, d)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(devices)
}

func (h *DeviceHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())

	idStr := chi.URLParam(r, "id")
	deviceID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "id inválido", http.StatusBadRequest)
		return
	}

	res, err := h.db.Exec(
		"DELETE FROM devices WHERE id = ? AND user_id = ?",
		deviceID, userID,
	)
	if err != nil {
		http.Error(w, "error eliminando dispositivo", http.StatusInternalServerError)
		return
	}

	affected, _ := res.RowsAffected()
	if affected == 0 {
		http.Error(w, "dispositivo no encontrado", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func generateToken() (string, error) {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	return hex.EncodeToString(b), err
}
