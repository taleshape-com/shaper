package snapshots

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

func newMinioClient(endpoint, region, accessKey, secretKey string) (*minio.Client, error) {
	if endpoint == "" {
		return nil, fmt.Errorf("S3 endpoint is required")
	}

	// Remove http:// or https:// prefix if present
	cleanEndpoint := strings.TrimPrefix(endpoint, "http://")
	cleanEndpoint = strings.TrimPrefix(cleanEndpoint, "https://")

	// Determine if SSL should be used based on the original endpoint
	useSSL := !strings.HasPrefix(endpoint, "http://")

	// Initialize MinIO client
	minioClient, err := minio.New(cleanEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
		Region: region,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create MinIO client: %w", err)
	}

	return minioClient, nil
}

func newS3Reader(ctx context.Context, bucket, key, endpoint, region, accessKey, secretKey string) (io.ReadCloser, error) {
	client, err := newMinioClient(endpoint, region, accessKey, secretKey)
	if err != nil {
		return nil, err
	}

	object, err := client.GetObject(ctx, bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get object from S3: %w", err)
	}

	return object, nil
}

func uploadFileToS3(ctx context.Context, filePath, bucket, key, endpoint, region, accessKey, secretKey string) error {
	client, err := newMinioClient(endpoint, region, accessKey, secretKey)
	if err != nil {
		return err
	}

	// Verify bucket exists
	exists, err := client.BucketExists(ctx, bucket)
	if err != nil {
		return fmt.Errorf("failed to check bucket %s: %w", bucket, err)
	}
	if !exists {
		return fmt.Errorf("bucket %s does not exist", bucket)
	}

	// Upload file directly
	_, err = client.FPutObject(ctx, bucket, key, filePath, minio.PutObjectOptions{
		ContentType: "application/octet-stream",
	})
	if err != nil {
		return fmt.Errorf("failed to upload file to S3: %w", err)
	}

	return nil
}
