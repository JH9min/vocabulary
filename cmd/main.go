package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"vocabulary/internal/handlers"
	"vocabulary/internal/middleware"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
)

var db *sql.DB

func main() {
	// 載入環境變數
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found")
	}

	// 設置資料庫連接
	var err error
	db, err = sql.Open("mysql", os.Getenv("DB_CONNECTION"))
	if err != nil {
		log.Fatal("Error connecting to the database:", err)
	}
	defer db.Close()

	// 測試資料庫連接
	if err = db.Ping(); err != nil {
		log.Fatal("Error pinging the database:", err)
	}

	// 初始化handlers
	handlers.Init(db)

	// 初始化Gin路由
	r := gin.Default()

	// 設置 session middleware
	store := cookie.NewStore([]byte(os.Getenv("JWT_SECRET")))
	r.Use(sessions.Sessions("vocabulary_session", store))

	// 獲取當前工作目錄
	wd, err := os.Getwd()
	log.Println("Current working directory:", wd)
	if err != nil {
		log.Fatal("Error getting working directory:", err)
	}

	// 載入所有HTML模板，包括子目錄
	templatesDir := filepath.Join(wd, "templates")
	r.LoadHTMLGlob(filepath.Join(templatesDir, "*/*.html"))

	// 設置靜態文件路徑
	r.Static("/static", filepath.Join(wd, "static"))

	// 路由設置
	setupRoutes(r)

	// 啟動服務器
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	r.Run(":" + port)
}

func setupRoutes(r *gin.Engine) {
	// 首頁重定向到登入頁面
	r.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusFound, "/login")
	})

	// 認證相關路由
	r.GET("/login", handlers.ShowLogin)
	r.POST("/login", handlers.Login)
	r.GET("/register", handlers.ShowRegister)
	r.POST("/register", handlers.Register)
	r.POST("/logout", handlers.Logout)

	// 需要認證的路由組
	auth := r.Group("")
	auth.Use(middleware.AuthRequired())
	{
		// 新聞相關
		auth.GET("/news", handlers.ShowNewsReader)
		auth.POST("/news/fetch", handlers.FetchNews)

		// 單字相關
		auth.GET("/vocabulary", handlers.ShowVocabulary)
		auth.POST("/vocabulary/lookup", handlers.LookupWord)
		auth.POST("/vocabulary/save", handlers.SaveWord)
		auth.DELETE("/vocabulary/:id", handlers.DeleteWord)

		// 單字卡測驗
		auth.GET("/flashcards", handlers.ShowFlashcards)
		auth.GET("/flashcards/test", handlers.StartTest)
		auth.POST("/flashcards/result", handlers.SaveTestResult)
	}

	// 將所有未定義的路由重定向到登入頁面
	r.NoRoute(func(c *gin.Context) {
		c.Redirect(http.StatusFound, "/login")
	})
}
