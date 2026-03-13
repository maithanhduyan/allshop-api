package storage

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type Storage struct {
	client    *minio.Client
	bucket    string
	publicURL string
}

func New(endpoint, accessKey, secretKey, bucket, publicURL string) (*Storage, error) {
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: false,
	})
	if err != nil {
		return nil, fmt.Errorf("init minio client: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	exists, err := client.BucketExists(ctx, bucket)
	if err != nil {
		return nil, fmt.Errorf("check bucket: %w", err)
	}

	if !exists {
		if err := client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{}); err != nil {
			return nil, fmt.Errorf("create bucket: %w", err)
		}

		// Set bucket policy to allow public read
		policy := fmt.Sprintf(`{
			"Version": "2012-10-17",
			"Statement": [{
				"Effect": "Allow",
				"Principal": {"AWS": ["*"]},
				"Action": ["s3:GetObject"],
				"Resource": ["arn:aws:s3:::%s/*"]
			}]
		}`, bucket)

		if err := client.SetBucketPolicy(ctx, bucket, policy); err != nil {
			return nil, fmt.Errorf("set bucket policy: %w", err)
		}

		log.Printf("Created bucket '%s' with public read policy", bucket)
	}

	log.Println("Connected to MinIO")
	return &Storage{client: client, bucket: bucket, publicURL: publicURL}, nil
}

// Upload uploads data to MinIO and returns the public URL
func (s *Storage) Upload(ctx context.Context, objectName string, reader io.Reader, size int64, contentType string) (string, error) {
	_, err := s.client.PutObject(ctx, s.bucket, objectName, reader, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return "", fmt.Errorf("upload object: %w", err)
	}

	return s.GetPublicURL(objectName), nil
}

// DownloadFromURL downloads an image from a URL and uploads it to MinIO
func (s *Storage) DownloadFromURL(ctx context.Context, imageURL, objectName string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, imageURL, nil)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("download image: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "image/jpeg"
	}

	return s.Upload(ctx, objectName, resp.Body, resp.ContentLength, contentType)
}

// GetPublicURL returns the public URL for an object
func (s *Storage) GetPublicURL(objectName string) string {
	return fmt.Sprintf("%s/%s/%s", s.publicURL, s.bucket, objectName)
}

// ServeImage serves an image from MinIO storage through the API
func (s *Storage) ServeImage(w http.ResponseWriter, r *http.Request, objectName string) {
	ctx := r.Context()

	obj, err := s.client.GetObject(ctx, s.bucket, objectName, minio.GetObjectOptions{})
	if err != nil {
		http.Error(w, "image not found", http.StatusNotFound)
		return
	}
	defer obj.Close()

	stat, err := obj.Stat()
	if err != nil {
		http.Error(w, "image not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", stat.ContentType)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", stat.Size))
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")

	io.Copy(w, obj)
}

// ObjectNameFromURL extracts an object name from a slug and image URL
func ObjectNameFromURL(slug, imageURL string) string {
	ext := path.Ext(imageURL)
	if ext == "" || len(ext) > 5 {
		ext = ".jpg"
	}
	// Clean any query params from extension
	if idx := strings.Index(ext, "?"); idx != -1 {
		ext = ext[:idx]
	}
	return fmt.Sprintf("products/%s%s", slug, ext)
}
