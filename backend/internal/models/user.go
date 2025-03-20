package models

import (
	"database/sql"
	"log"
	"time"
)

type User struct {
	ID        int64
	Username  string
	Password  string
	CreatedAt time.Time
}

func CreateUser(db *sql.DB, username, password string) error {
	query := `INSERT INTO users (username, password, created_at) VALUES (?, ?, ?)`
	_, err := db.Exec(query, username, password, time.Now())
	return err
}

func GetUserByUsername(db *sql.DB, username string) (*User, error) {
	user := &User{}
	query := `SELECT id, username, password, created_at FROM users WHERE username = ?`
	err := db.QueryRow(query, username).Scan(&user.ID, &user.Username, &user.Password, &user.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Println("⚠️ User not found:", username)
			return nil, nil // 返回 nil 而非錯誤
		}
		log.Println("❌ Database error:", err)
		return nil, err
	}

	log.Println("✅ Found user:", user.Username)
	log.Println("🔒 Hashed password from DB:", user.Password)
	return user, nil
}

func (u *User) SaveTestResult(db *sql.DB, wordID string, correct bool) error {
	query := `INSERT INTO test_results (user_id, word_id, correct, created_at) VALUES (?, ?, ?, ?)`
	_, err := db.Exec(query, u.ID, wordID, correct, time.Now())
	return err
}
