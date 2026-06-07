package handlers

import "database/sql"

func userHasAccess(db *sql.DB, userID, houseID int64) bool {
	var count int
	db.QueryRow(
		"SELECT COUNT(*) FROM user_houses WHERE user_id = ? AND house_id = ?",
		userID, houseID,
	).Scan(&count)
	return count > 0
}
