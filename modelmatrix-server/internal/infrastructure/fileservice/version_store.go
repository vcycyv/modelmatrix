package fileservice

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"strings"

	"modelmatrix-server/pkg/logger"

	"github.com/minio/minio-go/v7"
)

// VersionStore provides content-addressable storage for model version files.
// Copy to versions/content/{hash} only if not already present (deduplication).
type VersionStore interface {
	// EnsureVersionedCopy copies the object at sourcePath to versions/content/{contentHash}{ext}.
	// If contentHash is empty, it is computed from the source content.
	// If the version key already exists, no copy is performed (dedup).
	// Returns the full path minio://bucket/versions/content/{hash}{ext}.
	EnsureVersionedCopy(sourcePath, contentHash, fileExt string) (versionPath string, err error)
}

const versionContentPrefix = "versions/content/"

// EnsureVersionedCopy implements VersionStore for MinIO.
func (s *MinioFileService) EnsureVersionedCopy(sourcePath, contentHash, fileExt string) (versionPath string, err error) {
	ctx := context.Background()
	sourceKey := objectKeyFromPath(sourcePath, s.bucketName)
	if sourceKey == "" {
		return "", fmt.Errorf("invalid source path: %s", sourcePath)
	}

	// Normalize file extension (e.g. ".pkl" or "pkl" -> ".pkl")
	if fileExt != "" && !strings.HasPrefix(fileExt, ".") {
		fileExt = "." + fileExt
	}

	obj, err := s.client.GetObject(ctx, s.bucketName, sourceKey, minio.GetObjectOptions{})
	if err != nil {
		return "", fmt.Errorf("get source object: %w", err)
	}
	defer obj.Close()

	data, err := io.ReadAll(obj)
	if err != nil {
		return "", fmt.Errorf("read source object: %w", err)
	}

	if contentHash == "" {
		h := sha256.Sum256(data)
		contentHash = hex.EncodeToString(h[:])
	}

	versionKey := versionContentPrefix + contentHash + fileExt

	// Check if already exists (copy-if-not-exists)
	_, err = s.client.StatObject(ctx, s.bucketName, versionKey, minio.StatObjectOptions{})
	if err == nil {
		// Already exists, return path
		return s.bucketToPath(versionKey), nil
	}
	errResp := minio.ToErrorResponse(err)
	if errResp.Code != "NoSuchKey" {
		return "", fmt.Errorf("stat version object: %w", err)
	}

	// Copy to version store (put-if-not-exists would require CopyObject with condition; we use PutObject)
	_, err = s.client.PutObject(ctx, s.bucketName, versionKey, io.NopCloser(bytes.NewReader(data)), int64(len(data)), minio.PutObjectOptions{
		ContentType: "application/octet-stream",
	})
	if err != nil {
		return "", fmt.Errorf("put version object: %w", err)
	}
	logger.Info("Stored versioned file: %s (hash=%s)", versionKey, contentHash)
	return s.bucketToPath(versionKey), nil
}

// objectKeyFromPath extracts the object key from a path.
// Supports "minio://bucket/key" or plain "key".
func objectKeyFromPath(path, bucket string) string {
	path = strings.TrimSpace(path)
	if strings.HasPrefix(path, "minio://") {
		parts := strings.SplitN(path, "/", 4)
		if len(parts) >= 4 {
			return parts[3]
		}
		if len(parts) == 3 {
			return ""
		}
	}
	return path
}

func (s *MinioFileService) bucketToPath(objectKey string) string {
	return "minio://" + s.bucketName + "/" + objectKey
}
