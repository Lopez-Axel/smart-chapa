package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type DoorHandler struct {
	db   *sql.DB
	mqtt mqtt.Client
}

func NewDoorHandler(db *sql.DB, mqtt mqtt.Client) *DoorHandler {
	return &DoorHandler{db: db, mqtt: mqtt}
}

func (h *DoorHandler) Open(w http.ResponseWriter, r *http.Request) {
	var body struct {
		DeviceID int64  `json:"device_id"`
		Source   string `json:"source"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.DeviceID == 0 {
		http.Error(w, "device_id requerido", http.StatusBadRequest)
		return
	}

	token := h.mqtt.Publish("door/cmd", 1, false, "open")
	token.Wait()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (h *DoorHandler) Confirm(w http.ResponseWriter, r *http.Request) {
	var body struct {
		CommandID int64  `json:"command_id"`
		DeviceID  int64  `json:"device_id"`
		Action    string `json:"action"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "body inválido", http.StatusBadRequest)
		return
	}

	h.db.Exec(
		"INSERT INTO door_events (device_id, action, source) VALUES (?, ?, 'esp32')",
		body.DeviceID, body.Action,
	)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (h *DoorHandler) Events(w http.ResponseWriter, r *http.Request) {
	rows, err := h.db.Query(`
        SELECT id, device_id, action, source, created_at
        FROM door_events
        ORDER BY created_at DESC
        LIMIT 50
    `)
	if err != nil {
		http.Error(w, "error consultando eventos", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type Event struct {
		ID        int64  `json:"id"`
		DeviceID  int64  `json:"device_id"`
		Action    string `json:"action"`
		Source    string `json:"source"`
		CreatedAt string `json:"created_at"`
	}

	events := []Event{}
	for rows.Next() {
		var e Event
		rows.Scan(&e.ID, &e.DeviceID, &e.Action, &e.Source, &e.CreatedAt)
		events = append(events, e)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(events)
}
