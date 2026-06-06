package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func NewRouter(db *gorm.DB) *gin.Engine {
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery(), cors())

	handlers := NewHandlers(db)

	router.GET("/healthz", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	v1 := router.Group("/api/v1")
	{
		v1.POST("/public/configs", handlers.GetPublicConfigs)
		v1.GET("/auth/bootstrap", handlers.GetBootstrapStatus)
		v1.POST("/auth/bootstrap/admin", handlers.InitializeAdmin)
		v1.POST("/auth/login", handlers.Login)
		v1.POST("/auth/logout", handlers.Logout)
		v1.GET("/auth/providers", handlers.ListAuthProviders)
		v1.POST("/auth/providers", handlers.CreateAuthProvider)
		v1.PUT("/auth/providers/:providerId", handlers.UpdateAuthProvider)
		v1.GET("/auth/admission-policy", handlers.GetAuthAdmissionPolicy)
		v1.PUT("/auth/admission-policy", handlers.UpdateAuthAdmissionPolicy)
		v1.GET("/auth/oidc/:providerId/start", handlers.StartOIDC)
		v1.GET("/auth/oidc/callback", handlers.CompleteOIDC)
		v1.GET("/users/me", handlers.GetCurrentUser)
		v1.PUT("/users/me", handlers.UpdateCurrentUser)
		v1.GET("/users/me/external-identities", handlers.ListMyExternalIdentities)
		v1.DELETE("/users/me/external-identities/:identityId", handlers.UnbindMyExternalIdentity)
		v1.GET("/users", handlers.ListUsers)
		v1.POST("/users", handlers.CreateUser)
		v1.PUT("/users/:userId", handlers.UpdateUser)
		v1.GET("/configs/definitions", handlers.ListConfigDefinitions)
		v1.PUT("/configs", handlers.UpdateConfigs)

		v1.GET("/projects", handlers.ListProjects)
		v1.POST("/projects", handlers.CreateProject)
		v1.GET("/projects/:projectId", handlers.GetProject)
		v1.PUT("/projects/:projectId", handlers.UpdateProject)
		v1.DELETE("/projects/:projectId", handlers.DeleteProject)

		v1.GET("/projects/:projectId/members", handlers.ListProjectMembers)
		v1.POST("/projects/:projectId/members", handlers.CreateProjectMember)
		v1.PUT("/projects/:projectId/members/:memberId", handlers.UpdateProjectMember)
		v1.DELETE("/projects/:projectId/members/:memberId", handlers.DeleteProjectMember)

		v1.GET("/projects/:projectId/applications", handlers.ListApplications)
		v1.POST("/projects/:projectId/applications", handlers.CreateApplication)
		v1.POST("/projects/:projectId/applications/parse-config", handlers.ParseApplicationConfig)
		v1.GET("/projects/:projectId/applications/:applicationId", handlers.GetApplication)
		v1.PUT("/projects/:projectId/applications/:applicationId", handlers.UpdateApplication)
		v1.DELETE("/projects/:projectId/applications/:applicationId", handlers.DeleteApplication)

		v1.GET("/access-tokens", handlers.ListAccessTokens)
		v1.POST("/access-tokens", handlers.CreateAccessToken)
		v1.DELETE("/access-tokens/:tokenId", handlers.RevokeAccessToken)
	}

	return router
}

func cors() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ctx.Writer.Header().Set("Access-Control-Allow-Origin", "http://localhost:5173")
		ctx.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		ctx.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, Accept-Language")
		ctx.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")

		if ctx.Request.Method == http.MethodOptions {
			ctx.AbortWithStatus(http.StatusNoContent)
			return
		}

		ctx.Next()
	}
}
