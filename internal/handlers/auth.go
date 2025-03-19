package handlers

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"time"
	"vocabulary/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"golang.org/x/crypto/bcrypt"
)

var db *sql.DB

func Init(database *sql.DB) {
	db = database
}

func ShowLogin(c *gin.Context) {
	c.HTML(http.StatusOK, "login.html", gin.H{
		"title": "Login",
	})
}

func ShowRegister(c *gin.Context) {
	c.HTML(http.StatusOK, "register.html", gin.H{
		"title": "Register",
	})
}

func Login(c *gin.Context) {
	log.Println("🚀 Login function executed!") // 登入函式是否執行
	username := c.PostForm("username")
	password := c.PostForm("password")
	log.Println("📌 Received Login Request - Username:", username, "Password:", password)

	user, err := models.GetUserByUsername(db, username)
	if err != nil {
		c.HTML(http.StatusBadRequest, "login.html", gin.H{
			"error":    "Error checking username",
			"username": username,
		})
		return
	}

	if user == nil {
		c.HTML(http.StatusBadRequest, "login.html", gin.H{
			"error":    "Username does not exist",
			"username": username,
		})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		// 密碼錯誤
		log.Println("🔒 Hashed Password from DB:", user.Password)
		log.Println("🔑 User Input Password:", password)
		log.Println("❌ Password comparison failed:", err) // 記錄錯誤原因
		c.HTML(http.StatusBadRequest, "login.html", gin.H{
			"error":    "Incorrect password",
			"username": username, // 保留用戶輸入的用戶名
		})
		return
	}

	// 生成 JWT token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": user.ID,
		"exp":     time.Now().Add(time.Hour * 24).Unix(),
	})

	tokenString, err := token.SignedString([]byte(os.Getenv("JWT_SECRET")))
	if err != nil {
		c.HTML(http.StatusInternalServerError, "login.html", gin.H{
			"error":    "Error generating token",
			"username": username, // 保留用戶輸入的用戶名
		})
		return
	}

	c.SetCookie("token", tokenString, 3600*24, "/", "", false, true)
	c.Redirect(http.StatusFound, "/news")
}

func Register(c *gin.Context) {
	username := c.PostForm("username")
	password := c.PostForm("password")

	// 檢查用戶名是否已存在
	existingUser, err := models.GetUserByUsername(db, username)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "register.html", gin.H{
			"error":    "Error checking username",
			"username": username,
		})
		return
	}

	if existingUser != nil {
		c.HTML(http.StatusBadRequest, "register.html", gin.H{
			"error":    "Username already exists",
			"username": username,
		})
		return
	}

	// 加密密碼
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "register.html", gin.H{
			"error":    "Error processing registration",
			"username": username,
		})
		return
	}

	// 創建用戶
	if err := models.CreateUser(db, username, string(hashedPassword)); err != nil {
		c.HTML(http.StatusInternalServerError, "register.html", gin.H{
			"error":    "Error creating user",
			"username": username,
		})
		return
	}

	c.Redirect(http.StatusFound, "/login")
}

func Logout(c *gin.Context) {
	// Clear the JWT token cookie
	c.SetCookie("token", "", -1, "/", "", false, true)
	c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
}
