package handlers

import (
    "database/sql"
    "encoding/json"
    "net/http"
)

type DoorHandler struct {
    db *sql.DB
}

func NewDoorHandler(db *sql.DB) *DoorHandler {
    return &DoorHandler{db: db}
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

    if body.Source == "" {
        body.Source = "app"
    }

    _, err := h.db.Exec(
        "INSERT INTO pending_commands (device_id, command) VALUES (?, 'open')",
        body.DeviceID,
    )
    if err != nil {
        http.Error(w, "error creando comando", http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (h *DoorHandler) Pending(w http.ResponseWriter, r *http.Request) {
    var row struct {
        ID      int64  `json:"id"`
        Command string `json:"command"`
    }

    err := h.db.QueryRow(`
        SELECT id, command FROM pending_commands
        WHERE executed = 0
        ORDER BY created_at ASC
        LIMIT 1
    `).Scan(&row.ID, &row.Command)

    if err == sql.ErrNoRows {
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(map[string]string{"command": "none"})
        return
    }
    if err != nil {
        http.Error(w, "error consultando comandos", http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(row)
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

    h.db.Exec("UPDATE pending_commands SET executed = 1 WHERE id = ?", body.CommandID)

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
