package application

import (
	"errors"
	"testing"

	"modelmatrix-server/internal/module/folder/domain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Mock FolderRepository
// ---------------------------------------------------------------------------

type fakeFolderRepo struct {
	folders         map[string]*domain.Folder
	byParentAndName map[string]*domain.Folder // key: "parentID::name" or "root::name"
	childCounts     map[string]int64
	projectCounts   map[string]int64
	contentCounts   map[string]*domain.FolderContentsCount
	descendantIDs   map[string][]string // pathPattern → folderIDs
}

func newFakeFolderRepo(folders ...*domain.Folder) *fakeFolderRepo {
	r := &fakeFolderRepo{
		folders:         make(map[string]*domain.Folder),
		byParentAndName: make(map[string]*domain.Folder),
		childCounts:     make(map[string]int64),
		projectCounts:   make(map[string]int64),
		contentCounts:   make(map[string]*domain.FolderContentsCount),
		descendantIDs:   make(map[string][]string),
	}
	for _, f := range folders {
		r.folders[f.ID] = f
	}
	return r
}

func folderKey(parentID *string, name string) string {
	if parentID == nil {
		return "root::" + name
	}
	return *parentID + "::" + name
}

func (r *fakeFolderRepo) Create(f *domain.Folder) error {
	if f.ID == "" {
		f.ID = "gen-" + f.Name
	}
	if f.Path == "" {
		f.Path = "/" + f.ID
	}
	r.folders[f.ID] = f
	return nil
}
func (r *fakeFolderRepo) Update(f *domain.Folder) error { r.folders[f.ID] = f; return nil }
func (r *fakeFolderRepo) UpdatePath(id, path string) error {
	if f, ok := r.folders[id]; ok {
		f.Path = path
	}
	return nil
}
func (r *fakeFolderRepo) Delete(id string) error { delete(r.folders, id); return nil }
func (r *fakeFolderRepo) GetByID(id string) (*domain.Folder, error) {
	if f, ok := r.folders[id]; ok {
		return f, nil
	}
	return nil, domain.ErrFolderNotFound
}
func (r *fakeFolderRepo) GetChildren(parentID string) ([]domain.Folder, error) {
	var result []domain.Folder
	for _, f := range r.folders {
		if f.ParentID != nil && *f.ParentID == parentID {
			result = append(result, *f)
		}
	}
	return result, nil
}
func (r *fakeFolderRepo) GetRootFolders() ([]domain.Folder, error) {
	var result []domain.Folder
	for _, f := range r.folders {
		if f.ParentID == nil {
			result = append(result, *f)
		}
	}
	return result, nil
}
func (r *fakeFolderRepo) GetPath(id string) ([]domain.Folder, error)       { return nil, nil }
func (r *fakeFolderRepo) GetDescendants(id string) ([]domain.Folder, error) { return nil, nil }
func (r *fakeFolderRepo) GetByParentIDAndName(parentID *string, name string) (*domain.Folder, error) {
	key := folderKey(parentID, name)
	if f, ok := r.byParentAndName[key]; ok {
		return f, nil
	}
	return nil, nil
}
func (r *fakeFolderRepo) CountChildren(id string) (int64, error) { return r.childCounts[id], nil }
func (r *fakeFolderRepo) CountProjects(id string) (int64, error) { return r.projectCounts[id], nil }
func (r *fakeFolderRepo) GetContentsCount(id string) (*domain.FolderContentsCount, error) {
	if cc, ok := r.contentCounts[id]; ok {
		return cc, nil
	}
	return &domain.FolderContentsCount{}, nil
}
func (r *fakeFolderRepo) GetDescendantFolderIDs(pathPattern string) ([]string, error) {
	return r.descendantIDs[pathPattern], nil
}
func (r *fakeFolderRepo) DeleteDescendants(pathPattern string) error { return nil }

// ---------------------------------------------------------------------------
// Mock ProjectRepository
// ---------------------------------------------------------------------------

type fakeProjectRepo struct {
	projects    map[string]*domain.Project
	modelCounts map[string]int64
	buildCounts map[string]int64
}

func newFakeProjectRepo(projects ...*domain.Project) *fakeProjectRepo {
	r := &fakeProjectRepo{
		projects:    make(map[string]*domain.Project),
		modelCounts: make(map[string]int64),
		buildCounts: make(map[string]int64),
	}
	for _, p := range projects {
		r.projects[p.ID] = p
	}
	return r
}

func (r *fakeProjectRepo) Create(p *domain.Project) error {
	if p.ID == "" {
		p.ID = "gen-proj-" + p.Name
	}
	r.projects[p.ID] = p
	return nil
}
func (r *fakeProjectRepo) Update(p *domain.Project) error { r.projects[p.ID] = p; return nil }
func (r *fakeProjectRepo) Delete(id string) error          { delete(r.projects, id); return nil }
func (r *fakeProjectRepo) GetByID(id string) (*domain.Project, error) {
	if p, ok := r.projects[id]; ok {
		return p, nil
	}
	return nil, domain.ErrProjectNotFound
}
func (r *fakeProjectRepo) GetByFolderID(folderID string) ([]domain.Project, error) {
	var result []domain.Project
	for _, p := range r.projects {
		if p.FolderID != nil && *p.FolderID == folderID {
			result = append(result, *p)
		}
	}
	return result, nil
}
func (r *fakeProjectRepo) GetRootProjects() ([]domain.Project, error) {
	var result []domain.Project
	for _, p := range r.projects {
		if p.FolderID == nil {
			result = append(result, *p)
		}
	}
	return result, nil
}
func (r *fakeProjectRepo) GetByFolderIDAndName(folderID *string, name string) (*domain.Project, error) {
	for _, p := range r.projects {
		if p.Name == name {
			if folderID == nil && p.FolderID == nil {
				return p, nil
			}
			if folderID != nil && p.FolderID != nil && *folderID == *p.FolderID {
				return p, nil
			}
		}
	}
	return nil, nil
}
func (r *fakeProjectRepo) CountModels(id string) (int64, error) { return r.modelCounts[id], nil }
func (r *fakeProjectRepo) CountBuilds(id string) (int64, error) { return r.buildCounts[id], nil }
func (r *fakeProjectRepo) DeleteByFolderIDs(ids []string) error { return nil }

// ---------------------------------------------------------------------------
// Stub ModelDeleter / BuildDeleter
// ---------------------------------------------------------------------------

type stubDeleter struct{}

func (s *stubDeleter) DeleteByFolderID(folderID string) error  { return nil }
func (s *stubDeleter) DeleteByProjectID(projectID string) error { return nil }

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func buildFolderSvc(fr *fakeFolderRepo, pr *fakeProjectRepo) FolderService {
	svc := NewFolderService(nil, fr, pr)
	svc.SetModelDeleter(&stubDeleter{})
	svc.SetBuildDeleter(&stubDeleter{})
	return svc
}

// ---------------------------------------------------------------------------
// CreateFolder
// ---------------------------------------------------------------------------

func TestFolderService_CreateFolder_Valid(t *testing.T) {
	svc := buildFolderSvc(newFakeFolderRepo(), newFakeProjectRepo())
	f, err := svc.CreateFolder("Analytics", "desc", nil, "alice")
	require.NoError(t, err)
	assert.Equal(t, "Analytics", f.Name)
	assert.Equal(t, "alice", f.CreatedBy)
}

func TestFolderService_CreateFolder_EmptyName(t *testing.T) {
	svc := buildFolderSvc(newFakeFolderRepo(), newFakeProjectRepo())
	_, err := svc.CreateFolder("  ", "", nil, "alice")
	require.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrFolderNameEmpty))
}

