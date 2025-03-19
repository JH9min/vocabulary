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

func (u *User) SaveWord(db *sql.DB, word string, definitions []VocabularyDefinition) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 插入或更新主表
	result, err := tx.Exec(`
		INSERT INTO vocabularies (user_id, word, status) 
		VALUES (?, ?, 'active')
		ON DUPLICATE KEY UPDATE 
			status = 'active'
	`, u.ID, word)
	if err != nil {
		return err
	}

	// 獲取vocabulary_id
	var vocabularyID int64
	if id, err := result.LastInsertId(); err == nil {
		vocabularyID = id
	} else {
		// 如果是更新現有記錄，需要查詢ID
		err = tx.QueryRow("SELECT id FROM vocabularies WHERE user_id = ? AND word = ?", u.ID, word).Scan(&vocabularyID)
		if err != nil {
			return err
		}
	}

	// 刪除舊的定義
	_, err = tx.Exec("DELETE FROM vocabulary_definitions WHERE vocabulary_id = ?", vocabularyID)
	if err != nil {
		return err
	}

	// 插入新的定義
	for _, def := range definitions {
		_, err = tx.Exec(`
			INSERT INTO vocabulary_definitions (vocabulary_id, part_of_speech, definition, example) 
			VALUES (?, ?, ?, ?)
		`, vocabularyID, def.PartOfSpeech, def.Definition, def.Example)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (u *User) GetVocabularies(db *sql.DB) ([]Vocabulary, error) {
	// 先查詢所有單字
	rows, err := db.Query(`
		SELECT id, user_id, word, status, tested, created_at 
		FROM vocabularies 
		WHERE user_id = ? AND status = 'active' 
		ORDER BY created_at DESC
	`, u.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var vocabularies []Vocabulary
	for rows.Next() {
		var v Vocabulary
		err := rows.Scan(&v.ID, &v.UserID, &v.Word, &v.Status, &v.Tested, &v.CreatedAt)
		if err != nil {
			return nil, err
		}
		vocabularies = append(vocabularies, v)
	}

	// 查詢每個單字的定義
	for i := range vocabularies {
		rows, err := db.Query(`
			SELECT id, vocabulary_id, part_of_speech, definition, example, created_at 
			FROM vocabulary_definitions 
			WHERE vocabulary_id = ?
			ORDER BY id
		`, vocabularies[i].ID)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		for rows.Next() {
			var def VocabularyDefinition
			err := rows.Scan(&def.ID, &def.VocabularyID, &def.PartOfSpeech, &def.Definition, &def.Example, &def.CreatedAt)
			if err != nil {
				return nil, err
			}
			vocabularies[i].Definitions = append(vocabularies[i].Definitions, def)
		}
	}

	return vocabularies, nil
}

func (u *User) GetVocabularyByWord(db *sql.DB, word string) (*Vocabulary, error) {
	// 先查詢主表
	var v Vocabulary
	err := db.QueryRow(`
		SELECT id, user_id, word, status, tested, created_at 
		FROM vocabularies 
		WHERE user_id = ? AND word = ? AND status = 'active'
	`, u.ID, word).Scan(&v.ID, &v.UserID, &v.Word, &v.Status, &v.Tested, &v.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	// 查詢定義
	rows, err := db.Query(`
		SELECT id, vocabulary_id, part_of_speech, definition, example, created_at 
		FROM vocabulary_definitions 
		WHERE vocabulary_id = ?
		ORDER BY id
	`, v.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var def VocabularyDefinition
		err := rows.Scan(&def.ID, &def.VocabularyID, &def.PartOfSpeech, &def.Definition, &def.Example, &def.CreatedAt)
		if err != nil {
			return nil, err
		}
		v.Definitions = append(v.Definitions, def)
	}

	return &v, nil
}

func (u *User) RemoveWord(db *sql.DB, wordID int64) error {
	_, err := db.Exec(`
		UPDATE vocabularies 
		SET status = 'removed' 
		WHERE id = ? AND user_id = ?
	`, wordID, u.ID)
	return err
}

func (u *User) UpdateTestedStatus(db *sql.DB, wordID int64, tested bool) error {
	_, err := db.Exec(`
		UPDATE vocabularies 
		SET tested = ? 
		WHERE id = ? AND user_id = ?
	`, tested, wordID, u.ID)
	return err
}

func (u *User) SaveTestResult(db *sql.DB, wordID string, correct bool) error {
	query := `INSERT INTO test_results (user_id, word_id, correct, created_at) VALUES (?, ?, ?, ?)`
	_, err := db.Exec(query, u.ID, wordID, correct, time.Now())
	return err
}
