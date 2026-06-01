package main

import (
    "log"
    "net/http"
    "os"

    "github.com/go-chi/chi/v5"
    "github.com/go-chi/chi/v5/middleware"
    "github.com/joho/godotenv"

    "smart-chapa/internal/db"
    "smart-chapa/internal/handlers"
)

func main() {
    godotenv.Load()

    database, err := db.Init()
    if err != nil {
        log.Fatal("Error iniciando DB:", err)
    }
    defer database.Close()

    r := chi.NewRouter()
    r.Use(middleware.Logger)
    r.Use(middleware.Recoverer)

    dh := handlers.NewDoorHandler(database)
    dvh := handlers.NewDeviceHandler(database)

    r.Route("/api", func(r chi.Router) {
        r.Post("/door/open", dh.Open)
        r.Get("/door/pending", dh.Pending)
        r.Post("/door/confirm", dh.Confirm)
        r.Get("/door/events", dh.Events)

        r.Post("/devices", dvh.Create)
        r.Get("/devices", dvh.List)
    })

    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
    }

    log.Println("Servidor corriendo en puerto", port)
    log.Fatal(http.ListenAndServe(":"+port, r))
}
