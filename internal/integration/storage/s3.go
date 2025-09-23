package storage

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/eduardolat/pgbackweb/internal/util/strutil"
	"github.com/minio/minio-go/v7"
	minioCreds "github.com/minio/minio-go/v7/pkg/credentials"
)

const (
	ProviderAWS   = "aws"
	ProviderMinIO = "minio"
)

// StorageConfig holds configuration for storage operations
type StorageConfig struct {
	Provider      string
	AccessKey     string
	SecretKey     string
	Region        string
	Endpoint      string
	BucketName    string
	ForcePathStyle bool
}

// createS3Client creates a new S3 client for AWS S3
func createS3Client(
	accessKey, secretKey, region, endpoint string, forcePathStyle bool,
) (*s3.Client, error) {
	credentialsProvider := credentials.NewStaticCredentialsProvider(
		accessKey, secretKey, "",
	)

	//nolint:all
	endpointResolver := aws.EndpointResolverFunc(func(
		_ string, _ string,
	) (aws.Endpoint, error) {
		return aws.Endpoint{
			HostnameImmutable: !forcePathStyle,
			URL:               endpoint,
		}, nil
	})

	//nolint:all
	conf, err := config.LoadDefaultConfig(
		context.TODO(),
		config.WithRegion(region),
		config.WithEndpointResolver(endpointResolver),
		config.WithCredentialsProvider(credentialsProvider),
	)
	if err != nil {
		return nil, fmt.Errorf("error initializing storage config: %w", err)
	}

	s3Client := s3.NewFromConfig(conf)
	return s3Client, nil
}

