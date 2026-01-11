package fileservice

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"modelmatrix-server/pkg/config"
	"modelmatrix-server/pkg/logger"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// FileInfo represents metadata about a stored file
type FileInfo struct {
	ID           string    `json:"id"`
	OriginalName string    `json:"original_name"`
	Path         string    `json:"path"`
	Size         int64     `json:"size"`
	ContentType  string    `json:"content_type"`
	LastModified time.Time `json:"last_modified"`
	ETag         string    `json:"etag"`
}

// FileService defines the interface for file operations
type FileService interface {
	Save(filename string, reader io.Reader, size int64) (*FileInfo, error)
	SaveWithPath(subPath, filename string, reader io.Reader, size int64) (*FileInfo, error)
	Get(fileID string) (io.ReadCloser, *FileInfo, error)
	ReadFileContent(fileID string) ([]byte, *FileInfo, error)
	Delete(fileID string) error
	Exists(fileID string) bool
	GetInfo(fileID string) (*FileInfo, error)
	ValidateParquet(fileID string) error
	ValidateCSV(fileID string) error
	HealthCheck() error
}

// MinioFileService implements FileService using MinIO
type MinioFileService struct {
	client     *minio.Client
	bucketName string
}

// NewFileService creates a new FileService based on configuration
func NewFileService(cfg *config.FileServiceConfig) (FileService, error) {
	client, err := minio.New(cfg.MinioEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.MinioAccessKey, cfg.MinioSecretKey, ""),
		Secure: cfg.MinioUseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create minio client: %w", err)
	}

	storage := &MinioFileService{
		client:     client,
		bucketName: cfg.MinioBucket,
	}

	// Ensure bucket exists
	ctx := context.Background()
	if err := storage.ensureBucket(ctx); err != nil {
		return nil, err
	}

	logger.Info("MinIO file service initialized: endpoint=%s, bucket=%s", cfg.MinioEndpoint, cfg.MinioBucket)
	return storage, nil
}

// ensureBucket creates the bucket if it doesn't exist
func (s *MinioFileService) ensureBucket(ctx context.Context) error {
	exists, err := s.client.BucketExists(ctx, s.bucketName)
	if err != nil {
		return fmt.Errorf("failed to check bucket existence: %w", err)
	}

	if !exists {
		err = s.client.MakeBucket(ctx, s.bucketName, minio.MakeBucketOptions{})
		if err != nil {
			return fmt.Errorf("failed to create bucket: %w", err)
		}
		logger.Info("Created MinIO bucket: %s", s.bucketName)
	}

	return nil
}

// Save saves a file to MinIO storage
func (s *MinioFileService) Save(filename string, reader io.Reader, size int64) (*FileInfo, error) {
	return s.SaveWithPath("", filename, reader, size)
}

// SaveWithPath saves a file to a specific path prefix in MinIO
func (s *MinioFileService) SaveWithPath(subPath, filename string, reader io.Reader, size int64) (*FileInfo, error) {
	ctx := context.Background()

	// Generate unique ID for storage, preserving the file extension
	fileID := uuid.New().String()
	ext := strings.ToLower(filepath.Ext(filename))
	if ext != "" {
		fileID = fileID + ext // e.g., "uuid.csv" or "uuid.parquet"
	}

	// Build object key with optional subpath
	objectKey := fileID
	if subPath != "" {
		objectKey = fmt.Sprintf("%s/%s", strings.Trim(subPath, "/"), fileID)
	}

	// Determine content type
	contentType := getContentType(filename)

	// Store original filename in metadata
	userMetadata := map[string]string{
		"original-name": filename,
	}

	info, err := s.client.PutObject(ctx, s.bucketName, objectKey, reader, size, minio.PutObjectOptions{
		ContentType:  contentType,
		UserMetadata: userMetadata,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to upload file: %w", err)
	}

	return &FileInfo{
		ID:           objectKey,
		OriginalName: filename,
		Path:         objectKey,
		Size:         info.Size,
		ContentType:  contentType,
		ETag:         info.ETag,
		LastModified: time.Now(),
	}, nil
}

// Get retrieves a file from MinIO storage
func (s *MinioFileService) Get(fileID string) (io.ReadCloser, *FileInfo, error) {
	ctx := context.Background()

	obj, err := s.client.GetObject(ctx, s.bucketName, fileID, minio.GetObjectOptions{})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get file: %w", err)
	}

	stat, err := obj.Stat()
	if err != nil {
		obj.Close()
		return nil, nil, fmt.Errorf("failed to get file info: %w", err)
	}

	// Get original name from metadata
	originalName := stat.UserMetadata["Original-Name"]
	if originalName == "" {
		originalName = filepath.Base(stat.Key)
	}

	fileInfo := &FileInfo{
		ID:           stat.Key,
		OriginalName: originalName,
		Path:         stat.Key,
		Size:         stat.Size,
		ContentType:  stat.ContentType,
		LastModified: stat.LastModified,
		ETag:         stat.ETag,
	}

	return obj, fileInfo, nil
}

