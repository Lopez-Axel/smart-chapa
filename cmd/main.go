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
	authmiddleware "smart-chapa/internal/middleware"
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

	dvh := handlers.NewDeviceHandler(database)
	ah := handlers.NewAuthHandler(database)
	acth := handlers.NewActuatorHandler(database, mqttClient)
	hoh := handlers.NewHouseHandler(database)

	jwtMiddleware := authmiddleware.JWTMiddleware(os.Getenv("JWT_SECRET"))

	r.Route("/api", func(r chi.Router) {
		r.Post("/auth/register", ah.Register)
		r.Post("/auth/login", ah.Login)

		r.Group(func(r chi.Router) {
			r.Use(jwtMiddleware)

			r.Post("/devices", dvh.Create)
			r.Get("/devices", dvh.List)
			r.Delete("/devices/{id}", dvh.Delete)

			r.Post("/actuators", acth.Create)
			r.Get("/actuators", acth.List)
			r.Get("/actuators/{id}", acth.Get)
			r.Post("/actuators/{id}/on", acth.TurnOn)
			r.Post("/actuators/{id}/off", acth.TurnOff)
			r.Get("/actuators/{id}/events", acth.Events)

			r.Post("/houses", hoh.Create)
			r.Get("/houses", hoh.List)
			r.Get("/houses/{id}/devices", hoh.GetDevices)
			r.Post("/houses/{id}/members", hoh.AddMember)
		})
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Println("Servidor corriendo en puerto", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}