// createMinioClient creates a new MinIO client
func createMinioClient(
	accessKey, secretKey, endpoint string,
) (*minio.Client, error) {
	// Remover https:// do endpoint para o cliente MinIO
	minioEndpoint := strings.TrimPrefix(endpoint, "https://")
	minioEndpoint = strings.TrimPrefix(minioEndpoint, "http://")

	// Criar cliente MinIO
	minioClient, err := minio.New(minioEndpoint, &minio.Options{
		Creds:  minioCreds.NewStaticV4(accessKey, secretKey, ""),
		Secure: strings.HasPrefix(endpoint, "https://"),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create MinIO client: %w", err)
	}

	return minioClient, nil
}

// S3Test tests the connection to S3 or MinIO
func (Client) S3Test(
	provider string, accessKey, secretKey, region, endpoint, bucketName string, forcePathStyle bool,
) error {
	switch provider {
	case ProviderMinIO:
		minioClient, err := createMinioClient(accessKey, secretKey, endpoint)
		if err != nil {
			return err
		}

		exists, err := minioClient.BucketExists(context.Background(), bucketName)
		if err != nil {
			return fmt.Errorf("failed to test MinIO bucket: %w", err)
		}

		if !exists {
			return fmt.Errorf("bucket %s does not exist", bucketName)
		}

		return nil

	case ProviderAWS:
		s3Client, err := createS3Client(accessKey, secretKey, region, endpoint, forcePathStyle)
		if err != nil {
			return err
		}

		_, err = s3Client.HeadBucket(
			context.TODO(),
			&s3.HeadBucketInput{
				Bucket: aws.String(bucketName),
			},
		)
		if err != nil {
			return fmt.Errorf("failed to test S3 bucket: %w", err)
		}

		return nil

	default:
		return fmt.Errorf("unsupported provider: %s", provider)
	}
}

// S3Upload uploads a file to S3 or MinIO from a reader.
//
// Returns the file size, in bytes.
func (Client) S3Upload(
	provider string, accessKey, secretKey, region, endpoint, bucketName, key string, fileReader io.Reader, forcePathStyle bool,
) (int64, error) {
	switch provider {
	case ProviderMinIO:
		minioClient, err := createMinioClient(accessKey, secretKey, endpoint)
		if err != nil {
			return 0, err
		}

		key = strutil.RemoveLeadingSlash(key)
		contentType := strutil.GetContentTypeFromFileName(key)

		// Para MinIO, precisamos ler todo o conteúdo para saber o tamanho
		data, err := io.ReadAll(fileReader)
		if err != nil {
			return 0, fmt.Errorf("failed to read file content: %w", err)
		}

		_, err = minioClient.PutObject(context.Background(), bucketName, key, strings.NewReader(string(data)), int64(len(data)), minio.PutObjectOptions{
			ContentType: contentType,
		})
		if err != nil {
			return 0, fmt.Errorf("failed to upload file to MinIO: %w", err)
		}

		return int64(len(data)), nil

	case ProviderAWS:
		s3Client, err := createS3Client(accessKey, secretKey, region, endpoint, forcePathStyle)
		if err != nil {
			return 0, err
		}

		key = strutil.RemoveLeadingSlash(key)
		contentType := strutil.GetContentTypeFromFileName(key)

		uploader := manager.NewUploader(s3Client)
		_, err = uploader.Upload(
			context.TODO(),
			&s3.PutObjectInput{
				Bucket:      aws.String(bucketName),
				Key:         aws.String(key),
				Body:        fileReader,
				ContentType: aws.String(contentType),
			},
		)
		if err != nil {
			return 0, fmt.Errorf("failed to upload file to S3: %w", err)
		}

		fileHead, err := s3Client.HeadObject(
			context.TODO(),
			&s3.HeadObjectInput{
				Bucket: aws.String(bucketName),
				Key:    aws.String(key),
			},
		)
		if err != nil {
			return 0, fmt.Errorf("failed to get uploaded file info from S3: %w", err)
		}

		var fileSize int64
		if fileHead.ContentLength != nil {
			fileSize = *fileHead.ContentLength
		}

		return fileSize, nil

	default:
		return 0, fmt.Errorf("unsupported provider: %s", provider)
	}
}

// S3Delete deletes a file from S3 or MinIO
func (Client) S3Delete(
	provider string, accessKey, secretKey, region, endpoint, bucketName, key string, forcePathStyle bool,
) error {
	switch provider {
	case ProviderMinIO:
		minioClient, err := createMinioClient(accessKey, secretKey, endpoint)
		if err != nil {
			return err
		}

		key = strutil.RemoveLeadingSlash(key)

		err = minioClient.RemoveObject(context.Background(), bucketName, key, minio.RemoveObjectOptions{})
		if err != nil {
			return fmt.Errorf("failed to delete file from MinIO: %w", err)
		}

		return nil

	case ProviderAWS:
		s3Client, err := createS3Client(accessKey, secretKey, region, endpoint, forcePathStyle)
		if err != nil {
			return err
		}

		key = strutil.RemoveLeadingSlash(key)

		_, err = s3Client.DeleteObject(
			context.TODO(),
			&s3.DeleteObjectInput{
				Bucket: aws.String(bucketName),
				Key:    aws.String(key),
			},
		)
		if err != nil {
			return fmt.Errorf("failed to delete file from S3: %w", err)
		}

		return nil

	default:
		return fmt.Errorf("unsupported provider: %s", provider)
	}
}

// S3GetDownloadLink generates a presigned URL for downloading a file from S3 or MinIO
func (Client) S3GetDownloadLink(
	provider string, accessKey, secretKey, region, endpoint, bucketName, key string, expiration time.Duration, forcePathStyle bool,
) (string, error) {
	switch provider {
	case ProviderMinIO:
		minioClient, err := createMinioClient(accessKey, secretKey, endpoint)
		if err != nil {
			return "", fmt.Errorf("failed to create MinIO client: %w", err)
		}

		key = strutil.RemoveLeadingSlash(key)

		url, err := minioClient.PresignedGetObject(context.Background(), bucketName, key, expiration, nil)
		if err != nil {
			return "", fmt.Errorf("failed to generate presigned URL for MinIO: %w", err)
		}

		return url.String(), nil

	case ProviderAWS:
		s3Client, err := createS3Client(accessKey, secretKey, region, endpoint, forcePathStyle)
		if err != nil {
			return "", fmt.Errorf("failed to create S3 client: %w", err)
		}

		key = strutil.RemoveLeadingSlash(key)

		presigned, err := s3.NewPresignClient(s3Client).PresignGetObject(
			context.TODO(),
			&s3.GetObjectInput{
				Bucket: aws.String(bucketName),
				Key:    aws.String(key),
			},
			s3.WithPresignExpires(expiration),
		)
		if err != nil {
			return "", fmt.Errorf("failed to generate presigned URL: %w", err)
		}

		return presigned.URL, nil

	default:
		return "", fmt.Errorf("unsupported provider: %s", provider)
	}
}
