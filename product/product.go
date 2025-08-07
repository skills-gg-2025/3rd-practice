package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/gin-gonic/gin"
)

type Product struct {
	ID    string `json:"id" dynamodbav:"id"`
	Name  string `json:"name" dynamodbav:"name"`
	Price int    `json:"price" dynamodbav:"price"`
}

type ProductRequest struct {
	RequestID string `json:"requestid"`
	UUID      string `json:"uuid"`
	ID        string `json:"id"`
	Name      string `json:"name"`
	Price     int    `json:"price"`
}

var (
	TableName      string
	TableIndexName string
	dynamoClient   *dynamodb.Client
)

func initEnv() {
	TableName = os.Getenv("TABLE_NAME")
	if TableName == "" {
		log.Fatal("TABLE_NAME 환경변수가 필요합니다.")
	}
	TableIndexName = os.Getenv("TABLE_INDEX_NAME") // 선택 사항
}

func initDynamoDB() {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatalf("DynamoDB 설정 오류: %v", err)
	}
	dynamoClient = dynamodb.NewFromConfig(cfg)
}

func main() {
	initEnv()
	initDynamoDB()

	router := gin.Default()

	router.GET("/healthcheck", healthCheckHandler)
	router.POST("/v1/product", postProductHandler)
	router.GET("/v1/product", getProductHandler)

	log.Println("Server is running on port 8080")
	router.Run(":8080")
}

func healthCheckHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func postProductHandler(c *gin.Context) {
	var req ProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.RequestID == "" || req.UUID == "" || req.ID == "" || req.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "모든 필드(requestid, uuid, id, name, price)는 필수입니다."})
		return
	}

	product := Product{
		ID:    req.ID,
		Name:  req.Name,
		Price: req.Price,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	item, err := attributevalue.MarshalMap(product)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "DynamoDB 마샬링 실패"})
		return
	}

	_, err = dynamoClient.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(TableName),
		Item:      item,
	})
	if err != nil {
		fmt.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "DynamoDB 저장 실패"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "상품이 성공적으로 생성되었습니다."})
}

func getProductHandler(c *gin.Context) {
	id := c.Query("id")
	requestID := c.Query("requestid")
	uuid := c.Query("uuid")

	if id == "" || requestID == "" || uuid == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "쿼리 파라미터(id, requestid, uuid)는 필수입니다."})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	output, err := dynamoClient.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(TableName),
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: id},
		},
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "DynamoDB 조회 실패"})
		return
	}

	if output.Item == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "상품을 찾을 수 없습니다."})
		return
	}

	var product Product
	if err := attributevalue.UnmarshalMap(output.Item, &product); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "디코딩 실패"})
		return
	}

	c.JSON(http.StatusOK, product)
}
