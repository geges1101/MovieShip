package internal

import (
	"context"
	"mime/multipart"
	"net/url"
	"os"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type MinioClient struct {
	Client *minio.Client
	Bucket string
}

func NewMinioClient() (*MinioClient, error) {
	endpoint := os.Getenv("MINIO_ENDPOINT")
	accessKey := os.Getenv("MINIO_ACCESS_KEY")
	secretKey := os.Getenv("MINIO_SECRET_KEY")
	bucket := os.Getenv("MINIO_BUCKET")
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: false,
	})
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	exists, err := client.BucketExists(ctx, bucket)
	if err != nil {
		return nil, err
	}
	if !exists {
		err = client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{})
		if err != nil {
			return nil, err
		}
	}
	return &MinioClient{Client: client, Bucket: bucket}, nil
}

func (m *MinioClient) UploadVideo(file multipart.File, fileHeader *multipart.FileHeader, objectName string) error {
	_, err := m.Client.PutObject(context.Background(), m.Bucket, objectName, file, fileHeader.Size, minio.PutObjectOptions{ContentType: fileHeader.Header.Get("Content-Type")})
	return err
}

func (m *MinioClient) GetPresignedURL(objectName string, expiry time.Duration) (string, error) {
	reqParams := make(url.Values)
	urlObj, err := m.Client.PresignedGetObject(context.Background(), m.Bucket, objectName, expiry, reqParams)
	if err != nil {
		return "", err
	}
	return urlObj.String(), nil
}
