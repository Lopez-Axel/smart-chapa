package models

import "time"

type User struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

type House struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Address   string    `json:"address"`
	CreatedAt time.Time `json:"created_at"`
}

type Device struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Token     string    `json:"token"`
	UserID    int64     `json:"user_id"`
	HouseID   int64     `json:"house_id"`
	CreatedAt time.Time `json:"created_at"`
}

type ActuatorType string

const (
	TypeDoor   ActuatorType = "door"
	TypeLights ActuatorType = "lights"
	TypeGate   ActuatorType = "gate"
	TypeWindow ActuatorType = "window"
)

type Actuator struct {
	ID        int64     `json:"id"`
	DeviceID  int64     `json:"device_id"`
	Name      string    `json:"name"`
	Type      string    `json:"type"`
	RelayNum  int       `json:"relay_num"`
	State     string    `json:"state"`
	CreatedAt time.Time `json:"created_at"`
}

type ActuatorEvent struct {
	ID          int64     `json:"id"`
	ActuatorID  int64     `json:"actuator_id"`
	State       string    `json:"state"`
	Source      string    `json:"source"`
	CreatedAt   time.Time `json:"created_at"`
}
