package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/eternal-orbit-labs/gateway/internal/handler"
	"github.com/eternal-orbit-labs/gateway/internal/middleware"
	"github.com/eternal-orbit-labs/gateway/internal/platform"
	"github.com/eternal-orbit-labs/gateway/internal/service"
	"github.com/eternal-orbit-labs/gateway/internal/testutil"
)

const testJWTSecret = "test-secret-key-for-testing-only"

type testEnv struct {
	router      *gin.Engine
	jwt         *platform.JWTManager
	authSvc     *service.AuthService
	apiKeySvc   *service.APIKeyService
	users       *testutil.FakeUserRepo
	sessions    *testutil.FakeRefreshSessionRepo
	cache       *testutil.FakeSessionCache
	rateLimiter *testutil.FakeRateLimiter
	apiKeyRepo  *testutil.FakePersonalAPIKeyRepo
	appRepo     *testutil.FakeAppRepo
}

func setupTestEnv() *testEnv {
	gin.SetMode(gin.TestMode)

	users := testutil.NewFakeUserRepo()
	sessions := testutil.NewFakeRefreshSessionRepo()
	cache := testutil.NewFakeSessionCache()
	rateLimiter := testutil.NewFakeRateLimiter()
	orgs := testutil.NewFakeAppOrgRepo()
	apiKeyRepo := testutil.NewFakePersonalAPIKeyRepo()
	appRepo := testutil.NewFakeAppRepo()
	jwtMgr := platform.NewJWTManager(testJWTSecret, 15*time.Minute)

	authSvc := service.NewAuthService(users, orgs, sessions, cache, rateLimiter, jwtMgr, 24*time.Hour)
	apiKeySvc := service.NewAPIKeyService(apiKeyRepo)
	appSvc := service.NewAppService(appRepo)
	appOrgSvc := service.NewAppOrgService(appRepo, orgs, users)
	orgSidebarSvc := service.NewOrgSidebarService(appRepo, orgs)

	authHandler := handler.NewAuthHandler(authSvc, 15*time.Minute, 24*time.Hour, true)
	oauthHandler := handler.NewOAuthHandler(authSvc, "", "", "", "http://localhost:3000", 15*time.Minute, 24*time.Hour, true)
	apiKeyHandler := handler.NewAPIKeyHandler(apiKeySvc)
	appHandler := handler.NewAppHandler(appSvc)
	appOrgHandler := handler.NewAppOrgHandler(appOrgSvc, users)
	orgSidebarHandler := handler.NewOrgSidebarHandler(orgSidebarSvc)

	r := gin.New()
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
	protected.Use(middleware.Auth(jwtMgr, apiKeySvc, sessions, cache, 15*time.Minute, true))
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
	api.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	return &testEnv{
		router:      r,
		jwt:         jwtMgr,
		authSvc:     authSvc,
		apiKeySvc:   apiKeySvc,
		users:       users,
		sessions:    sessions,
		cache:       cache,
		rateLimiter: rateLimiter,
		apiKeyRepo:  apiKeyRepo,
		appRepo:     appRepo,
	}
}

// jsonRequest creates an HTTP request with JSON body.
func jsonRequest(method, path string, body interface{}) *http.Request {
	var buf bytes.Buffer
	if body != nil {
		json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	return req
}

// authedRequest creates an authenticated request with a valid JWT cookie.
func authedRequest(env *testEnv, method, path string, body interface{}, userID, email string) *http.Request {
	req := jsonRequest(method, path, body)
	token, _ := env.jwt.GenerateAccessToken(userID, email)
	req.AddCookie(&http.Cookie{Name: "eol_access", Value: token})
	return req
}

// doRequest executes a request and returns the recorder.
func doRequest(env *testEnv, req *http.Request) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	env.router.ServeHTTP(w, req)
	return w
}

// parseJSON unmarshals the response body into a map.
func parseJSON(t *testing.T, w *httptest.ResponseRecorder) map[string]interface{} {
	t.Helper()
	var result map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse JSON response: %v\nbody: %s", err, w.Body.String())
	}
	return result
}
