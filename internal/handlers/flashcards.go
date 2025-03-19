package handlers

import (
	"net/http"
	"os"
	"strconv"
	"vocabulary/internal/models"

	"github.com/gin-gonic/gin"
)

func ShowFlashcards(c *gin.Context) {
	c.HTML(http.StatusOK, "index.html", gin.H{
		"title": "Flashcards",
	})
}

func StartTest(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// 檢查是否為測試環境
	if os.Getenv("SKIP_DB") == "true" {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"words": []gin.H{
				{
					"id":     1,
					"word":   "example",
					"tested": false,
					"Definitions": []models.VocabularyDefinition{
						{
							ID:           1,
							VocabularyID: 1,
							PartOfSpeech: "noun",
							Definition:   "a representative form or pattern",
							Example:      "This is an example of a test word.",
						},
					},
				},
			},
		})
		return
	}

	vocabularies, err := models.GetByUserID(db, userID.(int64))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching vocabularies"})
		return
	}

	if len(vocabularies) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"error":   "No words in your vocabulary",
		})
		return
	}

	// 將單字轉換為前端需要的格式
	var words []gin.H
	for _, v := range vocabularies {
		word := gin.H{
			"id":          v.ID,
			"word":        v.Word,
			"tested":      v.Tested,
			"Definitions": v.Definitions,
		}
		words = append(words, word)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"words":   words,
	})
}

func SaveTestResult(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	wordIDStr := c.PostForm("word_id")
	if wordIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Word ID is required"})
		return
	}

	// 檢查是否為測試環境
	if os.Getenv("SKIP_DB") == "true" {
		c.JSON(http.StatusOK, gin.H{"success": true})
		return
	}

	wordID, err := strconv.ParseInt(wordIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid word ID format"})
		return
	}

	testedStr := c.PostForm("tested")
	tested := testedStr == "true"

	// 獲取單字
	vocabulary := &models.Vocabulary{ID: wordID, UserID: userID.(int64)}
	if err := vocabulary.UpdateTestedStatus(db, tested); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error updating word status"})
		return
	}

	// 如果需要，保存測試結果
	if tested {
		user := &models.User{ID: userID.(int64)}
		if err := user.SaveTestResult(db, wordIDStr, true); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error saving test result"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}