func TestFolderService_CreateFolder_Duplicate(t *testing.T) {
	fr := newFakeFolderRepo()
	fr.byParentAndName["root::Analytics"] = &domain.Folder{ID: "f1", Name: "Analytics"}
	svc := buildFolderSvc(fr, newFakeProjectRepo())
	_, err := svc.CreateFolder("Analytics", "", nil, "alice")
	require.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrFolderNameExists))
}

func TestFolderService_CreateFolder_WithParent(t *testing.T) {
	parent := &domain.Folder{ID: "p1", Name: "Parent", Path: "/p1"}
	fr := newFakeFolderRepo(parent)
	svc := buildFolderSvc(fr, newFakeProjectRepo())
	parentID := "p1"
	f, err := svc.CreateFolder("Child", "", &parentID, "bob")
	require.NoError(t, err)
	assert.Equal(t, "Child", f.Name)
	assert.Equal(t, 1, f.Depth)
}

// ---------------------------------------------------------------------------
// GetFolder
// ---------------------------------------------------------------------------

func TestFolderService_GetFolder_Found(t *testing.T) {
	folder := &domain.Folder{ID: "f1", Name: "Prod"}
	svc := buildFolderSvc(newFakeFolderRepo(folder), newFakeProjectRepo())
	result, err := svc.GetFolder("f1")
	require.NoError(t, err)
	assert.Equal(t, "Prod", result.Name)
}

