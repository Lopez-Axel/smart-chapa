package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"smart-chapa/internal/middleware"
	"smart-chapa/internal/models"
)

type HouseHandler struct {
	db *sql.DB
}

func NewHouseHandler(db *sql.DB) *HouseHandler {
	return &HouseHandler{db: db}
}

func (h *HouseHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())

	var body struct {
		Name      string  `json:"name"`
		Address   string  `json:"address"`
		City      string  `json:"city"`
		Country   string  `json:"country"`
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Name == "" {
		http.Error(w, "nombre requerido", http.StatusBadRequest)
		return
	}

	tx, err := h.db.Begin()
	if err != nil {
		http.Error(w, "error interno", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	res, err := tx.Exec(
		"INSERT INTO houses (name, address, city, country, latitude, longitude) VALUES (?, ?, ?, ?, ?, ?)",
		body.Name, body.Address, body.City, body.Country, body.Latitude, body.Longitude,
	)
	if err != nil {
		http.Error(w, "error creando casa", http.StatusInternalServerError)
		return
	}

	houseID, err := res.LastInsertId()
	if err != nil {
		http.Error(w, "error creando casa", http.StatusInternalServerError)
		return
	}

	_, err = tx.Exec(
		"INSERT INTO user_houses (user_id, house_id, role) VALUES (?, ?, 'owner')",
		userID, houseID,
	)
	if err != nil {
		http.Error(w, "error asignando propietario", http.StatusInternalServerError)
		return
	}

	if err := tx.Commit(); err != nil {
		http.Error(w, "error interno", http.StatusInternalServerError)
		return
	}

	house := models.House{ID: houseID, Name: body.Name, Address: body.Address, City: body.City, Country: body.Country, Latitude: body.Latitude, Longitude: body.Longitude}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(house)
}

func (h *HouseHandler) List(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())

	rows, err := h.db.Query(`
		SELECT h.id, h.name, h.address, h.city, h.country, h.latitude, h.longitude, h.created_at
		FROM houses h
		JOIN user_houses uh ON uh.house_id = h.id
		WHERE uh.user_id = ?
	`, userID)
	if err != nil {
		http.Error(w, "error consultando casas", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	houses := []models.House{}
	for rows.Next() {
		var house models.House
		if err := rows.Scan(&house.ID, &house.Name, &house.Address, &house.City, &house.Country, &house.Latitude, &house.Longitude, &house.CreatedAt); err != nil {
			continue
		}
		houses = append(houses, house)
	}
	if err := rows.Err(); err != nil {
		http.Error(w, "error leyendo casas", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(houses)
}

func (h *HouseHandler) AddMember(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())

	houseIDStr := chi.URLParam(r, "id")
	houseID, err := strconv.ParseInt(houseIDStr, 10, 64)
	if err != nil {
		http.Error(w, "id inválido", http.StatusBadRequest)
		return
	}

	var role string
	err = h.db.QueryRow(
		"SELECT role FROM user_houses WHERE user_id = ? AND house_id = ?",
		userID, houseID,
	).Scan(&role)
	if err != nil || role != "owner" {
		http.Error(w, "solo el propietario puede agregar miembros", http.StatusForbidden)
		return
	}

	var body struct {
		UserID int64  `json:"user_id"`
		Role   string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.UserID == 0 {
		http.Error(w, "user_id requerido", http.StatusBadRequest)
		return
	}
	if body.Role == "" {
		body.Role = "member"
	}

	_, err = h.db.Exec(
		"INSERT INTO user_houses (user_id, house_id, role) VALUES (?, ?, ?)",
		body.UserID, houseID, body.Role,
	)
	if err != nil {
		http.Error(w, "error agregando miembro", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (h *HouseHandler) GetDevices(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())

	houseIDStr := chi.URLParam(r, "id")
	houseID, err := strconv.ParseInt(houseIDStr, 10, 64)
	if err != nil {
		http.Error(w, "id inválido", http.StatusBadRequest)
		return
	}

	if !userHasAccess(h.db, userID, houseID) {
		http.Error(w, "casa no encontrada", http.StatusNotFound)
		return
	}

	rows, err := h.db.Query(
		"SELECT id, name, token, user_id, house_id, created_at FROM devices WHERE house_id = ?",
		houseID,
	)
	if err != nil {
		http.Error(w, "error consultando dispositivos", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	devices := []models.Device{}
	for rows.Next() {
		var d models.Device
		if err := rows.Scan(&d.ID, &d.Name, &d.Token, &d.UserID, &d.HouseID, &d.CreatedAt); err != nil {
			continue
		}
		devices = append(devices, d)
	}
	if err := rows.Err(); err != nil {
		http.Error(w, "error leyendo dispositivos", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(devices)
}