// Delete removes a file from MinIO storage
func (s *MinioFileService) Delete(fileID string) error {
	ctx := context.Background()
	err := s.client.RemoveObject(ctx, s.bucketName, fileID, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}
	return nil
}

// Exists checks if a file exists in storage
func (s *MinioFileService) Exists(fileID string) bool {
	ctx := context.Background()
	_, err := s.client.StatObject(ctx, s.bucketName, fileID, minio.StatObjectOptions{})
	if err != nil {
		errResponse := minio.ToErrorResponse(err)
		if errResponse.Code == "NoSuchKey" {
			return false
		}
		return false
	}
	return true
}

// GetInfo retrieves file metadata without downloading
func (s *MinioFileService) GetInfo(fileID string) (*FileInfo, error) {
	ctx := context.Background()

	stat, err := s.client.StatObject(ctx, s.bucketName, fileID, minio.StatObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	originalName := stat.UserMetadata["Original-Name"]
	if originalName == "" {
		originalName = filepath.Base(stat.Key)
	}

	return &FileInfo{
		ID:           stat.Key,
		OriginalName: originalName,
		Path:         stat.Key,
		Size:         stat.Size,
		ContentType:  stat.ContentType,
		LastModified: stat.LastModified,
		ETag:         stat.ETag,
	}, nil
}

// ValidateParquet validates a Parquet file
func (s *MinioFileService) ValidateParquet(fileID string) error {
	ctx := context.Background()

	// Get object for reading
	obj, err := s.client.GetObject(ctx, s.bucketName, fileID, minio.GetObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to get file: %w", err)
	}
	defer obj.Close()

	// Check Parquet magic number (PAR1)
	magic := make([]byte, 4)
	if _, err := obj.Read(magic); err != nil {
		return fmt.Errorf("failed to read file header: %w", err)
	}

	if string(magic) != "PAR1" {
		return fmt.Errorf("invalid Parquet file: incorrect magic number")
	}

	return nil
}

// ValidateCSV validates a CSV file
func (s *MinioFileService) ValidateCSV(fileID string) error {
	ctx := context.Background()

	// Get object for reading
	obj, err := s.client.GetObject(ctx, s.bucketName, fileID, minio.GetObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to get file: %w", err)
	}
	defer obj.Close()

	// Read first chunk to validate
	buf := make([]byte, 1024)
	n, err := obj.Read(buf)
	if err != nil && err != io.EOF {
		return fmt.Errorf("failed to read file: %w", err)
	}

	if n == 0 {
		return fmt.Errorf("empty CSV file")
	}

	// Check for common CSV patterns
	content := string(buf[:n])
	if !strings.Contains(content, ",") && !strings.Contains(content, ";") && !strings.Contains(content, "\t") {
		logger.Warn("CSV file may not have proper delimiters: %s", fileID)
	}

	return nil
}

// HealthCheck checks MinIO connectivity
func (s *MinioFileService) HealthCheck() error {
	ctx := context.Background()
	_, err := s.client.BucketExists(ctx, s.bucketName)
	if err != nil {
		return fmt.Errorf("MinIO health check failed: %w", err)
	}
	return nil
}

// ReadFileContent reads the entire file content (for processing)
func (s *MinioFileService) ReadFileContent(fileID string) ([]byte, *FileInfo, error) {
	reader, fileInfo, err := s.Get(fileID)
	if err != nil {
		return nil, nil, err
	}
	defer reader.Close()

	content, err := io.ReadAll(reader)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read file content: %w", err)
	}

	return content, fileInfo, nil
}

// getContentType determines MIME type from filename
func getContentType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".csv":
		return "text/csv"
	case ".parquet":
		return "application/vnd.apache.parquet"
	case ".json":
		return "application/json"
	case ".txt":
		return "text/plain"
	case ".xlsx":
		return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	case ".xls":
		return "application/vnd.ms-excel"
	default:
		return "application/octet-stream"
	}
}

// SupportedFormats returns list of supported file formats
func SupportedFormats() []string {
	return []string{"csv", "parquet", "json"}
}

// IsSupported checks if a file format is supported
func IsSupported(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".csv", ".parquet", ".json":
		return true
	default:
		return false
	}
}

// SaveFromBytes is a helper to save from byte slice
func (s *MinioFileService) SaveFromBytes(subPath, filename string, data []byte) (*FileInfo, error) {
	reader := bytes.NewReader(data)
	return s.SaveWithPath(subPath, filename, reader, int64(len(data)))
}
