package main

import (
	"log"
	"math"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type StressRequest struct {
	RequestID string `json:"requestid" binding:"required"`
	UUID      string `json:"uuid" binding:"required"`
	Length    int    `json:"length" binding:"required"`
}

func main() {
	router := gin.Default()

	router.POST("/v1/stress", handleStress)
	router.GET("/healthcheck", handleHealthCheck)

	log.Println("Starting stress app on port 8080")
	router.Run(":8080")
}

func handleStress(c *gin.Context) {
	var req StressRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	start := time.Now()

	// CPU 부하 발생 (length만큼 반복 연산)
	for i := 0; i < req.Length; i++ {
		_ = math.Sqrt(float64(i)) // 단순 계산
	}

	duration := time.Since(start).Milliseconds()

	log.Printf("[stress] length=%d duration=%dms requestid=%s uuid=%s",
		req.Length, duration, req.RequestID, req.UUID)

	c.JSON(http.StatusCreated, gin.H{
		"status":      "stressed",
		"duration_ms": duration,
	})
}

func handleHealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
