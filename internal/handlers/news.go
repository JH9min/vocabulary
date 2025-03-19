package handlers

import (
	"io"
	"net/http"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/gin-gonic/gin"
)

func ShowNewsReader(c *gin.Context) {
	c.HTML(http.StatusOK, "reader.html", gin.H{
		"title": "News Reader",
	})
}

func FetchNews(c *gin.Context) {
	url := c.PostForm("url")
	if url == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "URL is required"})
		return
	}

	// 發送HTTP請求獲取新聞內容
	resp, err := http.Get(url)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch news"})
		return
	}
	defer resp.Body.Close()

	// 讀取響應內容
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read response"})
		return
	}

	// 使用goquery解析HTML
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse HTML"})
		return
	}
	doc.Find("header, footer, nav, aside, script, style, .ad, .comments, .related-articles").Remove()
	// 提取文章內容（這裡的選擇器需要根據目標網站調整）
	var content string
	doc.Find("p").Each(func(i int, s *goquery.Selection) {
		text := strings.TrimSpace(s.Text())
		if text != "" {
			content += "<p>" + text + "</p>"
		}
	})

	// If no paragraphs found, try getting the body content
	if content == "" {
		content = doc.Find("body").Text()
		// Split content by newlines and create paragraphs
		paragraphs := strings.Split(content, "\n")
		content = ""
		for _, p := range paragraphs {
			if text := strings.TrimSpace(p); text != "" {
				content += "<p>" + text + "</p>"
			}
		}
	}

	c.Header("Content-Type", "text/html")
	c.String(http.StatusOK, content)
}
