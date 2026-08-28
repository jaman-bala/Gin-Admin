package http

import (
	"context"

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
	netpprof "net/http/pprof"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	scalar "github.com/MarceloPetrucio/go-scalar-api-reference"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/swaggo/swag"
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
	adminMid          gin.HandlerFunc
	loginRateLimitMid gin.HandlerFunc

	auditRecorder *auditlog.AsyncRecorder
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
	auditRecorder := auditlog.NewAsyncRecorder(auditRepo, 4096, 256, time.Second)
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
		auditMid:          middleware.AuditMiddleware(auditRecorder),
		adminMid:          middleware.RequireRoleLevelMiddleware(middleware.RoleLevelAdmin),
		loginRateLimitMid: middleware.LoginRateLimitMiddleware(redisCache),

		auditRecorder: auditRecorder,
	}
}

// SetupRoutes initializes the Gin engine, wires all dependencies, and registers
// routes. The returned shutdown function flushes background workers (audit
// recorder) and must be called after the HTTP server has stopped.
func SetupRoutes(db *sqlx.DB, cfg *config.Config) (*gin.Engine, func(context.Context) error) {
	server := gin.Default()

	// No reverse proxy in front of this service by default, so no X-Forwarded-*
	// header should be trusted — otherwise c.ClientIP() (used by the login
	// rate limiter and audit logging) can be spoofed by any client, trivially
	// defeating brute-force protection. Set TRUSTED_PROXIES to the real
	// proxy's IP/CIDR when one is actually in front of this service.
	if err := server.SetTrustedProxies(cfg.Server.TrustedProxies); err != nil {
		panic("invalid TRUSTED_PROXIES: " + err.Error())
	}

	server.Use(middleware.RequestIDMiddleware())
	server.Use(middleware.MetricsMiddleware())
	server.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.Server.CORSOrigins,
		AllowMethods:     []string{"GET", "POST", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", middleware.RequestIDHeader},
		AllowCredentials: true,
	}))
	server.Use(middleware.BodySizeLimit(cfg.Server.MaxBodyBytes))

	// Registered after the global middleware above so /docs also goes through
	// RequestID/Metrics/CORS/BodySizeLimit — Gin snapshots a route's middleware
	// chain at registration time, so this must come after the server.Use calls.
	server.GET("/docs", func(c *gin.Context) {
		specJSON, err := swag.ReadDoc()
		if err != nil {
			c.String(500, "failed to read swagger spec: %s", err.Error())
			return
		}
		html, err := scalar.ApiReferenceHTML(&scalar.Options{
			SpecContent: specJSON,
			DarkMode:    true,
			Theme:       scalar.ThemeDeepSpace,
			PageTitle:   "AUTH SERVICE API",
		})
		if err != nil {
			c.String(500, err.Error())
			return
		}
		c.Header("Content-Type", "text/html")
		c.String(200, html)
	})

	server.GET("/metrics", middleware.MetricsAuth(cfg.Server.MetricsToken), gin.WrapH(promhttp.Handler()))

	deps := buildDeps(db, cfg)
	server.GET("/health", deps.healthHdl.Check)
	registerRoutes(server, deps)

	return server, deps.auditRecorder.Close
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
		authGroup.GET("/me", deps.authMid, deps.userHdl.GetMe)
	}

	users := api.Group("/users")
	users.Use(deps.authMid, deps.auditMid)
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
	auditGroup.Use(deps.authMid, deps.adminMid, deps.auditMid)
	{
		auditGroup.GET("", deps.auditHdl.GetAllLogs)
	}

	// Runtime profiling (net/http/pprof), admin-only. Includes the
	// goroutineleak profile introduced in Go 1.27 for detecting
	// permanently blocked goroutines.
	debug := server.Group("/debug/pprof")
	debug.Use(deps.authMid, deps.adminMid)
	{
		debug.GET("/", gin.WrapF(netpprof.Index))
		debug.GET("/cmdline", gin.WrapF(netpprof.Cmdline))
		debug.GET("/profile", gin.WrapF(netpprof.Profile))
		debug.GET("/symbol", gin.WrapF(netpprof.Symbol))
		debug.POST("/symbol", gin.WrapF(netpprof.Symbol))
		debug.GET("/trace", gin.WrapF(netpprof.Trace))
		for _, p := range []string{"allocs", "block", "goroutine", "goroutineleak", "heap", "mutex", "threadcreate"} {
			debug.GET("/"+p, gin.WrapH(netpprof.Handler(p)))
		}
	}
}
