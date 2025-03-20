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
	"github.com/golang-jwt/jwt/v4"
	"github.com/joho/godotenv"
)

var db *sql.DB

func main() {
	// 嘗試載入 .env 檔案
	envPath := getEnvPath()
	if err := godotenv.Load(envPath); err != nil {
		log.Printf("Warning: .env file not found at %s", envPath)
	}

	// 確認是否需要資料庫
	skipDB := os.Getenv("SKIP_DB") == "true"
	if !skipDB {
		// 根據環境選擇資料庫連線字串
		dbConn := os.Getenv("DB_CONNECTION_LOCAL") // 預設本地
		if os.Getenv("DOCKER_ENV") == "true" {     // 如果是在 Docker 內部執行
			dbConn = os.Getenv("DB_CONNECTION_DOCKER")
		}
		log.Println("Using DB Connection:", dbConn)

		// 設置資料庫連接
		var err error
		db, err = sql.Open("mysql", dbConn)
		if err != nil {
			log.Fatal("Error connecting to the database:", err)
		}
		defer db.Close()

		// 測試資料庫連接
		if err = db.Ping(); err != nil {
			log.Fatal("Error pinging the database:", err)
		}
	}
	// 初始化 handlers
	handlers.Init(db)

	// 初始化 Gin 路由
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

	// 載入所有 HTML 模板，包括子目錄
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

// 取得正確的 .env 路徑
func getEnvPath() string {
	// 預設環境變數位置
	envPath := "../../.env" // 適用於 `backend/cmd/main.go` 執行時

	// 如果是 Docker 環境，則使用 `/app/.env`
	if os.Getenv("DOCKER_ENV") == "true" {
		envPath = "/app/.env"
	}
	return envPath
}

// 測試環境的 Middleware
func testEnvironmentMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if os.Getenv("SKIP_DB") == "true" {
			// 檢查是否為測試用戶
			token, _ := c.Cookie("token")
			if token == "" {
				c.Redirect(http.StatusFound, "/login")
				c.Abort()
				return
			}

			// 解析 token
			claims := jwt.MapClaims{}
			_, err := jwt.ParseWithClaims(token, claims, func(token *jwt.Token) (interface{}, error) {
				return []byte(os.Getenv("JWT_SECRET")), nil
			})

			if err != nil || claims["user_id"] != float64(1) {
				c.Redirect(http.StatusFound, "/login")
				c.Abort()
				return
			}
		}
		c.Next()
	}
}

// 設置路由
func setupRoutes(r *gin.Engine) {
	// 添加測試環境中間件
	r.Use(testEnvironmentMiddleware())

	// 首頁重定向到登入頁面
	r.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusFound, "/login")
	})

	// 公開路由
	r.GET("/login", handlers.ShowLogin)
	r.POST("/login", handlers.Login)
	r.GET("/register", handlers.ShowRegister)
	r.POST("/register", handlers.Register)
	r.POST("/logout", handlers.Logout)

	// 需要認證的路由
	authorized := r.Group("/")
	authorized.Use(middleware.AuthRequired())
	{
		// 新聞相關
		authorized.GET("/news", handlers.ShowNewsReader)
		authorized.POST("/news/fetch", handlers.FetchNews)

		// 單字相關
		authorized.GET("/vocabulary", handlers.ShowVocabulary)
		authorized.POST("/vocabulary/lookup", handlers.LookupWord)
		authorized.POST("/vocabulary/save", handlers.SaveWord)
		authorized.DELETE("/vocabulary/:id", handlers.DeleteWord)
		authorized.GET("/vocabulary/:id", handlers.GetVocabulary)
		authorized.PUT("/vocabulary/:id", handlers.UpdateVocabulary)

		// 單字卡測驗
		authorized.GET("/flashcards", handlers.ShowFlashcards)
		authorized.GET("/flashcards/test", handlers.StartTest)
		authorized.POST("/flashcards/result", handlers.SaveTestResult)
	}

	// 將所有未定義的路由重定向到登入頁面
	r.NoRoute(func(c *gin.Context) {
		c.Redirect(http.StatusFound, "/login")
	})
}