func TestFolderService_GetFolder_NotFound(t *testing.T) {
	svc := buildFolderSvc(newFakeFolderRepo(), newFakeProjectRepo())
	_, err := svc.GetFolder("missing")
	require.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrFolderNotFound))
}

// ---------------------------------------------------------------------------
// UpdateFolder
// ---------------------------------------------------------------------------

func TestFolderService_UpdateFolder_Valid(t *testing.T) {
	folder := &domain.Folder{ID: "f1", Name: "Old", Path: "/f1"}
	svc := buildFolderSvc(newFakeFolderRepo(folder), newFakeProjectRepo())
	result, err := svc.UpdateFolder("f1", "New Name", "desc")
	require.NoError(t, err)
	assert.Equal(t, "New Name", result.Name)
}

func TestFolderService_UpdateFolder_EmptyName(t *testing.T) {
	folder := &domain.Folder{ID: "f1", Name: "Prod"}
	svc := buildFolderSvc(newFakeFolderRepo(folder), newFakeProjectRepo())
	_, err := svc.UpdateFolder("f1", "", "")
	require.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrFolderNameEmpty))
}

func TestFolderService_UpdateFolder_NotFound(t *testing.T) {
	svc := buildFolderSvc(newFakeFolderRepo(), newFakeProjectRepo())
	_, err := svc.UpdateFolder("missing", "Name", "")
	require.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrFolderNotFound))
}

// ---------------------------------------------------------------------------
// DeleteFolder
// ---------------------------------------------------------------------------

func TestFolderService_DeleteFolder_Empty(t *testing.T) {
	folder := &domain.Folder{ID: "f1", Name: "Empty", Path: "/f1"}
	svc := buildFolderSvc(newFakeFolderRepo(folder), newFakeProjectRepo())
	err := svc.DeleteFolder("f1", false)
	require.NoError(t, err)
}

func TestFolderService_DeleteFolder_HasChildren_Rejected(t *testing.T) {
	folder := &domain.Folder{ID: "f1", Name: "Parent", Path: "/f1"}
	fr := newFakeFolderRepo(folder)
	fr.childCounts["f1"] = 3
	svc := buildFolderSvc(fr, newFakeProjectRepo())
	err := svc.DeleteFolder("f1", false)
	require.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrFolderHasChildren))
}

func TestFolderService_DeleteFolder_HasProjects_Rejected(t *testing.T) {
	folder := &domain.Folder{ID: "f1", Name: "Folder", Path: "/f1"}
	fr := newFakeFolderRepo(folder)
	fr.projectCounts["f1"] = 2
	svc := buildFolderSvc(fr, newFakeProjectRepo())
	err := svc.DeleteFolder("f1", false)
	require.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrFolderHasProjects))
}

