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

	db.Exec("PRAGMA foreign_keys = ON")

	if err := createTables(db); err != nil {
		return nil, err
	}

	migrate(db)
	return db, nil
}

func createTables(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id            INTEGER PRIMARY KEY AUTOINCREMENT,
			name          TEXT NOT NULL,
			email         TEXT UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			created_at    DATETIME DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE IF NOT EXISTS houses (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			name       TEXT NOT NULL,
			address    TEXT,
			city       TEXT DEFAULT '',
			country    TEXT DEFAULT '',
			latitude   REAL DEFAULT 0,
			longitude  REAL DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE IF NOT EXISTS user_houses (
			user_id  INTEGER REFERENCES users(id),
			house_id INTEGER REFERENCES houses(id),
			role     TEXT NOT NULL DEFAULT 'owner',
			PRIMARY KEY (user_id, house_id)
		);

		CREATE TABLE IF NOT EXISTS devices (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			name       TEXT NOT NULL,
			token      TEXT UNIQUE NOT NULL,
			user_id    INTEGER REFERENCES users(id),
			house_id   INTEGER REFERENCES houses(id),
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE IF NOT EXISTS actuators (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			device_id  INTEGER REFERENCES devices(id),
			name       TEXT NOT NULL,
			type       TEXT NOT NULL,
			relay_num  INTEGER NOT NULL,
			state      TEXT DEFAULT 'off',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE IF NOT EXISTS actuator_events (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			actuator_id INTEGER REFERENCES actuators(id),
			state       TEXT NOT NULL,
			source      TEXT NOT NULL,
			details     TEXT DEFAULT '',
			created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
		);
	`)
	return err
}

func migrate(db *sql.DB) {
	db.Exec("DROP TABLE IF EXISTS door_events")
	db.Exec("DROP TABLE IF EXISTS light_events")
	db.Exec("DROP TABLE IF EXISTS pending_commands")

	db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_devices_name_user ON devices(name, user_id)")

	db.Exec("ALTER TABLE houses ADD COLUMN city TEXT DEFAULT ''")
	db.Exec("ALTER TABLE houses ADD COLUMN country TEXT DEFAULT ''")
	db.Exec("ALTER TABLE houses ADD COLUMN latitude REAL DEFAULT 0")
	db.Exec("ALTER TABLE houses ADD COLUMN longitude REAL DEFAULT 0")
	db.Exec("ALTER TABLE actuator_events ADD COLUMN details TEXT DEFAULT ''")
}
