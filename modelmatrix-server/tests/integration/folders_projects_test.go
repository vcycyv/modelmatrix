package integration

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// FoldersProjectsTestSuite tests /api/folders and /api/projects endpoints.
type FoldersProjectsTestSuite struct {
	suite.Suite
	client    *http.Client
	baseURL   string
	authToken string
}

func (s *FoldersProjectsTestSuite) SetupSuite() {
	s.client = newAPIClient()
	s.baseURL = testServerURL
	s.authToken = authenticate(s.T(), s.client, s.baseURL, "michael.jordan", "111222333")
}

func (s *FoldersProjectsTestSuite) TearDownSuite() {
	truncateAllTables(s.T())
}

// --- Folder tests ---

func (s *FoldersProjectsTestSuite) TestCreateFolder() {
	resp := makeRequest(s.T(), s.client, "POST", s.baseURL+"/api/folders", s.authToken,
		map[string]string{"name": "Root Folder A", "description": "top-level folder"})
	defer resp.Body.Close()
	requireCreated(s.T(), resp)

	var result struct {
		Data struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"data"`
	}
	parseResponse(s.T(), resp, &result)
	assert.NotEmpty(s.T(), result.Data.ID)
	assert.Equal(s.T(), "Root Folder A", result.Data.Name)
}

func (s *FoldersProjectsTestSuite) TestCreateFolder_MissingName() {
	resp := makeRequest(s.T(), s.client, "POST", s.baseURL+"/api/folders", s.authToken,
		map[string]string{"description": "no name"})
	defer resp.Body.Close()
	requireBadRequest(s.T(), resp)
}

func (s *FoldersProjectsTestSuite) TestCreateSubfolder() {
	parentID := s.createFolder("Parent Folder")

	resp := makeRequest(s.T(), s.client, "POST", s.baseURL+"/api/folders", s.authToken,
		map[string]interface{}{"name": "Child Folder", "parent_id": parentID})
	defer resp.Body.Close()
	requireCreated(s.T(), resp)

	var result struct {
		Data struct {
			ID       string  `json:"id"`
			ParentID *string `json:"parent_id"`
		} `json:"data"`
	}
	parseResponse(s.T(), resp, &result)
	require.NotNil(s.T(), result.Data.ParentID)
	assert.Equal(s.T(), parentID, *result.Data.ParentID)
}

func (s *FoldersProjectsTestSuite) TestGetFolderChildren() {
	parentID := s.createFolder("Parent For Children")
	s.createSubfolder(parentID, "Child 1")
	s.createSubfolder(parentID, "Child 2")

	resp := makeRequest(s.T(), s.client, "GET", s.baseURL+"/api/folders/"+parentID+"/children", s.authToken, nil)
	defer resp.Body.Close()
	requireSuccess(s.T(), resp)

	var result struct {
		Data []struct{ ID string `json:"id"` } `json:"data"`
	}
	parseResponse(s.T(), resp, &result)
	assert.GreaterOrEqual(s.T(), len(result.Data), 2)
}

func (s *FoldersProjectsTestSuite) TestUpdateFolder() {
	folderID := s.createFolder("Folder To Update")

	resp := makeRequest(s.T(), s.client, "PUT", s.baseURL+"/api/folders/"+folderID, s.authToken,
		map[string]string{"name": "Updated Folder Name"})
	defer resp.Body.Close()
	requireSuccess(s.T(), resp)

	var result struct {
		Data struct{ Name string `json:"name"` } `json:"data"`
	}
	parseResponse(s.T(), resp, &result)
	assert.Equal(s.T(), "Updated Folder Name", result.Data.Name)
}

func (s *FoldersProjectsTestSuite) TestDeleteFolder() {
	folderID := s.createFolder("Folder To Delete")

	resp := makeRequest(s.T(), s.client, "DELETE", s.baseURL+"/api/folders/"+folderID, s.authToken, nil)
	defer resp.Body.Close()
	requireNoContent(s.T(), resp)

	getResp := makeRequest(s.T(), s.client, "GET", s.baseURL+"/api/folders/"+folderID, s.authToken, nil)
	defer getResp.Body.Close()
	requireNotFound(s.T(), getResp)
}

func (s *FoldersProjectsTestSuite) TestGetFolderContentsCount() {
	folderID := s.createFolder("Folder With Count")

	resp := makeRequest(s.T(), s.client, "GET", s.baseURL+"/api/folders/"+folderID+"/contents-count", s.authToken, nil)
	defer resp.Body.Close()
	requireSuccess(s.T(), resp)

	var result struct {
		Data struct {
			SubfolderCount int `json:"subfolder_count"`
			ProjectCount   int `json:"project_count"`
		} `json:"data"`
	}
	parseResponse(s.T(), resp, &result)
	assert.Equal(s.T(), 0, result.Data.SubfolderCount)
	assert.Equal(s.T(), 0, result.Data.ProjectCount)
}

func (s *FoldersProjectsTestSuite) TestGetRootFolders() {
	s.createFolder("Root List Folder")

	resp := makeRequest(s.T(), s.client, "GET", s.baseURL+"/api/folders", s.authToken, nil)
	defer resp.Body.Close()
	requireSuccess(s.T(), resp)

	var result struct {
		Data []interface{} `json:"data"`
	}
	parseResponse(s.T(), resp, &result)
	assert.GreaterOrEqual(s.T(), len(result.Data), 1)
}

// TestGetFolder_OK verifies GET /api/folders/:id returns 200 with expected fields (not only 404-after-delete).
func (s *FoldersProjectsTestSuite) TestGetFolder_OK() {
	folderID := s.createFolder("Folder For GET")

	resp := makeRequest(s.T(), s.client, "GET", s.baseURL+"/api/folders/"+folderID, s.authToken, nil)
	defer resp.Body.Close()
	requireSuccess(s.T(), resp)

	var result struct {
		Data struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"data"`
	}
	parseResponse(s.T(), resp, &result)
	assert.Equal(s.T(), folderID, result.Data.ID)
	assert.Equal(s.T(), "Folder For GET", result.Data.Name)
}

