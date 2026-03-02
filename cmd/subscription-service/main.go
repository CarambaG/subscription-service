package main

import (
	_ "subscription-service/docs"
	"subscription-service/internal/config"
	"subscription-service/internal/handler"
	"subscription-service/internal/logger"
	db "subscription-service/internal/postgres"
	"subscription-service/internal/repository"
	"subscription-service/internal/service"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// @title           Subscription API
// @version         1.0
// @description     Это сервис подписок на Go.
// @host            localhost:8080
// @BasePath        /

func main() {
	cfg, err := config.Load(".env")
	if err != nil {
		logger.Warn("config load: %v", err)
	}

	logger.Init()

	pool, err := db.ConnectWithRetry(cfg)
	if err != nil {
		logger.Error("postgres connect: %v", err)
		return
	}
	defer pool.Close()

	logger.Info("Connected to database: %s\n", cfg.PostgresDSN())

	repo := repository.NewSubscriptionRepo(pool)
	svc := service.NewSubscriptionService(repo)
	h := handler.NewSubscriptionHandler(svc)

	r := gin.Default()
	h.Register(r)

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	logger.Info("server started on %s:", cfg.HTTPPort)
	r.Run(":" + cfg.HTTPPort)
}
