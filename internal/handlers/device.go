package handlers

import (
    "crypto/rand"
    "database/sql"
    "encoding/hex"
    "encoding/json"
    "net/http"

    "smart-chapa/internal/models"
)

type DeviceHandler struct {
    db *sql.DB
}

func NewDeviceHandler(db *sql.DB) *DeviceHandler {
    return &DeviceHandler{db: db}
}

func (h *DeviceHandler) Create(w http.ResponseWriter, r *http.Request) {
    var body struct {
        Name string `json:"name"`
    }
    if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Name == "" {
        http.Error(w, "nombre requerido", http.StatusBadRequest)
        return
    }

    token, err := generateToken()
    if err != nil {
        http.Error(w, "error generando token", http.StatusInternalServerError)
        return
    }

    res, err := h.db.Exec(
        "INSERT INTO devices (name, token) VALUES (?, ?)",
        body.Name, token,
    )
    if err != nil {
        http.Error(w, "error creando dispositivo", http.StatusInternalServerError)
        return
    }

    id, _ := res.LastInsertId()
    device := models.Device{ID: id, Name: body.Name, Token: token}

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusCreated)
    json.NewEncoder(w).Encode(device)
}

func (h *DeviceHandler) List(w http.ResponseWriter, r *http.Request) {
    rows, err := h.db.Query("SELECT id, name, token, created_at FROM devices")
    if err != nil {
        http.Error(w, "error consultando dispositivos", http.StatusInternalServerError)
        return
    }
    defer rows.Close()

    devices := []models.Device{}
    for rows.Next() {
        var d models.Device
        rows.Scan(&d.ID, &d.Name, &d.Token, &d.CreatedAt)
        devices = append(devices, d)
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(devices)
}

func generateToken() (string, error) {
    b := make([]byte, 16)
    _, err := rand.Read(b)
    return hex.EncodeToString(b), err
}
