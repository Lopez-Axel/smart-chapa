package handlers

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type LightHandler struct {
	db   *sql.DB
	mqtt mqtt.Client
}

func NewLightHandler(db *sql.DB, mqtt mqtt.Client) *LightHandler {
	h := &LightHandler{db: db, mqtt: mqtt}
	h.subscribe()
	return h
}

func (h *LightHandler) subscribe() {
	token := h.mqtt.Subscribe("lights/status", 1, func(client mqtt.Client, msg mqtt.Message) {
		payload := string(msg.Payload())
		log.Printf("Estado de luz recibido: %s", payload)

		var data struct {
			DeviceID int64  `json:"device_id"`
			State    string `json:"state"`
		}
		if err := json.Unmarshal(msg.Payload(), &data); err != nil || data.State == "" {
			h.db.Exec(
				"INSERT INTO light_events (device_id, state, source) VALUES (?, ?, 'mqtt')",
				0, payload,
			)
			return
		}

		h.db.Exec(
			"INSERT INTO light_events (device_id, state, source) VALUES (?, ?, 'mqtt')",
			data.DeviceID, data.State,
		)
	})
	token.Wait()
	log.Println("Suscrito a lights/status")
}

func (h *LightHandler) TurnOn(w http.ResponseWriter, r *http.Request) {
	var body struct {
		DeviceID int64 `json:"device_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.DeviceID == 0 {
		http.Error(w, "device_id requerido", http.StatusBadRequest)
		return
	}

	token := h.mqtt.Publish("lights/cmd", 1, false, "turn_on")
	token.Wait()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (h *LightHandler) TurnOff(w http.ResponseWriter, r *http.Request) {
	var body struct {
		DeviceID int64 `json:"device_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.DeviceID == 0 {
		http.Error(w, "device_id requerido", http.StatusBadRequest)
		return
	}

	token := h.mqtt.Publish("lights/cmd", 1, false, "turn_off")
	token.Wait()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (h *LightHandler) Events(w http.ResponseWriter, r *http.Request) {
	rows, err := h.db.Query(`
		SELECT id, device_id, state, source, created_at
		FROM light_events
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
		State     string `json:"state"`
		Source    string `json:"source"`
		CreatedAt string `json:"created_at"`
	}

	events := []Event{}
	for rows.Next() {
		var e Event
		rows.Scan(&e.ID, &e.DeviceID, &e.State, &e.Source, &e.CreatedAt)
		events = append(events, e)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(events)
}
