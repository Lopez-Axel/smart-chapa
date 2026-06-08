package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	mqtt "github.com/eclipse/paho.mqtt.golang"

	"smart-chapa/internal/middleware"
	"smart-chapa/internal/models"
)

type ActuatorHandler struct {
	db   *sql.DB
	mqtt mqtt.Client
}

func NewActuatorHandler(db *sql.DB, mqttClient mqtt.Client) *ActuatorHandler {
	return &ActuatorHandler{db: db, mqtt: mqttClient}
}

func (h *ActuatorHandler) SetMQTT(client mqtt.Client) {
	h.mqtt = client
}

func topicForActuator(deviceID int64, actuatorType string, suffix string) string {
	return fmt.Sprintf("%d/%s/%s", deviceID, actuatorType, suffix)
}

func normalizeState(s string) string {
	switch s {
	case "relay_on", "turn_on":
		return "on"
	case "relay_off", "turn_off":
		return "off"
	}
	return s
}

func (h *ActuatorHandler) SubscribeEvents() {
	token := h.mqtt.Subscribe("+/+/status", 1, func(client mqtt.Client, msg mqtt.Message) {
		parts := strings.Split(msg.Topic(), "/")
		if len(parts) != 3 {
			log.Printf("topic inválido: %s", msg.Topic())
			return
		}
		deviceID, err := strconv.ParseInt(parts[0], 10, 64)
		if err != nil {
			log.Printf("device_id inválido en topic: %s", msg.Topic())
			return
		}

		payload := string(msg.Payload())
		log.Printf("Status recibido [%s]: %s", msg.Topic(), payload)

		var data struct {
			Relay int    `json:"relay"`
			State string `json:"state"`
		}
		if err := json.Unmarshal(msg.Payload(), &data); err != nil || data.State == "" {
			if _, e := h.db.Exec("INSERT INTO actuator_events (actuator_id, state, source) VALUES (?, ?, 'mqtt')", 0, payload); e != nil {
				log.Printf("error guardando status raw: %v", e)
			}
			return
		}

		var actuatorID int64
		var currentState string
		err = h.db.QueryRow(
			"SELECT id, state FROM actuators WHERE device_id = ? AND relay_num = ?",
			deviceID, data.Relay,
		).Scan(&actuatorID, &currentState)
		if err != nil {
			log.Printf("actuador no encontrado para device=%d relay=%d", deviceID, data.Relay)
			if _, e := h.db.Exec("INSERT INTO actuator_events (actuator_id, state, source) VALUES (?, ?, 'mqtt')", 0, payload); e != nil {
				log.Printf("error guardando status raw: %v", e)
			}
			return
		}

		normalized := normalizeState(data.State)

		if _, err := h.db.Exec("INSERT INTO actuator_events (actuator_id, state, source, details) VALUES (?, ?, 'mqtt', ?)", actuatorID, normalized, payload); err != nil {
			log.Printf("error guardando evento: %v", err)
		}

		switch currentState {
		case "pending_on":
			h.db.Exec("UPDATE actuators SET state = 'on' WHERE id = ?", actuatorID)
		case "pending_off":
			h.db.Exec("UPDATE actuators SET state = 'off' WHERE id = ?", actuatorID)
		default:
			if normalized != currentState {
				h.db.Exec("UPDATE actuators SET state = ? WHERE id = ?", normalized, actuatorID)
			}
		}
	})
	if token.Wait(); token.Error() != nil {
		log.Printf("error suscribiendo a +/+/status: %v", token.Error())
	} else {
		log.Println("Suscrito a +/+/status")
	}
}

