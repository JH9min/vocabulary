package models

import (
	"database/sql"
	"time"
)

type Vocabulary struct {
	ID          int64
	UserID      int64
	Word        string
	Status      string
	Tested      bool
	CreatedAt   time.Time
	Definitions []VocabularyDefinition
}

type VocabularyDefinition struct {
	ID           int64
	VocabularyID int64
	PartOfSpeech string
	Definition   string
	Example      string
	CreatedAt    time.Time
}

type TestResult struct {
	ID        int64
	UserID    int64
	WordID    int64
	Correct   bool
	CreatedAt time.Time
}

// Get retrieves a vocabulary word and its definitions from the database
func (v *Vocabulary) Get(db *sql.DB) error {
	// Get vocabulary word
	query := `
		SELECT id, user_id, word, status, tested, created_at
		FROM vocabularies
		WHERE id = ?
	`
	err := db.QueryRow(query, v.ID).Scan(
		&v.ID,
		&v.UserID,
		&v.Word,
		&v.Status,
		&v.Tested,
		&v.CreatedAt,
	)
	if err != nil {
		return err
	}

	// Get definitions
	query = `
		SELECT id, vocabulary_id, part_of_speech, definition, example, created_at
		FROM vocabulary_definitions
		WHERE vocabulary_id = ?
	`
	rows, err := db.Query(query, v.ID)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var def VocabularyDefinition
		err := rows.Scan(
			&def.ID,
			&def.VocabularyID,
			&def.PartOfSpeech,
			&def.Definition,
			&def.Example,
			&def.CreatedAt,
		)
		if err != nil {
			return err
		}
		v.Definitions = append(v.Definitions, def)
	}

	return rows.Err()
}

// Save saves the vocabulary word and its definitions to the database
func (v *Vocabulary) Save(db *sql.DB) error {
	// Begin transaction
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Update vocabulary word
	query := `
		UPDATE vocabularies
		SET word = ?, status = ?, tested = ?
		WHERE id = ?
	`
	_, err = tx.Exec(query, v.Word, v.Status, v.Tested, v.ID)
	if err != nil {
		return err
	}

	// Insert new definitions
	query = `
		INSERT INTO vocabulary_definitions (vocabulary_id, part_of_speech, definition, example)
		VALUES (?, ?, ?, ?)
	`
	for _, def := range v.Definitions {
		_, err = tx.Exec(query, v.ID, def.PartOfSpeech, def.Definition, def.Example)
		if err != nil {
			return err
		}
	}

	// Commit transaction
	return tx.Commit()
}

// DeleteDefinitions deletes all definitions for this vocabulary word
func (v *Vocabulary) DeleteDefinitions(db *sql.DB) error {
	_, err := db.Exec("DELETE FROM vocabulary_definitions WHERE vocabulary_id = ?", v.ID)
	return err
}

func (v *Vocabulary) Remove(db *sql.DB) error {
	_, err := db.Exec(`
		UPDATE vocabularies 
		SET status = 'removed' 
		WHERE id = ?
	`, v.ID)
	return err
}

// GetByUserID retrieves all vocabulary words for a user
func GetByUserID(db *sql.DB, userID int64) ([]Vocabulary, error) {
	// 先查詢所有單字
	rows, err := db.Query(`
		SELECT id, user_id, word, status, tested, created_at 
		FROM vocabularies 
		WHERE user_id = ? AND status = 'active' 
		ORDER BY created_at DESC
	`, userID)
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

// GetByWord retrieves a vocabulary word by its word text
func GetByWord(db *sql.DB, userID int64, word string) (*Vocabulary, error) {
	// 先查詢主表
	var v Vocabulary
	err := db.QueryRow(`
		SELECT id, user_id, word, status, tested, created_at 
		FROM vocabularies 
		WHERE user_id = ? AND word = ? AND status = 'active'
	`, userID, word).Scan(&v.ID, &v.UserID, &v.Word, &v.Status, &v.Tested, &v.CreatedAt)

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

// Create creates a new vocabulary word with its definitions
func Create(db *sql.DB, userID int64, word string, definitions []VocabularyDefinition) error {
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
	`, userID, word)
	if err != nil {
		return err
	}

	// 獲取vocabulary_id
	var vocabularyID int64
	if id, err := result.LastInsertId(); err == nil {
		vocabularyID = id
	} else {
		// 如果是更新現有記錄，需要查詢ID
		err = tx.QueryRow("SELECT id FROM vocabularies WHERE user_id = ? AND word = ?", userID, word).Scan(&vocabularyID)
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

// UpdateTestedStatus updates the tested status of a vocabulary word
func (v *Vocabulary) UpdateTestedStatus(db *sql.DB, tested bool) error {
	_, err := db.Exec(`
		UPDATE vocabularies 
		SET tested = ? 
		WHERE id = ? AND user_id = ?
	`, tested, v.ID, v.UserID)
	return err
}
