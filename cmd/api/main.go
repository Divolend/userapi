package main

import (
	"log"
	_ "rest-api-gin/docs"
	"rest-api-gin/internal/database"
	"rest-api-gin/internal/handler"
	"rest-api-gin/internal/repository"
	"rest-api-gin/internal/service"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// @title User Management API
// @version 1.0
// @description REST API для управления пользователями
// @host localhost:8080
// @BasePath /
func main() {
	db, err := database.NewConnection()
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	log.Println("Successfully connected to database!")
	repo := repository.NewPostgresUserRepository(db)
	handler.UserService = service.NewUserService(repo)
	router := gin.Default()
	router.Use(handler.LoggerMiddleware)

	router.GET("/health", handler.HealthHandler)
	router.GET("/users/:id", handler.IdHandler)
	router.POST("/users", handler.AuthMiddleware, handler.CreateUserHandler)
	router.PATCH("/users/:id", handler.AuthMiddleware, handler.UpdateHandler)
	router.DELETE("/users/:id", handler.AuthMiddleware, handler.DeleteHandler)
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	router.Run(":8080")

}