// --- Project tests ---

func (s *FoldersProjectsTestSuite) TestCreateProject() {
	resp := makeRequest(s.T(), s.client, "POST", s.baseURL+"/api/projects", s.authToken,
		map[string]string{"name": "Root Project A", "description": "top-level project"})
	defer resp.Body.Close()
	requireCreated(s.T(), resp)

	var result struct {
		Data struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"data"`
	}
	parseResponse(s.T(), resp, &result)
	assert.NotEmpty(s.T(), result.Data.ID)
	assert.Equal(s.T(), "Root Project A", result.Data.Name)
}

// TestListAndGetProject verifies GET /api/projects and GET /api/projects/:id.
func (s *FoldersProjectsTestSuite) TestListAndGetProject() {
	projectID := s.createProject("Listed Project")

	listResp := makeRequest(s.T(), s.client, "GET", s.baseURL+"/api/projects", s.authToken, nil)
	defer listResp.Body.Close()
	requireSuccess(s.T(), listResp)
	var listResult struct {
		Data []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"data"`
	}
	parseResponse(s.T(), listResp, &listResult)
	require.NotEmpty(s.T(), listResult.Data)
	found := false
	for _, p := range listResult.Data {
		if p.ID == projectID {
			found = true
			assert.Equal(s.T(), "Listed Project", p.Name)
			break
		}
	}
	require.True(s.T(), found, "created root project should appear in GET /api/projects")

	getResp := makeRequest(s.T(), s.client, "GET", s.baseURL+"/api/projects/"+projectID, s.authToken, nil)
	defer getResp.Body.Close()
	requireSuccess(s.T(), getResp)
	var getResult struct {
		Data struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"data"`
	}
	parseResponse(s.T(), getResp, &getResult)
	assert.Equal(s.T(), projectID, getResult.Data.ID)
	assert.Equal(s.T(), "Listed Project", getResult.Data.Name)
}

func (s *FoldersProjectsTestSuite) TestCreateProjectInFolder() {
	folderID := s.createFolder("Folder For Project")

	resp := makeRequest(s.T(), s.client, "POST", s.baseURL+"/api/projects", s.authToken,
		map[string]interface{}{"name": "Project In Folder", "folder_id": folderID})
	defer resp.Body.Close()
	requireCreated(s.T(), resp)

	var result struct {
		Data struct {
			FolderID *string `json:"folder_id"`
		} `json:"data"`
	}
	parseResponse(s.T(), resp, &result)
	require.NotNil(s.T(), result.Data.FolderID)
	assert.Equal(s.T(), folderID, *result.Data.FolderID)
}

func (s *FoldersProjectsTestSuite) TestGetProjectsInFolder() {
	folderID := s.createFolder("Folder For Projects List")
	s.createProjectInFolder(folderID, "P1")
	s.createProjectInFolder(folderID, "P2")

	resp := makeRequest(s.T(), s.client, "GET", s.baseURL+"/api/folders/"+folderID+"/projects", s.authToken, nil)
	defer resp.Body.Close()
	requireSuccess(s.T(), resp)

	var result struct {
		Data []interface{} `json:"data"`
	}
	parseResponse(s.T(), resp, &result)
	assert.GreaterOrEqual(s.T(), len(result.Data), 2)
}

func (s *FoldersProjectsTestSuite) TestUpdateProject() {
	projectID := s.createProject("Project To Update")

	resp := makeRequest(s.T(), s.client, "PUT", s.baseURL+"/api/projects/"+projectID, s.authToken,
		map[string]string{"name": "Updated Project Name"})
	defer resp.Body.Close()
	requireSuccess(s.T(), resp)

	var result struct {
		Data struct{ Name string `json:"name"` } `json:"data"`
	}
	parseResponse(s.T(), resp, &result)
	assert.Equal(s.T(), "Updated Project Name", result.Data.Name)
}

func (s *FoldersProjectsTestSuite) TestDeleteProject() {
	projectID := s.createProject("Project To Delete")

	resp := makeRequest(s.T(), s.client, "DELETE", s.baseURL+"/api/projects/"+projectID, s.authToken, nil)
	defer resp.Body.Close()
	requireNoContent(s.T(), resp)
}

func (s *FoldersProjectsTestSuite) TestFolderUnauthorized() {
	resp := makeRequest(s.T(), s.client, "GET", s.baseURL+"/api/folders", "", nil)
	defer resp.Body.Close()
	requireUnauthorized(s.T(), resp)
}

// --- helpers ---

func (s *FoldersProjectsTestSuite) createFolder(name string) string {
	resp := makeRequest(s.T(), s.client, "POST", s.baseURL+"/api/folders", s.authToken,
		map[string]string{"name": name})
	defer resp.Body.Close()
	requireCreated(s.T(), resp)
	var r struct {
		Data struct{ ID string `json:"id"` } `json:"data"`
	}
	parseResponse(s.T(), resp, &r)
	require.NotEmpty(s.T(), r.Data.ID)
	return r.Data.ID
}

func (s *FoldersProjectsTestSuite) createSubfolder(parentID, name string) string {
	resp := makeRequest(s.T(), s.client, "POST", s.baseURL+"/api/folders", s.authToken,
		map[string]interface{}{"name": name, "parent_id": parentID})
	defer resp.Body.Close()
	requireCreated(s.T(), resp)
	var r struct {
		Data struct{ ID string `json:"id"` } `json:"data"`
	}
	parseResponse(s.T(), resp, &r)
	return r.Data.ID
}

func (s *FoldersProjectsTestSuite) createProject(name string) string {
	resp := makeRequest(s.T(), s.client, "POST", s.baseURL+"/api/projects", s.authToken,
		map[string]string{"name": name})
	defer resp.Body.Close()
	requireCreated(s.T(), resp)
	var r struct {
		Data struct{ ID string `json:"id"` } `json:"data"`
	}
	parseResponse(s.T(), resp, &r)
	return r.Data.ID
}

func (s *FoldersProjectsTestSuite) createProjectInFolder(folderID, name string) string {
	resp := makeRequest(s.T(), s.client, "POST", s.baseURL+"/api/projects", s.authToken,
		map[string]interface{}{"name": name, "folder_id": folderID})
	defer resp.Body.Close()
	requireCreated(s.T(), resp)
	var r struct {
		Data struct{ ID string `json:"id"` } `json:"data"`
	}
	parseResponse(s.T(), resp, &r)
	return r.Data.ID
}

func TestFoldersProjectsSuite(t *testing.T) {
	suite.Run(t, new(FoldersProjectsTestSuite))
}
