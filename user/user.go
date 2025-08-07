package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
)

type UserRequest struct {
	RequestID     string `json:"requestid" binding:"required"`
	UUID          string `json:"uuid" binding:"required"`
	Username      string `json:"username" binding:"required"`
	Email         string `json:"email" binding:"required,email"`
	StatusMessage string `json:"status_message" binding:"required"`
}

type User struct {
	ID            string `json:"id"`
	Username      string `json:"username"`
	Email         string `json:"email"`
	StatusMessage string `json:"status_message"`
}

var db *sql.DB

func main() {
	mysqlUser := os.Getenv("MYSQL_USER")
	mysqlPassword := os.Getenv("MYSQL_PASSWORD")
	mysqlHost := os.Getenv("MYSQL_HOST")
	mysqlPort := os.Getenv("MYSQL_PORT")
	mysqlDBName := os.Getenv("MYSQL_DBNAME")

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true",
		mysqlUser, mysqlPassword, mysqlHost, mysqlPort, mysqlDBName)

	var err error
	db, err = sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("MySQL 연결 실패: %v", err)
	}
	defer db.Close()

	if err = db.Ping(); err != nil {
		log.Fatalf("MySQL Ping 실패: %v", err)
	}

	router := gin.Default()

	router.POST("/v1/user", handleCreateUser)
	router.GET("/v1/user", handleGetUser)
	router.GET("/healthcheck", handleHealthCheck)

	log.Println("User service started on :8080")
	router.Run(":8080")
}

func handleCreateUser(c *gin.Context) {
	var userReq UserRequest
	if err := c.ShouldBindJSON(&userReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}

	query := `INSERT INTO user (id, username, email, status_message) VALUES (?, ?, ?, ?)`
	_, err := db.Exec(query, userReq.RequestID, userReq.Username, userReq.Email, userReq.StatusMessage)
	if err != nil {
		log.Printf("DB insert error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database insert failed"})
		return
	}

	log.Printf("User created: %s (%s)", userReq.Username, userReq.Email)
	c.String(http.StatusCreated, "User created successfully")
}

func handleGetUser(c *gin.Context) {
	requestID := c.Query("requestid")
	uuid := c.Query("uuid")
	email := c.Query("email")

	if requestID == "" || uuid == "" || email == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing query parameters: requestid, uuid, email required"})
		return
	}

	var user User
	query := `SELECT id, username, email, status_message FROM user WHERE email = ?`
	err := db.QueryRow(query, email).Scan(&user.ID, &user.Username, &user.Email, &user.StatusMessage)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		} else {
			log.Printf("DB select error: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		}
		return
	}

	c.JSON(http.StatusOK, user)
}

func handleHealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
