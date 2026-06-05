package models

import "time"

type Device struct {
    ID        int64     `json:"id"`
    Name      string    `json:"name"`
    Token     string    `json:"token"`
    CreatedAt time.Time `json:"created_at"`
}

type DoorEvent struct {
    ID        int64     `json:"id"`
    DeviceID  int64     `json:"device_id"`
    Action    string    `json:"action"`
    Source    string    `json:"source"`
    CreatedAt time.Time `json:"created_at"`
}

type PendingCommand struct {
    ID        int64     `json:"id"`
    DeviceID  int64     `json:"device_id"`
    Command   string    `json:"command"`
    Executed  bool      `json:"executed"`
    CreatedAt time.Time `json:"created_at"`
}

type LightEvent struct {
    ID        int64     `json:"id"`
    DeviceID  int64     `json:"device_id"`
    State     string    `json:"state"`
    Source    string    `json:"source"`
    CreatedAt time.Time `json:"created_at"`
}
