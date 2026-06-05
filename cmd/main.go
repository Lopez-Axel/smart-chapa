package main

import (
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/joho/godotenv"

	"smart-chapa/internal/db"
	"smart-chapa/internal/handlers"
)

func newMQTTClient() mqtt.Client {
	opts := mqtt.NewClientOptions().
		AddBroker("tls://" + os.Getenv("MQTT_HOST") + ":8883").
		SetUsername(os.Getenv("MQTT_USER")).
		SetPassword(os.Getenv("MQTT_PASSWORD")).
		SetClientID("go-backend").
		SetAutoReconnect(true)

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Fatal("MQTT error:", token.Error())
	}

	log.Println("MQTT conectado")
	return client
}

func main() {
	godotenv.Load()

	database, err := db.Init()
	if err != nil {
		log.Fatal("Error iniciando DB:", err)
	}
	defer database.Close()

	mqttClient := newMQTTClient()

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	dh := handlers.NewDoorHandler(database, mqttClient)
	dvh := handlers.NewDeviceHandler(database)
	lh := handlers.NewLightHandler(database, mqttClient)

	r.Route("/api", func(r chi.Router) {
		r.Post("/door/open", dh.Open)
		r.Post("/door/confirm", dh.Confirm)
		r.Get("/door/events", dh.Events)

		r.Post("/devices", dvh.Create)
		r.Get("/devices", dvh.List)

		r.Post("/lights/on", lh.TurnOn)
		r.Post("/lights/off", lh.TurnOff)
		r.Get("/lights/events", lh.Events)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Println("Servidor corriendo en puerto", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}
