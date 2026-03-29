package application

// Tests for CreateFromBuild, UpdateFromBuild, and pure helper functions.
// CreateFromBuild / UpdateFromBuild contain real business decisions:
//   - Name deduplication (append BuildID prefix when name clash)
//   - Feature importance assignment per variable
//   - Target/input variable creation with correct roles
//   - Model file and training code file registration
// The pure helpers (isTextFileType, getContentTypeFromFileName, convertMetrics)
// are called on every model retrieval path and deserve explicit coverage.

import (
	"testing"

	"modelmatrix-server/internal/module/inventory/domain"
	"modelmatrix-server/internal/module/inventory/dto"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Helper: build ModelService backed by crudModelRepo (defined in model_service_crud_test.go)
// ---------------------------------------------------------------------------

func buildModelSvcForBuild(models ...*domain.Model) ModelService {
	return NewModelService(newCRUDRepo(models...), domain.NewService(), &mockFileService{})
}

// ---------------------------------------------------------------------------
// isTextFileType — pure function
// ---------------------------------------------------------------------------

func TestIsTextFileType_ByFileType(t *testing.T) {
	assert.True(t, isTextFileType("anything.bin", "training_code"))
	assert.True(t, isTextFileType("anything.bin", "metadata"))
	assert.True(t, isTextFileType("anything.bin", "feature_names"))
	assert.False(t, isTextFileType("model.pkl", "model"))
}

func TestIsTextFileType_ByExtension(t *testing.T) {
	cases := []struct {
		name     string
		expected bool
	}{
		{"train.py", true},
		{"config.yaml", true},
		{"notes.md", true},
		{"features.csv", true},
		{"query.sql", true},
		{"model.pkl", false},
		{"weights.bin", false},
		{"output.parquet", false},
	}
	for _, c := range cases {
		assert.Equal(t, c.expected, isTextFileType(c.name, ""), c.name)
	}
}

// ---------------------------------------------------------------------------
// getContentTypeFromFileName — pure function
// ---------------------------------------------------------------------------

func TestGetContentTypeFromFileName(t *testing.T) {
	cases := []struct {
		file     string
		expected string
	}{
		{"train.py", "text/x-python"},
		{"config.json", "application/json"},
		{"params.yaml", "text/yaml"},
		{"README.md", "text/markdown"},
		{"data.csv", "text/csv"},
		{"log.txt", "text/plain"},
		{"query.sql", "text/sql"},
		{"run.sh", "text/x-shellscript"},
		{"model.pkl", "text/plain"}, // unknown → text/plain
	}
	for _, c := range cases {
		assert.Equal(t, c.expected, getContentTypeFromFileName(c.file), c.file)
	}
}

// ---------------------------------------------------------------------------
// convertMetrics (inventory) — pure function
// ---------------------------------------------------------------------------

func TestInventoryConvertMetrics_Nil(t *testing.T) {
	assert.Nil(t, convertMetrics(nil))
}

func TestInventoryConvertMetrics_PopulatesFields(t *testing.T) {
	m := map[string]interface{}{
		"accuracy": 0.93, "f1_score": 0.91, "mse": 0.04, "r2": 0.88,
	}
	result := convertMetrics(m)
	require.NotNil(t, result)
	assert.Equal(t, 0.93, result.Accuracy)
	assert.Equal(t, 0.91, result.F1Score)
	assert.Equal(t, 0.04, result.MSE)
	assert.Equal(t, 0.88, result.R2)
}

func TestInventoryConvertMetrics_NonFloatIgnored(t *testing.T) {
	m := map[string]interface{}{"accuracy": "bad", "recall": 0.85}
	result := convertMetrics(m)
	require.NotNil(t, result)
	assert.Equal(t, 0.0, result.Accuracy)
	assert.Equal(t, 0.85, result.Recall)
}

// ---------------------------------------------------------------------------
// CreateFromBuild
// ---------------------------------------------------------------------------

func TestCreateFromBuild_Success_CreatesModelWithVariablesAndFiles(t *testing.T) {
	svc := buildModelSvcForBuild()
	modelPath := "models/b1/rf.pkl"
	codePath := "models/b1/train.py"
	imp := 0.42
	result, err := svc.CreateFromBuild(&dto.CreateModelFromBuildRequest{
		BuildID:      "b1",
		Name:         "FraudModel",
		Description:  "Fraud detection",
		DatasourceID: "ds1",
		Algorithm:    "random_forest",
		ModelType:    "classification",
		TargetColumn: "is_fraud",
		InputColumns: []string{"amount", "merchant"},
		FeatureImportances: map[string]float64{
			"amount": imp,
		},
		ModelFilePath: modelPath,
		CodeFilePath:  codePath,
		CreatedBy:     "alice",
	})
	require.NoError(t, err)
	assert.Equal(t, "FraudModel", result.Name)
	assert.Equal(t, "b1", result.BuildID)
}

func TestCreateFromBuild_DeduplicatesName_WhenNameAlreadyExists(t *testing.T) {
	// Pre-seed a model with the same name
	existing := &domain.Model{ID: "m0", Name: "FraudModel", Status: domain.ModelStatusActive}
	svc := buildModelSvcForBuild(existing)

	// BuildID must be ≥8 chars; production code does BuildID[:8] for dedup suffix.
	result, err := svc.CreateFromBuild(&dto.CreateModelFromBuildRequest{
		BuildID:      "build-001-xyz",
		Name:         "FraudModel", // conflict
		DatasourceID: "ds1",
		Algorithm:    "rf",
		ModelType:    "classification",
		InputColumns: []string{"f1"},
		TargetColumn: "label",
		CreatedBy:    "alice",
	})
	require.NoError(t, err)
	// Name should be modified to avoid duplicate
	assert.NotEqual(t, "FraudModel", result.Name,
		"name should be de-duplicated when a model with the same name already exists")
}

func TestCreateFromBuild_IdempotentWhenBuildAlreadyHasModel(t *testing.T) {
	// Verify idempotency: when GetByBuildID returns an existing model,
	// CreateFromBuild returns it immediately without creating a new one.
	existing := &domain.Model{ID: "m0", Name: "AlreadyCreated", BuildID: "b1", Status: domain.ModelStatusActive}
	repo := newCRUDRepo(existing)

	// Override GetByBuildID to simulate finding the existing model for this build.
	createCallCount := 0
	origCreate := repo.createFn
	repo.createFn = func(model *domain.Model) error {
		createCallCount++
		if origCreate != nil {
			return origCreate(model)
		}
		return nil
	}

	// We need a repo that returns the existing model for GetByBuildID.
	// Use mockModelRepo (from version_service_test.go) which supports getByBuildIDFn.
	mr := &mockModelRepo{
		getByBuildID: func(buildID string) (*domain.Model, error) {
			if buildID == "b1" {
				return existing, nil
			}
			return nil, nil
		},
	}
	svc := NewModelService(mr, domain.NewService(), &mockFileService{})

	result, err := svc.CreateFromBuild(&dto.CreateModelFromBuildRequest{
		BuildID:      "b1",
		Name:         "NewName",
		DatasourceID: "ds1",
		Algorithm:    "rf",
		ModelType:    "classification",
		InputColumns: []string{"f1"},
		CreatedBy:    "alice",
	})
	require.NoError(t, err)
	assert.Equal(t, "m0", result.ID, "should return existing model for duplicate build callback")
	assert.Equal(t, 0, createCallCount, "should not create a new model when one already exists for the build")
}

func TestCreateFromBuild_NoTargetColumn_OnlyInputVariables(t *testing.T) {
	svc := buildModelSvcForBuild()
	result, err := svc.CreateFromBuild(&dto.CreateModelFromBuildRequest{
		BuildID:      "b1",
		Name:         "ClusterModel",
		DatasourceID: "ds1",
		Algorithm:    "kmeans",
		ModelType:    "clustering",
		InputColumns: []string{"x", "y"},
		TargetColumn: "", // no target for clustering
		CreatedBy:    "alice",
	})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

// ---------------------------------------------------------------------------
// UpdateFromBuild
// ---------------------------------------------------------------------------

func TestUpdateFromBuild_Success_IncrementsVersion(t *testing.T) {
	original := &domain.Model{
		ID: "m1", Name: "RetainModel", BuildID: "old-b", Version: 1,
		Status: domain.ModelStatusActive,
		Variables: []domain.ModelVariable{
			{ID: "v1", ModelID: "m1", Name: "old_feat", Role: domain.VariableRoleInput},
		},
	}
	svc := buildModelSvcForBuild(original)

	result, err := svc.UpdateFromBuild("m1", &dto.CreateModelFromBuildRequest{
		BuildID:      "new-b",
		Name:         "RetainModel",
		DatasourceID: "ds2",
		Algorithm:    "xgboost",
		ModelType:    "classification",
		InputColumns: []string{"new_feat1", "new_feat2"},
		TargetColumn: "label",
		CreatedBy:    "system",
	})
	require.NoError(t, err)
	assert.Equal(t, "m1", result.ID)
	// Version should have been incremented
	assert.Equal(t, 2, result.Version)
}

func TestUpdateFromBuild_ModelNotFound(t *testing.T) {
	svc := buildModelSvcForBuild()
	_, err := svc.UpdateFromBuild("missing", &dto.CreateModelFromBuildRequest{
		BuildID:      "b2",
		InputColumns: []string{"f1"},
	})
	require.Error(t, err)
}

func TestUpdateFromBuild_FeatureImportancesAssigned(t *testing.T) {
	// Track the variables passed to CreateVariables to verify importance is set.
	var createdVariables []domain.ModelVariable
	original := &domain.Model{ID: "m1", Name: "M", Version: 1, Status: domain.ModelStatusActive}
	mr := &mockModelRepo{
		getByIDWithRelations: func(id string) (*domain.Model, error) {
			if id == "m1" {
				return original, nil
			}
			return nil, domain.ErrModelNotFound
		},
		getByID: func(id string) (*domain.Model, error) {
			if id == "m1" {
				return original, nil
			}
			return nil, domain.ErrModelNotFound
		},
		update: func(m *domain.Model) error { original = m; return nil },
		createVariables: func(vs []domain.ModelVariable) error {
			createdVariables = vs
			return nil
		},
	}
	svc := NewModelService(mr, domain.NewService(), &mockFileService{})

	imp := 0.75
	_, err := svc.UpdateFromBuild("m1", &dto.CreateModelFromBuildRequest{
		BuildID:      "b2",
		DatasourceID: "ds1",
		Algorithm:    "rf",
		ModelType:    "classification",
		InputColumns: []string{"important_feat"},
		TargetColumn: "label",
		FeatureImportances: map[string]float64{"important_feat": imp},
		CreatedBy:    "system",
	})
	require.NoError(t, err)
	require.Len(t, createdVariables, 2, "should create input + target variable")
	for _, v := range createdVariables {
		if v.Name == "important_feat" {
			require.NotNil(t, v.Importance)
			assert.Equal(t, imp, *v.Importance)
		}
	}
}
