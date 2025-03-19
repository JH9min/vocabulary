package models

import (
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
