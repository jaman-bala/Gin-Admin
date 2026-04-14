package http

import (
	"gin_auth_service/config"
	"gin_auth_service/internal/application/analytics"
	"gin_auth_service/internal/application/auditlog"
	"gin_auth_service/internal/application/auth"
	"gin_auth_service/internal/application/file"
	"gin_auth_service/internal/application/token"
	"gin_auth_service/internal/application/user"
	"gin_auth_service/internal/infrastructure/http/handler"
	"gin_auth_service/internal/infrastructure/http/middleware"
	"gin_auth_service/internal/infrastructure/minio"
	"gin_auth_service/internal/infrastructure/postgres"
	"gin_auth_service/internal/infrastructure/redis"
	"gin_auth_service/internal/pkg/jwt"
	"net/http"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// SetupRoutes initializes the Gin engine and defines all API routes.
func SetupRoutes(db *sqlx.DB, cfg *config.Config) *gin.Engine {
	server := gin.Default()

	server.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	server.GET("/health", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	server.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173", "http://localhost:3000", "http://localhost:8085"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		AllowCredentials: true,
	}))

	// Input sanitization middleware for XSS/SQLi protection
	server.Use(middleware.SanitizeMiddleware())

	// Infrastructure (Adapters)
	userRepo := postgres.NewUserRepository(db)
	auditRepo := postgres.NewAuditRepository(db)
	analyticsRepo := postgres.NewAnalyticsRepository(db)
	tokenRepo, _ := redis.NewCache(cfg)
	storage, err := minio.NewStorage(
		cfg.Minio.MinioHost+":"+cfg.Minio.MinioPort,
		cfg.Minio.MinioAccessKey,
		cfg.Minio.MinioSecretKey,
		cfg.Minio.MinioSSL,
		cfg.Minio.MinioPublicHost,
	)
	if err != nil {
		panic("failed to connect to MinIO: " + err.Error())
	}

	// Utils
	jwtService := jwt.NewJWTService(cfg.JWT.Secret)

	// Application Services
	fileSvc := file.NewUseCase(storage, cfg.Minio.MinioBucket)
	tokenSvc := token.NewUseCase(tokenRepo, jwtService)
	auditSvc := auditlog.NewUseCase(auditRepo)
	userSvc := user.NewUseCase(userRepo, fileSvc)
	authSvc := auth.NewUseCase(userRepo, tokenSvc, fileSvc, cfg)
	analyticsSvc := analytics.NewUseCase(analyticsRepo)

	// Middleware
	authMid := middleware.AuthMiddleware(authSvc)
	auditMid := middleware.AuditMiddleware(auditSvc)
	blacklistMid := middleware.TokenBlacklistMiddleware(tokenSvc)
	adminMid := middleware.RequireRoleLevelMiddleware(3)
	loginRateLimitMid := middleware.LoginRateLimitMiddleware(tokenRepo)

	// Handlers
	authHdl := handler.NewAuthHandler(authSvc)
	userHdl := handler.NewUserHandler(userSvc)
	auditHdl := handler.NewAuditHandler(auditSvc)
	analyticsHdl := handler.NewAnalyticsHandler(analyticsSvc)

	api := server.Group("/api/v1")
	{
		authGroup := api.Group("/auth")
		authGroup.Use(auditMid)
		{
			authGroup.POST("/login", loginRateLimitMid, authHdl.Login)
			authGroup.POST("/logout", authHdl.Logout)
			authGroup.POST("/refresh", authHdl.Refresh)
			authGroup.GET("/me", blacklistMid, authMid, authHdl.UserMe)
		}

		// `/users/me` needs to avoid auth group if we want it under users logically, but we mapped it to authHdl.UserMe.
		// Let's create a dedicated group for `/users`
		users := api.Group("/users")
		users.Use(blacklistMid, authMid, auditMid)
		{
			// /users/me specific routes (must go before /:id)
			users.GET("/me", authHdl.UserMe)
			users.PUT("/me", userHdl.UpdateMe)
			users.PATCH("/me", userHdl.UpdateMe)
			// users.PUT("/me", userHdl.UpdateMe) // if implemented, else skip or map to Patch

			// admin only routes
			adminUsers := users.Group("")
			adminUsers.Use(adminMid)
			{
				adminUsers.POST("", userHdl.Create)
				adminUsers.GET("", userHdl.GetAll)
				adminUsers.GET("/stats", analyticsHdl.GetUserStats)
				adminUsers.GET("/:id", userHdl.GetByID)
				adminUsers.GET("/phone/:phone", userHdl.GetByPhone)
				adminUsers.PATCH("/:id", userHdl.Patch) // The handler is currently Patch, frontend openapi might use put or patch
				adminUsers.PUT("/:id", userHdl.Patch)   // Alias for frontend compatibility
				adminUsers.DELETE("/:id", userHdl.Delete)
			}
		}

		auditGroup := api.Group("/audit")
		auditGroup.Use(blacklistMid, authMid, adminMid)
		{
			auditGroup.GET("", auditHdl.GetAllLogs)
		}
	}

	return server
}