func TestFolderService_DeleteFolder_NotFound(t *testing.T) {
	svc := buildFolderSvc(newFakeFolderRepo(), newFakeProjectRepo())
	err := svc.DeleteFolder("missing", false)
	require.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrFolderNotFound))
}

// ---------------------------------------------------------------------------
// GetRootFolders / GetChildren / GetFolderContentsCount
// ---------------------------------------------------------------------------

func TestFolderService_GetRootFolders(t *testing.T) {
	root1 := &domain.Folder{ID: "r1", Name: "Root1"}
	root2 := &domain.Folder{ID: "r2", Name: "Root2"}
	parentID := "r1"
	child := &domain.Folder{ID: "c1", Name: "Child", ParentID: &parentID}
	svc := buildFolderSvc(newFakeFolderRepo(root1, root2, child), newFakeProjectRepo())
	roots, err := svc.GetRootFolders()
	require.NoError(t, err)
	assert.Len(t, roots, 2)
}

func TestFolderService_GetChildren(t *testing.T) {
	parentID := "p1"
	child1 := &domain.Folder{ID: "c1", Name: "A", ParentID: &parentID}
	child2 := &domain.Folder{ID: "c2", Name: "B", ParentID: &parentID}
	svc := buildFolderSvc(newFakeFolderRepo(child1, child2), newFakeProjectRepo())
	children, err := svc.GetChildren("p1")
	require.NoError(t, err)
	assert.Len(t, children, 2)
}

func TestFolderService_GetFolderContentsCount(t *testing.T) {
	folder := &domain.Folder{ID: "f1", Name: "Folder", Path: "/f1"}
	fr := newFakeFolderRepo(folder)
	fr.contentCounts["f1"] = &domain.FolderContentsCount{ProjectCount: 3, BuildCount: 5}
	svc := buildFolderSvc(fr, newFakeProjectRepo())
	counts, err := svc.GetFolderContentsCount("f1")
	require.NoError(t, err)
	assert.Equal(t, int64(3), counts.ProjectCount)
	assert.Equal(t, int64(5), counts.BuildCount)
}

// ---------------------------------------------------------------------------
// CreateProject
// ---------------------------------------------------------------------------

func TestFolderService_CreateProject_Valid(t *testing.T) {
	svc := buildFolderSvc(newFakeFolderRepo(), newFakeProjectRepo())
	p, err := svc.CreateProject("Project A", "desc", nil, "alice")
	require.NoError(t, err)
	assert.Equal(t, "Project A", p.Name)
}

func TestFolderService_CreateProject_EmptyName(t *testing.T) {
	svc := buildFolderSvc(newFakeFolderRepo(), newFakeProjectRepo())
	_, err := svc.CreateProject("", "", nil, "alice")
	require.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrProjectNameEmpty))
}

func TestFolderService_CreateProject_DuplicateName(t *testing.T) {
	existing := &domain.Project{ID: "p1", Name: "Alpha"}
	svc := buildFolderSvc(newFakeFolderRepo(), newFakeProjectRepo(existing))
	_, err := svc.CreateProject("Alpha", "", nil, "alice")
	require.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrProjectNameExists))
}

func TestFolderService_CreateProject_InvalidFolder(t *testing.T) {
	folderID := "nonexistent"
	svc := buildFolderSvc(newFakeFolderRepo(), newFakeProjectRepo())
	_, err := svc.CreateProject("P", "", &folderID, "alice")
	require.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrFolderNotFound))
}

// ---------------------------------------------------------------------------
// GetProject / UpdateProject
// ---------------------------------------------------------------------------

func TestFolderService_GetProject_Found(t *testing.T) {
	p := &domain.Project{ID: "p1", Name: "Fraud Detection"}
	svc := buildFolderSvc(newFakeFolderRepo(), newFakeProjectRepo(p))
	result, err := svc.GetProject("p1")
	require.NoError(t, err)
	assert.Equal(t, "Fraud Detection", result.Name)
}

