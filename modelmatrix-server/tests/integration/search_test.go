package integration

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

// SearchTestSuite tests GET /api/search.
type SearchTestSuite struct {
	suite.Suite
	client    *http.Client
	baseURL   string
	authToken string
}

func (s *SearchTestSuite) SetupSuite() {
	s.client = newAPIClient()
	s.baseURL = testServerURL
	s.authToken = authenticate(s.T(), s.client, s.baseURL, "michael.jordan", "111222333")
	s.seedSearchData()
}

func (s *SearchTestSuite) TearDownSuite() {
	truncateAllTables(s.T())
}

// seedSearchData creates folders, projects, and builds to make search non-empty.
func (s *SearchTestSuite) seedSearchData() {
	// Create a distinctly-named folder
	makeRequest(s.T(), s.client, "POST", s.baseURL+"/api/folders", s.authToken,
		map[string]string{"name": "SearchTargetFolder", "description": "for search tests"}).Body.Close()

	// Create a distinctly-named project
	makeRequest(s.T(), s.client, "POST", s.baseURL+"/api/projects", s.authToken,
		map[string]string{"name": "SearchTargetProject", "description": "for search tests"}).Body.Close()
}

// TestSearch_ByName verifies that a name query returns matching resources.
func (s *SearchTestSuite) TestSearch_ByName() {
	resp := makeRequest(s.T(), s.client, "GET", s.baseURL+"/api/search?q=SearchTarget&type=all", s.authToken, nil)
	defer resp.Body.Close()
	requireSuccess(s.T(), resp)

	var result struct {
		Data struct {
			Query   string        `json:"query"`
			Total   int           `json:"total"`
			Results []interface{} `json:"results"`
		} `json:"data"`
	}
	parseResponse(s.T(), resp, &result)
	assert.Equal(s.T(), "SearchTarget", result.Data.Query)
	assert.GreaterOrEqual(s.T(), result.Data.Total, 2, "should find at least folder and project")
}

// TestSearch_FilterByType verifies that type=folder returns only folders.
func (s *SearchTestSuite) TestSearch_FilterByType() {
	resp := makeRequest(s.T(), s.client, "GET", s.baseURL+"/api/search?q=SearchTarget&type=folder", s.authToken, nil)
	defer resp.Body.Close()
	requireSuccess(s.T(), resp)

	var result struct {
		Data struct {
			Total   int `json:"total"`
			Results []struct {
				Type string `json:"type"`
			} `json:"results"`
		} `json:"data"`
	}
	parseResponse(s.T(), resp, &result)
	assert.GreaterOrEqual(s.T(), result.Data.Total, 1)
	for _, r := range result.Data.Results {
		assert.Equal(s.T(), "folder", r.Type)
	}
}

// TestSearch_EmptyQuery returns results (or empty) without error.
func (s *SearchTestSuite) TestSearch_EmptyQuery() {
	resp := makeRequest(s.T(), s.client, "GET", s.baseURL+"/api/search?q=&type=all", s.authToken, nil)
	defer resp.Body.Close()
	// Empty query may return 200 with empty list or 400 — both are acceptable
	assert.NotEqual(s.T(), http.StatusInternalServerError, resp.StatusCode)
}

// TestSearch_NoResults verifies that a query with no matches returns empty results.
func (s *SearchTestSuite) TestSearch_NoResults() {
	resp := makeRequest(s.T(), s.client, "GET", s.baseURL+"/api/search?q=zzznomatchzzz&type=all", s.authToken, nil)
	defer resp.Body.Close()
	requireSuccess(s.T(), resp)

	var result struct {
		Data struct {
			Total   int           `json:"total"`
			Results []interface{} `json:"results"`
		} `json:"data"`
	}
	parseResponse(s.T(), resp, &result)
	assert.Equal(s.T(), 0, result.Data.Total)
	assert.Empty(s.T(), result.Data.Results)
}

// TestSearch_Unauthorized verifies that unauthenticated requests return 401.
func (s *SearchTestSuite) TestSearch_Unauthorized() {
	resp := makeRequest(s.T(), s.client, "GET", s.baseURL+"/api/search?q=test", "", nil)
	defer resp.Body.Close()
	requireUnauthorized(s.T(), resp)
}

func TestSearchSuite(t *testing.T) {
	suite.Run(t, new(SearchTestSuite))
}
