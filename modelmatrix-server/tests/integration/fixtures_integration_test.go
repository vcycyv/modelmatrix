package integration

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// FixturesTestSuite exercises HTTP endpoints against data seeded via DB fixtures (builders).
// Direct inserts skip LDAP/API overhead for rows that are not under test; the API layer is still exercised on read paths.
type FixturesTestSuite struct {
	suite.Suite
	client    *http.Client
	baseURL   string
	authToken string
}

func (s *FixturesTestSuite) SetupSuite() {
	s.client = newAPIClient()
	s.baseURL = testServerURL
	s.authToken = authenticate(s.T(), s.client, s.baseURL, "michael.jordan", "111222333")
}

func (s *FixturesTestSuite) SetupTest() {
	truncateAllTables(s.T())
}

func (s *FixturesTestSuite) TearDownSuite() {
	truncateAllTables(s.T())
}

// TestListCollections_SeededViaDB verifies GET /api/collections sees rows inserted with CollectionBuilder.
func (s *FixturesTestSuite) TestListCollections_SeededViaDB() {
	NewCollectionBuilder(s.T()).WithName("Alpha").Build()
	NewCollectionBuilder(s.T()).WithName("Beta").Build()

	resp := makeRequest(s.T(), s.client, "GET", s.baseURL+"/api/collections?page=1&page_size=20", s.authToken, nil)
	defer resp.Body.Close()
	requireSuccess(s.T(), resp)

	var result struct {
		Code int `json:"code"`
		Data struct {
			Collections []struct {
				Name string `json:"name"`
			} `json:"collections"`
			Total int `json:"total"`
		} `json:"data"`
	}
	require.NoError(s.T(), json.NewDecoder(resp.Body).Decode(&result))

	require.Equal(s.T(), 200, result.Code)
	require.Equal(s.T(), 2, result.Data.Total, "list should reflect only rows seeded in this test (truncate per test)")
	names := make(map[string]bool)
	for _, c := range result.Data.Collections {
		names[c.Name] = true
	}
	assert.True(s.T(), names["Alpha"] && names["Beta"], "expected seeded collection names in list response")
}

// TestGetCollection_SeededViaDB verifies GET /api/collections/:id for a DB-seeded collection.
func (s *FixturesTestSuite) TestGetCollection_SeededViaDB() {
	c := NewCollectionBuilder(s.T()).WithName("Direct DB").WithDescription("from builder").Build()

	resp := makeRequest(s.T(), s.client, "GET", s.baseURL+"/api/collections/"+c.ID, s.authToken, nil)
	defer resp.Body.Close()
	requireSuccess(s.T(), resp)

	var result struct {
		Code int `json:"code"`
		Data struct {
			ID          string `json:"id"`
			Name        string `json:"name"`
			Description string `json:"description"`
		} `json:"data"`
	}
	require.NoError(s.T(), json.NewDecoder(resp.Body).Decode(&result))

	assert.Equal(s.T(), 200, result.Code)
	assert.Equal(s.T(), c.ID, result.Data.ID)
	assert.Equal(s.T(), "Direct DB", result.Data.Name)
	assert.Equal(s.T(), "from builder", result.Data.Description)
}

// TestGetDatasource_SeededViaDB verifies GET sees datasource + columns seeded via builders.
func (s *FixturesTestSuite) TestGetDatasource_SeededViaDB() {
	col := NewCollectionBuilder(s.T()).WithName("Col With DS").Build()
	ds := NewDatasourceBuilder(s.T(), col.ID).WithName("ds1").WithFilePath("minio/path/file.parquet").Build()
	NewColumnBuilder(s.T(), ds.ID).WithName("x1").Build()
	NewColumnBuilder(s.T(), ds.ID).WithName("x2").WithRole("target").Build()

	resp := makeRequest(s.T(), s.client, "GET", s.baseURL+"/api/datasources/"+ds.ID, s.authToken, nil)
	defer resp.Body.Close()
	requireSuccess(s.T(), resp)

	var result struct {
		Code int `json:"code"`
		Data struct {
			ID           string `json:"id"`
			Name         string `json:"name"`
			CollectionID string `json:"collection_id"`
			ColumnCount  int    `json:"column_count"`
		} `json:"data"`
	}
	require.NoError(s.T(), json.NewDecoder(resp.Body).Decode(&result))

	assert.Equal(s.T(), 200, result.Code)
	assert.Equal(s.T(), ds.ID, result.Data.ID)
	assert.Equal(s.T(), "ds1", result.Data.Name)
	assert.Equal(s.T(), col.ID, result.Data.CollectionID)
	assert.GreaterOrEqual(s.T(), result.Data.ColumnCount, 2)
}

func TestFixturesSuite(t *testing.T) {
	suite.Run(t, new(FixturesTestSuite))
}