func TestFolderService_GetProject_NotFound(t *testing.T) {
	svc := buildFolderSvc(newFakeFolderRepo(), newFakeProjectRepo())
	_, err := svc.GetProject("missing")
	require.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrProjectNotFound))
}

func TestFolderService_UpdateProject_Valid(t *testing.T) {
	p := &domain.Project{ID: "p1", Name: "Old Name"}
	svc := buildFolderSvc(newFakeFolderRepo(), newFakeProjectRepo(p))
	result, err := svc.UpdateProject("p1", "New Name", "desc")
	require.NoError(t, err)
	assert.Equal(t, "New Name", result.Name)
}

func TestFolderService_UpdateProject_EmptyName(t *testing.T) {
	p := &domain.Project{ID: "p1", Name: "Alpha"}
	svc := buildFolderSvc(newFakeFolderRepo(), newFakeProjectRepo(p))
	_, err := svc.UpdateProject("p1", "", "")
	require.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrProjectNameEmpty))
}

// ---------------------------------------------------------------------------
// DeleteProject
// ---------------------------------------------------------------------------

func TestFolderService_DeleteProject_Empty(t *testing.T) {
	p := &domain.Project{ID: "p1", Name: "Empty Project"}
	svc := buildFolderSvc(newFakeFolderRepo(), newFakeProjectRepo(p))
	err := svc.DeleteProject("p1", false)
	require.NoError(t, err)
}

func TestFolderService_DeleteProject_HasModels_Rejected(t *testing.T) {
	p := &domain.Project{ID: "p1", Name: "With Models"}
	pr := newFakeProjectRepo(p)
	pr.modelCounts["p1"] = 5
	svc := buildFolderSvc(newFakeFolderRepo(), pr)
	err := svc.DeleteProject("p1", false)
	require.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrProjectHasModels))
}

func TestFolderService_DeleteProject_HasBuilds_Rejected(t *testing.T) {
	p := &domain.Project{ID: "p1", Name: "With Builds"}
	pr := newFakeProjectRepo(p)
	pr.buildCounts["p1"] = 3
	svc := buildFolderSvc(newFakeFolderRepo(), pr)
	err := svc.DeleteProject("p1", false)
	require.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrProjectHasBuilds))
}

func TestFolderService_DeleteProject_ForceWithModels(t *testing.T) {
	p := &domain.Project{ID: "p1", Name: "With Models"}
	pr := newFakeProjectRepo(p)
	pr.modelCounts["p1"] = 5
	svc := buildFolderSvc(newFakeFolderRepo(), pr)
	err := svc.DeleteProject("p1", true)
	require.NoError(t, err)
}

func TestFolderService_DeleteProject_NotFound(t *testing.T) {
	svc := buildFolderSvc(newFakeFolderRepo(), newFakeProjectRepo())
	err := svc.DeleteProject("missing", false)
	require.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrProjectNotFound))
}

// ---------------------------------------------------------------------------
// GetProjectsInFolder / GetRootProjects
// ---------------------------------------------------------------------------

func TestFolderService_GetProjectsInFolder(t *testing.T) {
	folderID := "f1"
	p1 := &domain.Project{ID: "p1", Name: "P1", FolderID: &folderID}
	p2 := &domain.Project{ID: "p2", Name: "P2", FolderID: &folderID}
	p3 := &domain.Project{ID: "p3", Name: "P3"} // different folder
	svc := buildFolderSvc(newFakeFolderRepo(), newFakeProjectRepo(p1, p2, p3))
	projects, err := svc.GetProjectsInFolder("f1")
	require.NoError(t, err)
	assert.Len(t, projects, 2)
}

