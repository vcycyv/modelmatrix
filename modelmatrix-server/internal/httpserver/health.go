package httpserver

import (
	"modelmatrix-server/internal/infrastructure/db"
	"modelmatrix-server/internal/infrastructure/fileservice"
	infraldap "modelmatrix-server/internal/infrastructure/ldap"
	"modelmatrix-server/pkg/response"

	"github.com/gin-gonic/gin"
)

// HealthHandler returns the same handler used for production GET /api/health:
// PostgreSQL connectivity, LDAP, and object storage (MinIO/S3-compatible).
func HealthHandler(ldapClient infraldap.Client, fileService fileservice.FileService) gin.HandlerFunc {
	return func(c *gin.Context) {
		health := struct {
			Status   string `json:"status"`
			Database string `json:"database"`
			LDAP     string `json:"ldap"`
			MinIO    string `json:"minio"`
		}{
			Status:   "healthy",
			Database: "healthy",
			LDAP:     "healthy",
			MinIO:    "healthy",
		}

		if err := db.HealthCheck(); err != nil {
			health.Status = "unhealthy"
			health.Database = "unhealthy: " + err.Error()
		}

		if err := ldapClient.HealthCheck(); err != nil {
			health.Status = "unhealthy"
			health.LDAP = "unhealthy: " + err.Error()
		}

		if err := fileService.HealthCheck(); err != nil {
			health.Status = "unhealthy"
			health.MinIO = "unhealthy: " + err.Error()
		}

		if health.Status == "healthy" {
			response.Success(c, health)
		} else {
			response.ServiceUnavailable(c, health.Status)
		}
	}
}
