package application

import (
	"modelmatrix_backend/internal/module/datasource/domain"
	"modelmatrix_backend/internal/module/datasource/dto"
)

// CollectionService defines the interface for collection application service
type CollectionService interface {
	Create(req *dto.CreateCollectionRequest, createdBy string) (*dto.CollectionResponse, error)
	Update(id string, req *dto.UpdateCollectionRequest) (*dto.CollectionResponse, error)
	Delete(id string) error
	GetByID(id string) (*dto.CollectionResponse, error)
	List(params *dto.ListParams) (*dto.CollectionListResponse, error)
}

// DatasourceService defines the interface for datasource application service
type DatasourceService interface {
	Create(req *dto.CreateDatasourceRequest, createdBy string) (*dto.DatasourceResponse, error)
	Update(id string, req *dto.UpdateDatasourceRequest) (*dto.DatasourceResponse, error)
	Delete(id string) error
	GetByID(id string) (*dto.DatasourceDetailResponse, error)
	List(collectionID *string, params *dto.ListParams) (*dto.DatasourceListResponse, error)
	UploadFile(datasourceID string, filename string, fileData []byte) (*dto.FileUploadResponse, error)
}

// ColumnService defines the interface for column application service
type ColumnService interface {
	GetByDatasourceID(datasourceID string) ([]dto.ColumnResponse, error)
	UpdateRole(datasourceID, columnID string, role string) (*dto.ColumnResponse, error)
	BulkUpdateRoles(datasourceID string, req *dto.BulkUpdateColumnRolesRequest) ([]dto.ColumnResponse, error)
	CreateColumns(datasourceID string, req *dto.CreateColumnsRequest) ([]dto.ColumnResponse, error)
}

// toCollectionResponse converts domain entity to DTO
func toCollectionResponse(collection *domain.Collection, datasourceCount int) *dto.CollectionResponse {
	return &dto.CollectionResponse{
		ID:              collection.ID,
		Name:            collection.Name,
		Description:     collection.Description,
		DatasourceCount: datasourceCount,
		CreatedBy:       collection.CreatedBy,
		CreatedAt:       collection.CreatedAt,
		UpdatedAt:       collection.UpdatedAt,
	}
}

// toDatasourceResponse converts domain entity to DTO
func toDatasourceResponse(datasource *domain.Datasource, collectionName string) *dto.DatasourceResponse {
	resp := &dto.DatasourceResponse{
		ID:             datasource.ID,
		CollectionID:   datasource.CollectionID,
		CollectionName: collectionName,
		Name:           datasource.Name,
		Description:    datasource.Description,
		Type:           string(datasource.Type),
		FilePath:       datasource.FilePath,
		ColumnCount:    len(datasource.Columns),
		CreatedBy:      datasource.CreatedBy,
		CreatedAt:      datasource.CreatedAt,
		UpdatedAt:      datasource.UpdatedAt,
	}

	if datasource.ConnectionConfig != nil {
		resp.ConnectionConfig = &dto.ConnectionConfigResponse{
			Host:     datasource.ConnectionConfig.Host,
			Port:     datasource.ConnectionConfig.Port,
			Database: datasource.ConnectionConfig.Database,
			Username: datasource.ConnectionConfig.Username,
			Schema:   datasource.ConnectionConfig.Schema,
			Table:    datasource.ConnectionConfig.Table,
			SSLMode:  datasource.ConnectionConfig.SSLMode,
		}
	}

	return resp
}

// toColumnResponse converts domain entity to DTO
func toColumnResponse(column *domain.Column) *dto.ColumnResponse {
	return &dto.ColumnResponse{
		ID:          column.ID,
		Name:        column.Name,
		DataType:    column.DataType,
		Role:        string(column.Role),
		Description: column.Description,
		CreatedAt:   column.CreatedAt,
		UpdatedAt:   column.UpdatedAt,
	}
}

// toColumnResponseList converts domain entities to DTOs
func toColumnResponseList(columns []domain.Column) []dto.ColumnResponse {
	result := make([]dto.ColumnResponse, len(columns))
	for i, col := range columns {
		result[i] = *toColumnResponse(&col)
	}
	return result
}

