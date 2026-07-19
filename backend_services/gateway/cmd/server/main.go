package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	scalar "github.com/MarceloPetrucio/go-scalar-api-reference"
	"github.com/redis/go-redis/v9"

	"github.com/eternal-orbit-labs/gateway/internal/config"
	"github.com/eternal-orbit-labs/gateway/internal/handler"
	"github.com/eternal-orbit-labs/gateway/internal/middleware"
	"github.com/eternal-orbit-labs/gateway/internal/platform"
	"github.com/eternal-orbit-labs/gateway/internal/repository"
	"github.com/eternal-orbit-labs/gateway/internal/service"

	_ "github.com/eternal-orbit-labs/gateway/docs"
)

// @title           EOL Gateway API
// @version         1.0
// @description     API gateway for Eternal Orbit Labs — authentication, organizations, teams, and app management.
//
// @host            localhost:8080
// @BasePath        /
// @schemes         http https
//
// @securityDefinitions.apikey CookieAuth
// @in                         cookie
// @name                       eol_access
// @description                JWT access token set automatically on login/signup.
//
// @securityDefinitions.apikey BearerAPIKey
// @in                         header
// @name                       Authorization
// @description                Personal API key — send as "Bearer eol_k1_..."

func main() {
	godotenv.Load()

	cfg := config.Load()
	ctx := context.Background()

	// Platform — choose between external (Docker) databases or embedded in-memory.
	var (
		pool *pgxpool.Pool
		rdb  *redis.Client
	)

	if cfg.IsLocalMemory() {
		log.Println("DATABASES_MEMORY=LOCAL → starting embedded postgres + redis")
		embedded, err := platform.StartEmbeddedDatabases(ctx)
		if err != nil {
			log.Fatalf("embedded databases: %v", err)
		}
		defer embedded.Stop()

		pool = embedded.Pool
		rdb = embedded.Redis
	} else {
		var err error
		pool, err = platform.NewPostgresPool(ctx, cfg.DatabaseURL)
		if err != nil {
			log.Fatalf("postgres: %v", err)
		}
		defer pool.Close()

		rdb, err = platform.NewRedisClient(ctx, cfg.RedisURL)
		if err != nil {
			log.Fatalf("redis: %v", err)
		}
		defer rdb.Close()
	}

	if err := platform.RunMigrations(ctx, pool, "migrations"); err != nil {
		log.Fatalf("migrations: %v", err)
	}

	jwt := platform.NewJWTManager(cfg.JWTSecret, cfg.JWTAccessTTL)

	// Repositories
	userRepo := repository.NewUserPG(pool)
	sessionRepo := repository.NewRefreshSessionPG(pool)
	sessionCache := repository.NewSessionRedisCache(rdb)
	apiKeyRepo := repository.NewPersonalAPIKeyPG(pool)
	rateLimiterRepo := repository.NewRateLimiterRedis(rdb)
	appRepo := repository.NewAppPG(pool)
	appOrgRepo := repository.NewAppOrgPG(pool)

	// Services
	authSvc := service.NewAuthService(userRepo, appOrgRepo, sessionRepo, sessionCache, rateLimiterRepo, jwt, cfg.RefreshTokenTTL)
	apiKeySvc := service.NewAPIKeyService(apiKeyRepo)
	appSvc := service.NewAppService(appRepo)
	appOrgSvc := service.NewAppOrgService(appRepo, appOrgRepo, userRepo)
	orgSidebarSvc := service.NewOrgSidebarService(appRepo, appOrgRepo)

	// Handlers
	authHandler := handler.NewAuthHandler(authSvc, cfg.JWTAccessTTL, cfg.RefreshTokenTTL, cfg.IsDev())
	oauthHandler := handler.NewOAuthHandler(
		authSvc,
		cfg.GoogleClientID,
		cfg.GoogleClientSecret,
		cfg.GoogleRedirectURL,
		cfg.CORSAllowedOrigins[0],
		cfg.JWTAccessTTL,
		cfg.RefreshTokenTTL,
		cfg.IsDev(),
	)
	apiKeyHandler := handler.NewAPIKeyHandler(apiKeySvc)
	appHandler := handler.NewAppHandler(appSvc)
	appOrgHandler := handler.NewAppOrgHandler(appOrgSvc, userRepo)
	orgSidebarHandler := handler.NewOrgSidebarHandler(orgSidebarSvc)

	// Router
	if !cfg.IsDev() {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.Default()
	r.Use(middleware.CORS(cfg.CORSAllowedOrigins))

	api := r.Group("/api")

	// Public auth routes
	auth := api.Group("/auth")
	auth.POST("/signup", authHandler.Signup)
	auth.POST("/login", authHandler.Login)
	auth.POST("/refresh", authHandler.Refresh)
	auth.POST("/logout", authHandler.Logout)
	auth.GET("/google", oauthHandler.GoogleRedirect)
	auth.GET("/google/callback", oauthHandler.GoogleCallback)

	// Protected routes
	protected := api.Group("")
	protected.Use(middleware.Auth(jwt, apiKeySvc, sessionRepo, sessionCache, cfg.JWTAccessTTL, cfg.IsDev()))
	protected.GET("/auth/me", authHandler.Me)
	protected.PATCH("/auth/me", authHandler.UpdateMe)
	protected.POST("/auth/change-password", authHandler.ChangePassword)
	protected.POST("/auth/apikeys", apiKeyHandler.Create)
	protected.GET("/auth/apikeys", apiKeyHandler.List)
	protected.DELETE("/auth/apikeys/:id", apiKeyHandler.Delete)
	protected.GET("/apps", appHandler.List)
	protected.GET("/apps/:slug", appHandler.Get)
	protected.GET("/apps/:slug/orgs", appOrgHandler.List)
	protected.POST("/apps/:slug/orgs", appOrgHandler.Create)
	protected.GET("/apps/:slug/orgs/:orgId/sidebar", orgSidebarHandler.Get)
	protected.GET("/apps/:slug/orgs/:orgId/members", appOrgHandler.ListMembers)
	protected.POST("/apps/:slug/orgs/:orgId/members", appOrgHandler.CreateMember)
	protected.PATCH("/apps/:slug/orgs/:orgId/members/:memberId", appOrgHandler.UpdateMember)
	protected.DELETE("/apps/:slug/orgs/:orgId/members/:memberId", appOrgHandler.RemoveMember)
	protected.GET("/apps/:slug/orgs/:orgId", appOrgHandler.Get)
	protected.POST("/apps/:slug/orgs/:orgId/invites", appOrgHandler.CreateInvite)
	protected.PATCH("/apps/:slug/orgs/:orgId/invites/:inviteId", appOrgHandler.UpdateInvite)
	protected.DELETE("/apps/:slug/orgs/:orgId/invites/:inviteId", appOrgHandler.DeleteInvite)
	protected.POST("/apps/:slug/invites/:inviteId/accept", appOrgHandler.AcceptInvite)
	protected.GET("/apps/:slug/orgs/:orgId/teams", appOrgHandler.ListTeams)
	protected.POST("/apps/:slug/orgs/:orgId/teams", appOrgHandler.CreateTeam)
	protected.GET("/apps/:slug/orgs/:orgId/teams/:teamId", appOrgHandler.GetTeam)
	protected.GET("/apps/:slug/orgs/:orgId/teams/:teamId/members", appOrgHandler.ListTeamMembers)
	protected.POST("/apps/:slug/orgs/:orgId/teams/:teamId/members", appOrgHandler.AddTeamMembers)
	protected.DELETE("/apps/:slug/orgs/:orgId/teams/:teamId/members/:userId", appOrgHandler.RemoveTeamMember)
	protected.DELETE("/apps/:slug/orgs/:orgId/teams/:teamId/members", appOrgHandler.BulkRemoveTeamMembers)

	// Health check
	//
	// @Summary      Health check
	// @Description  Returns the service health status.
	// @Tags         Health
	// @Produce      json
	// @Success      200  {object}  handler.HealthResponse
	// @Router       /api/health [get]
	api.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// API Docs — Scalar UI (no auth required)
	r.GET("/api/docs", func(c *gin.Context) {
		htmlContent, err := scalar.ApiReferenceHTML(&scalar.Options{
			SpecURL: "./docs/swagger.json",
			CustomOptions: scalar.CustomOptions{
				PageTitle: "EOL Gateway API",
			},
			DarkMode:   true,
			Theme:      scalar.ThemeKepler,
			HideModels: true,
		})
		if err != nil {
			c.String(http.StatusInternalServerError, "failed to generate docs: %v", err)
			return
		}
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(htmlContent))
	})
	r.StaticFile("/api/docs/swagger.json", "./docs/swagger.json")

	// Server
	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: r,
	}

	go func() {
		log.Printf("gateway starting on :%s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("shutting down...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("shutdown: %v", err)
	}
}
