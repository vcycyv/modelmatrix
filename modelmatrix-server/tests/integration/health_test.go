package integration

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHealthEndpoint verifies GET /api/health uses the same checks as production (DB, LDAP, MinIO).
func TestHealthEndpoint(t *testing.T) {
	client := newAPIClient()
	resp, err := client.Get(testServerURL + "/api/health")
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode, "health must be 200 when all dependencies are up")

	var envelope struct {
		Code int `json:"code"`
		Data struct {
			Status   string `json:"status"`
			Database string `json:"database"`
			LDAP     string `json:"ldap"`
			MinIO    string `json:"minio"`
		} `json:"data"`
	}
	parseResponse(t, resp, &envelope)
	require.Equal(t, 200, envelope.Code)
	assert.Equal(t, "healthy", envelope.Data.Status)
	assert.Equal(t, "healthy", envelope.Data.Database)
	assert.Equal(t, "healthy", envelope.Data.LDAP)
	assert.Equal(t, "healthy", envelope.Data.MinIO)
}
