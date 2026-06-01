package db

import (
    "database/sql"
    _ "modernc.org/sqlite"
)

func Init() (*sql.DB, error) {
    db, err := sql.Open("sqlite", "smart_chapa.db")
    if err != nil {
        return nil, err
    }

    if err := createTables(db); err != nil {
        return nil, err
    }

    return db, nil
}

func createTables(db *sql.DB) error {
    _, err := db.Exec(`
        CREATE TABLE IF NOT EXISTS devices (
            id         INTEGER PRIMARY KEY AUTOINCREMENT,
            name       TEXT NOT NULL,
            token      TEXT UNIQUE NOT NULL,
            created_at DATETIME DEFAULT CURRENT_TIMESTAMP
        );

        CREATE TABLE IF NOT EXISTS door_events (
            id         INTEGER PRIMARY KEY AUTOINCREMENT,
            device_id  INTEGER REFERENCES devices(id),
            action     TEXT NOT NULL,
            source     TEXT NOT NULL,
            created_at DATETIME DEFAULT CURRENT_TIMESTAMP
        );

        CREATE TABLE IF NOT EXISTS pending_commands (
            id         INTEGER PRIMARY KEY AUTOINCREMENT,
            device_id  INTEGER REFERENCES devices(id),
            command    TEXT NOT NULL,
            executed   INTEGER DEFAULT 0,
            created_at DATETIME DEFAULT CURRENT_TIMESTAMP
        );
    `)
    return err
}
