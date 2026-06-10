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
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"gin_auth_service/internal/infrastructure/postgres"
	"gin_auth_service/internal/infrastructure/redis"
	"gin_auth_service/internal/pkg/jwt"
	"strings"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// routeDeps holds initialized handlers and middleware needed for route registration.
type routeDeps struct {
	healthHdl    *handler.HealthHandler
	authHdl      *handler.AuthHandler
	userHdl      *handler.UserHandler
	auditHdl     *handler.AuditHandler
	analyticsHdl *handler.AnalyticsHandler

	authMid           gin.HandlerFunc
	auditMid          gin.HandlerFunc
	blacklistMid      gin.HandlerFunc
	adminMid          gin.HandlerFunc
	loginRateLimitMid gin.HandlerFunc
}

// buildDeps initializes all infrastructure, application services, and handlers.
// Panics on fatal infrastructure errors (Redis, MinIO) — fail fast rather than
// serving requests against a broken dependency graph.
func buildDeps(db *sqlx.DB, cfg *config.Config) *routeDeps {
	// Infrastructure
	userRepo := postgres.NewUserRepository(db)
	auditRepo := postgres.NewAuditRepository(db)
	analyticsRepo := postgres.NewAnalyticsRepository(db)

	redisCache, err := redis.NewCache(cfg)
	if err != nil {
		panic("failed to connect to Redis: " + err.Error())
	}

	minioEndpoint := cfg.Minio.MinioHost
	if !strings.Contains(minioEndpoint, ":") {
		minioEndpoint += ":" + cfg.Minio.MinioPort
	}
	storage, err := minio.NewStorage(
		minioEndpoint,
		cfg.Minio.MinioAccessKey,
		cfg.Minio.MinioSecretKey,
		cfg.Minio.MinioSSL,
		cfg.Minio.MinioPublicHost,
	)
	if err != nil {
		panic("failed to connect to MinIO: " + err.Error())
	}

	// Application services
	jwtService := jwt.NewJWTService(cfg.JWT.Secret)
	fileSvc := file.NewUseCase(storage, cfg.Minio.MinioBucket)
	tokenSvc := token.NewUseCase(redisCache, jwtService)
	auditSvc := auditlog.NewUseCase(auditRepo)
	analyticsSvc := analytics.NewUseCase(analyticsRepo)
	userSvc := user.NewUseCase(userRepo, fileSvc)
	authSvc := auth.NewUseCase(userRepo, tokenSvc, userSvc, cfg.JWT.Expiry, cfg.JWT.RefreshExpiry)

	return &routeDeps{
		healthHdl:    handler.NewHealthHandler(db, redisCache, storage),
		authHdl:      handler.NewAuthHandler(authSvc),
		userHdl:      handler.NewUserHandler(userSvc),
		auditHdl:     handler.NewAuditHandler(auditSvc),
		analyticsHdl: handler.NewAnalyticsHandler(analyticsSvc),

		authMid:           middleware.AuthMiddleware(authSvc),
		auditMid:          middleware.AuditMiddleware(auditSvc),
		blacklistMid:      middleware.TokenBlacklistMiddleware(tokenSvc),
		adminMid:          middleware.RequireRoleLevelMiddleware(middleware.RoleLevelAdmin),
		loginRateLimitMid: middleware.LoginRateLimitMiddleware(redisCache),
	}
}

// SetupRoutes initializes the Gin engine, wires all dependencies, and registers routes.
func SetupRoutes(db *sqlx.DB, cfg *config.Config) *gin.Engine {
	server := gin.Default()

	server.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	server.Use(middleware.RequestIDMiddleware())
	server.Use(middleware.MetricsMiddleware())
	server.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173", "http://localhost:3000", "http://localhost:8085"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", middleware.RequestIDHeader},
		AllowCredentials: true,
	}))
	server.Use(middleware.SanitizeMiddleware())

	server.GET("/metrics", gin.WrapH(promhttp.Handler()))

	deps := buildDeps(db, cfg)
	server.GET("/health", deps.healthHdl.Check)
	registerRoutes(server, deps)

	return server
}

// registerRoutes declares all API routes. It has no infrastructure concerns —
// only Gin group/route wiring using pre-built handlers and middleware.
func registerRoutes(server *gin.Engine, deps *routeDeps) {
	api := server.Group("/api/v1")

	authGroup := api.Group("/auth")
	authGroup.Use(deps.auditMid)
	{
		authGroup.POST("/login", deps.loginRateLimitMid, deps.authHdl.Login)
		authGroup.POST("/logout", deps.authHdl.Logout)
		authGroup.POST("/refresh", deps.authHdl.Refresh)
		authGroup.GET("/me", deps.blacklistMid, deps.authMid, deps.userHdl.GetMe)
	}

	users := api.Group("/users")
	users.Use(deps.blacklistMid, deps.authMid, deps.auditMid)
	{
		users.GET("/me", deps.userHdl.GetMe)
		users.PUT("/me", deps.userHdl.UpdateMe)
		users.PATCH("/me", deps.userHdl.UpdateMe)

		adminUsers := users.Group("")
		adminUsers.Use(deps.adminMid)
		{
			adminUsers.POST("", deps.userHdl.Create)
			adminUsers.GET("", deps.userHdl.GetAll)
			adminUsers.GET("/stats", deps.analyticsHdl.GetUserStats)
			adminUsers.GET("/:id", deps.userHdl.GetByID)
			adminUsers.GET("/phone/:phone", deps.userHdl.GetByPhone)
			adminUsers.PATCH("/:id", deps.userHdl.Patch)
			adminUsers.PUT("/:id", deps.userHdl.Patch)
			adminUsers.DELETE("/:id", deps.userHdl.Delete)
		}
	}

	auditGroup := api.Group("/audit")
	auditGroup.Use(deps.blacklistMid, deps.authMid, deps.adminMid, deps.auditMid)
	{
		auditGroup.GET("", deps.auditHdl.GetAllLogs)
	}
}