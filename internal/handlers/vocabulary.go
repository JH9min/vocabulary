package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"vocabulary/internal/models"

	"github.com/gin-gonic/gin"
)

type DictionaryResponse struct {
	Word       string `json:"word"`
	Definition string `json:"definition"`
}

func ShowVocabulary(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// 檢查是否為測試環境
	if os.Getenv("SKIP_DB") == "true" {
		c.HTML(http.StatusOK, "list.html", gin.H{
			"title": "My Vocabulary",
			"vocabularies": []models.Vocabulary{
				{
					ID:     1,
					UserID: 1,
					Word:   "example",
					Status: "active",
					Tested: true,
					Definitions: []models.VocabularyDefinition{
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

	// 獲取用戶的單字列表
	user := &models.User{ID: userID.(int64)}
	vocabularies, err := user.GetVocabularies(db)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching vocabularies"})
		return
	}
	log.Println("Rendering vocabulary page") // 應該在 console 看到這行
	c.HTML(http.StatusOK, "list.html", gin.H{
		"title":        "My Vocabulary",
		"vocabularies": vocabularies,
	})
}

func LookupWord(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	word := c.PostForm("word")
	if word == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Word is required"})
		return
	}

	// 檢查是否為測試環境
	if os.Getenv("SKIP_DB") == "true" {
		c.JSON(http.StatusOK, gin.H{
			"word": word,
			"definitions": []map[string]string{
				{
					"partOfSpeech": "noun",
					"definition":   "Test definition for " + word,
					"example":      "This is a test example for " + word,
				},
			},
			"exists": false,
		})
		return
	}

	// 先檢查用戶的詞彙庫中是否已有此單字
	user := &models.User{ID: userID.(int64)}
	existingWord, err := user.GetVocabularyByWord(db, word)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error checking existing word"})
		return
	}

	// 如果單字已存在且為活躍狀態，直接返回
	if existingWord != nil {
		// 將定義轉換為API格式
		var definitions []map[string]string
		for _, def := range existingWord.Definitions {
			definition := map[string]string{
				"partOfSpeech": def.PartOfSpeech,
				"definition":   def.Definition,
			}
			if def.Example != "" {
				definition["example"] = def.Example
			}
			definitions = append(definitions, definition)
		}

		c.JSON(http.StatusOK, gin.H{
			"word":        existingWord.Word,
			"definitions": definitions,
			"exists":      true,
			"tested":      existingWord.Tested,
		})
		return
	}

	// 如果單字不存在，則查詢 Dictionary API
	url := fmt.Sprintf("https://api.dictionaryapi.dev/api/v2/entries/en/%s", url.QueryEscape(word))
	resp, err := http.Get(url)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to lookup word"})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Word not found"})
		return
	}

	var result []struct {
		Meanings []struct {
			PartOfSpeech string `json:"partOfSpeech"`
			Definitions  []struct {
				Definition string `json:"definition"`
				Example    string `json:"example"`
			} `json:"definitions"`
		} `json:"meanings"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse dictionary response"})
		return
	}

	if len(result) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "No definitions found"})
		return
	}

	// 整理定義
	var definitions []map[string]string
	for _, entry := range result {
		for _, meaning := range entry.Meanings {
			for _, def := range meaning.Definitions {
				definition := map[string]string{
					"partOfSpeech": meaning.PartOfSpeech,
					"definition":   def.Definition,
				}
				if def.Example != "" {
					definition["example"] = def.Example
				}
				definitions = append(definitions, definition)
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"word":        word,
		"definitions": definitions,
		"exists":      false,
	})
}

func SaveWord(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	word := c.PostForm("word")
	definitionsJSON := c.PostForm("definitions")

	if word == "" || definitionsJSON == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Word and definitions are required"})
		return
	}

	// 檢查是否為測試環境
	if os.Getenv("SKIP_DB") == "true" {
		c.JSON(http.StatusOK, gin.H{
			"message":           "Word saved successfully (test mode)",
			"definitions_saved": 1,
			"total_definitions": 1,
		})
		return
	}

	// URL decode the definitions JSON string
	decodedJSON, err := url.QueryUnescape(definitionsJSON)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid definitions format"})
		return
	}

	var definitions []map[string]string
	if err := json.Unmarshal([]byte(decodedJSON), &definitions); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid definitions format"})
		return
	}

	// 限制定義數量最多為 5 個
	if len(definitions) > 5 {
		definitions = definitions[:5]
	}

	user := &models.User{ID: userID.(int64)}

	// 檢查單字是否已存在
	existingWord, err := user.GetVocabularyByWord(db, word)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error checking existing word"})
		return
	}

	if existingWord != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Word already exists in your vocabulary",
			"word":  existingWord,
		})
		return
	}

	// 轉換定義格式
	var vocabDefinitions []models.VocabularyDefinition
	for _, def := range definitions {
		vocabDef := models.VocabularyDefinition{
			PartOfSpeech: def["partOfSpeech"],
			Definition:   def["definition"],
			Example:      def["example"],
		}
		vocabDefinitions = append(vocabDefinitions, vocabDef)
	}

	// 保存單字和定義
	if err := user.SaveWord(db, word, vocabDefinitions); err != nil {
		log.Println("Error saving word:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error saving word"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":           "Word saved successfully",
		"definitions_saved": len(vocabDefinitions),
		"total_definitions": len(definitions),
	})
}

func DeleteWord(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	wordIDStr := c.Param("id")
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

	user := &models.User{ID: userID.(int64)}
	if err := user.RemoveWord(db, wordID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error deleting word"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}