func TestFolderService_GetRootProjects(t *testing.T) {
	folderID := "f1"
	root1 := &domain.Project{ID: "p1", Name: "Root1"}
	root2 := &domain.Project{ID: "p2", Name: "Root2"}
	nested := &domain.Project{ID: "p3", Name: "Nested", FolderID: &folderID}
	svc := buildFolderSvc(newFakeFolderRepo(), newFakeProjectRepo(root1, root2, nested))
	projects, err := svc.GetRootProjects()
	require.NoError(t, err)
	assert.Len(t, projects, 2)
}

// ---------------------------------------------------------------------------
// cascadeDeleteFolder — via DeleteFolder(force=true)
// ---------------------------------------------------------------------------

// trackingDeleter records which IDs were deleted so tests can assert behavior
type trackingDeleter struct {
	folderIDs  []string
	projectIDs []string
}

func (d *trackingDeleter) DeleteByFolderID(id string) error {
	d.folderIDs = append(d.folderIDs, id)
	return nil
}
func (d *trackingDeleter) DeleteByProjectID(id string) error {
	d.projectIDs = append(d.projectIDs, id)
	return nil
}

func buildFolderSvcWithTrackers(fr *fakeFolderRepo, pr *fakeProjectRepo, md, bd *trackingDeleter) FolderService {
	svc := NewFolderService(nil, fr, pr)
	svc.SetModelDeleter(md)
	svc.SetBuildDeleter(bd)
	return svc
}

func TestFolderService_DeleteFolder_Force_CascadesModelsAndBuilds(t *testing.T) {
	// Set up: folder f1 with a direct project p1 and a descendant folder f2 with project p2
	folderID := "f1"
	folder := &domain.Folder{ID: "f1", Name: "Root", Path: "/f1"}
	folderID2 := "f2"
	p1 := &domain.Project{ID: "p1", Name: "DirectProject", FolderID: &folderID}
	p2 := &domain.Project{ID: "p2", Name: "DescendantProject", FolderID: &folderID2}

	fr := newFakeFolderRepo(folder)
	// f2 is a descendant of /f1
	fr.descendantIDs["/f1/%"] = []string{"f2"}

	pr := newFakeProjectRepo(p1, p2)

	md := &trackingDeleter{}
	bd := &trackingDeleter{}
	svc := buildFolderSvcWithTrackers(fr, pr, md, bd)

	err := svc.DeleteFolder("f1", true)
	require.NoError(t, err)

	// Expect model deleter called for: f2 (descendant folder), p2 (descendant project), f1 (direct folder), p1 (direct project)
	assert.Contains(t, md.folderIDs, "f2")
	assert.Contains(t, md.folderIDs, "f1")
	assert.Contains(t, md.projectIDs, "p1")
	assert.Contains(t, md.projectIDs, "p2")
	// Build deleter should mirror model deleter
	assert.Contains(t, bd.folderIDs, "f1")
}

func TestFolderService_DeleteFolder_Force_EmptyFolder(t *testing.T) {
	// No descendants, no projects — should succeed cleanly
	folder := &domain.Folder{ID: "f1", Name: "Empty", Path: "/f1"}
	fr := newFakeFolderRepo(folder)
	md := &trackingDeleter{}
	bd := &trackingDeleter{}
	svc := buildFolderSvcWithTrackers(fr, newFakeProjectRepo(), md, bd)

	err := svc.DeleteFolder("f1", true)
	require.NoError(t, err)
	// Model deleter should still be called for the folder itself
	assert.Contains(t, md.folderIDs, "f1")
}

func TestFolderService_DeleteFolder_NotForce_HasChildren_Rejected(t *testing.T) {
	folder := &domain.Folder{ID: "f1", Name: "WithChildren", Path: "/f1"}
	fr := newFakeFolderRepo(folder)
	fr.childCounts["f1"] = 2

	svc := buildFolderSvc(fr, newFakeProjectRepo())
	err := svc.DeleteFolder("f1", false)
	require.Error(t, err)
	assert.Equal(t, domain.ErrFolderHasChildren, err)
}