func (h *ActuatorHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())

	var body struct {
		DeviceID int64  `json:"device_id"`
		Name     string `json:"name"`
		Type     string `json:"type"`
		RelayNum int    `json:"relay_num"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.DeviceID == 0 || body.Name == "" || body.Type == "" {
		http.Error(w, "device_id, name y type requeridos", http.StatusBadRequest)
		return
	}

	var ownerID int64
	err := h.db.QueryRow("SELECT user_id FROM devices WHERE id = ?", body.DeviceID).Scan(&ownerID)
	if err != nil {
		http.Error(w, "dispositivo no encontrado", http.StatusNotFound)
		return
	}
	if ownerID != userID {
		http.Error(w, "no tienes acceso a este dispositivo", http.StatusForbidden)
		return
	}

	var existing int
	h.db.QueryRow("SELECT COUNT(*) FROM actuators WHERE device_id = ? AND relay_num = ?", body.DeviceID, body.RelayNum).Scan(&existing)
	if existing > 0 {
		http.Error(w, "relay_num ya existe para este dispositivo", http.StatusConflict)
		return
	}

	res, err := h.db.Exec(
		"INSERT INTO actuators (device_id, name, type, relay_num) VALUES (?, ?, ?, ?)",
		body.DeviceID, body.Name, body.Type, body.RelayNum,
	)
	if err != nil {
		http.Error(w, "error creando actuador", http.StatusInternalServerError)
		return
	}

	id, _ := res.LastInsertId()
	actuator := models.Actuator{
		ID:       id,
		DeviceID: body.DeviceID,
		Name:     body.Name,
		Type:     body.Type,
		RelayNum: body.RelayNum,
		State:    "off",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(actuator)
}

func (h *ActuatorHandler) List(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())

	deviceIDStr := r.URL.Query().Get("device_id")
	if deviceIDStr == "" {
		http.Error(w, "device_id requerido", http.StatusBadRequest)
		return
	}
	deviceID, err := strconv.ParseInt(deviceIDStr, 10, 64)
	if err != nil {
		http.Error(w, "device_id inválido", http.StatusBadRequest)
		return
	}

	var ownerID int64
	err = h.db.QueryRow("SELECT user_id FROM devices WHERE id = ?", deviceID).Scan(&ownerID)
	if err != nil {
		http.Error(w, "dispositivo no encontrado", http.StatusNotFound)
		return
	}
	if ownerID != userID {
		http.Error(w, "no tienes acceso a este dispositivo", http.StatusForbidden)
		return
	}

	rows, err := h.db.Query(
		"SELECT id, device_id, name, type, relay_num, state, created_at FROM actuators WHERE device_id = ?",
		deviceID,
	)
	if err != nil {
		http.Error(w, "error consultando actuadores", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	actuators := []models.Actuator{}
	for rows.Next() {
		var a models.Actuator
		if err := rows.Scan(&a.ID, &a.DeviceID, &a.Name, &a.Type, &a.RelayNum, &a.State, &a.CreatedAt); err != nil {
			continue
		}
		actuators = append(actuators, a)
	}
	if err := rows.Err(); err != nil {
		http.Error(w, "error leyendo actuadores", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(actuators)
}

func (h *ActuatorHandler) TurnOn(w http.ResponseWriter, r *http.Request) {
	h.control(w, r, "turn_on")
}

func (h *ActuatorHandler) TurnOff(w http.ResponseWriter, r *http.Request) {
	h.control(w, r, "turn_off")
}

func (h *ActuatorHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())

	actuatorIDStr := chi.URLParam(r, "id")
	actuatorID, err := strconv.ParseInt(actuatorIDStr, 10, 64)
	if err != nil {
		http.Error(w, "id inválido", http.StatusBadRequest)
		return
	}

	var a models.Actuator
	err = h.db.QueryRow(
		"SELECT a.id, a.device_id, a.name, a.type, a.relay_num, a.state, a.created_at FROM actuators a JOIN devices d ON a.device_id = d.id WHERE a.id = ? AND d.user_id = ?",
		actuatorID, userID,
	).Scan(&a.ID, &a.DeviceID, &a.Name, &a.Type, &a.RelayNum, &a.State, &a.CreatedAt)
	if err != nil {
		http.Error(w, "actuador no encontrado", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(a)
}

func (h *ActuatorHandler) control(w http.ResponseWriter, r *http.Request, command string) {
	userID := middleware.UserIDFromContext(r.Context())

	actuatorIDStr := chi.URLParam(r, "id")
	actuatorID, err := strconv.ParseInt(actuatorIDStr, 10, 64)
	if err != nil {
		http.Error(w, "id inválido", http.StatusBadRequest)
		return
	}

	var relayNum int
	var deviceID int64
	var actuatorType string
	var ownerID int64
	var currentState string
	err = h.db.QueryRow(
		"SELECT a.relay_num, a.device_id, a.type, d.user_id, a.state FROM actuators a JOIN devices d ON a.device_id = d.id WHERE a.id = ?",
		actuatorID,
	).Scan(&relayNum, &deviceID, &actuatorType, &ownerID, &currentState)
	if err != nil {
		http.Error(w, "actuador no encontrado", http.StatusNotFound)
		return
	}
	if ownerID != userID {
		http.Error(w, "no tienes acceso a este actuador", http.StatusForbidden)
		return
	}

	switch currentState {
	case "pending_on":
		if command == "turn_off" {
			http.Error(w, "no se puede apagar mientras hay un encendido pendiente", http.StatusConflict)
			return
		}
	case "pending_off":
		if command == "turn_on" {
			http.Error(w, "no se puede encender mientras hay un apagado pendiente", http.StatusConflict)
			return
		}
	case "on":
		if command == "turn_on" {
			http.Error(w, "el actuador ya se encuentra encendido", http.StatusConflict)
			return
		}
	case "off":
		if command == "turn_off" {
			http.Error(w, "el actuador ya se encuentra apagado", http.StatusConflict)
			return
		}
	}

	topic := topicForActuator(deviceID, actuatorType, "cmd")
	payload := fmt.Sprintf(`{"relay":%d,"state":"%s"}`, relayNum, command)

	retry := (currentState == "pending_on" && command == "turn_on") || (currentState == "pending_off" && command == "turn_off")
	if !retry {
		pendingState := "pending_on"
		if command == "turn_off" {
			pendingState = "pending_off"
		}

		tx, err := h.db.Begin()
		if err != nil {
			http.Error(w, "error interno", http.StatusInternalServerError)
			return
		}
		defer tx.Rollback()

		_, err = tx.Exec("UPDATE actuators SET state = ? WHERE id = ?", pendingState, actuatorID)
		if err != nil {
			http.Error(w, "error actualizando estado", http.StatusInternalServerError)
			return
		}

		_, err = tx.Exec(
			"INSERT INTO actuator_events (actuator_id, state, source, details) VALUES (?, ?, 'http', ?)",
			actuatorID, pendingState, payload,
		)
		if err != nil {
			http.Error(w, "error guardando evento", http.StatusInternalServerError)
			return
		}

		if err := tx.Commit(); err != nil {
			http.Error(w, "error interno", http.StatusInternalServerError)
			return
		}
	}

	token := h.mqtt.Publish(topic, 1, false, payload)
	if token.Wait(); token.Error() != nil {
		log.Printf("error publicando MQTT: %v", token.Error())
		return
	}

	log.Printf("Publicado %s en %s", payload, topic)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (h *ActuatorHandler) Events(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())

	actuatorIDStr := chi.URLParam(r, "id")
	actuatorID, err := strconv.ParseInt(actuatorIDStr, 10, 64)
	if err != nil {
		http.Error(w, "id inválido", http.StatusBadRequest)
		return
	}

	var ownerID int64
	err = h.db.QueryRow(
		"SELECT d.user_id FROM actuators a JOIN devices d ON a.device_id = d.id WHERE a.id = ?",
		actuatorID,
	).Scan(&ownerID)
	if err != nil {
		http.Error(w, "actuador no encontrado", http.StatusNotFound)
		return
	}
	if ownerID != userID {
		http.Error(w, "no tienes acceso a este actuador", http.StatusForbidden)
		return
	}

	rows, err := h.db.Query(`
		SELECT id, actuator_id, state, source, details, created_at
		FROM actuator_events
		WHERE actuator_id = ?
		ORDER BY created_at DESC
		LIMIT 50
	`, actuatorID)
	if err != nil {
		http.Error(w, "error consultando eventos", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type Event struct {
		ID         int64  `json:"id"`
		ActuatorID int64  `json:"actuator_id"`
		State      string `json:"state"`
		Source     string `json:"source"`
		Details    string `json:"details"`
		CreatedAt  string `json:"created_at"`
	}

	events := []Event{}
	for rows.Next() {
		var e Event
		if err := rows.Scan(&e.ID, &e.ActuatorID, &e.State, &e.Source, &e.Details, &e.CreatedAt); err != nil {
			continue
		}
		events = append(events, e)
	}
	if err := rows.Err(); err != nil {
		http.Error(w, "error leyendo eventos", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(events)
}
