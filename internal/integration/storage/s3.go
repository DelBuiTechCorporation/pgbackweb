package storage

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	awscred "github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/eduardolat/pgbackweb/internal/util/strutil"
	minio "github.com/minio/minio-go/v7"
	miniocred "github.com/minio/minio-go/v7/pkg/credentials"
)

const (
	s3SignatureV2 = "v2"
	s3SignatureV4 = "v4"
)

// createS3Client creates a new S3 client
func createS3ClientV4(
	accessKey, secretKey, region, endpoint string,
	forcePathStyle bool,
) (*s3.Client, error) {
	normalizedEndpoint, err := normalizeS3Endpoint(endpoint)
	if err != nil {
		return nil, err
	}

	credentialsProvider := awscred.NewStaticCredentialsProvider(
		accessKey, secretKey, "",
	)

	//nolint:all
	endpointResolver := aws.EndpointResolverFunc(func(
		_ string, _ string,
	) (aws.Endpoint, error) {
		return aws.Endpoint{
			HostnameImmutable: true,
			URL:               normalizedEndpoint,
		}, nil
	})

	normalizedRegion := normalizeS3Region(region)

	//nolint:all
	conf, err := config.LoadDefaultConfig(
		context.TODO(),
		config.WithRegion(normalizedRegion),
		config.WithEndpointResolver(endpointResolver),
		config.WithCredentialsProvider(credentialsProvider),
	)
	if err != nil {
		return nil, fmt.Errorf("error initializing storage config: %w", err)
	}

	s3Client := s3.NewFromConfig(conf, func(o *s3.Options) {
		o.UsePathStyle = forcePathStyle
	})
	return s3Client, nil
}

func createS3ClientV2(
	accessKey, secretKey, region, endpoint string,
	forcePathStyle bool,
) (*minio.Client, error) {
	normalizedEndpoint, err := normalizeS3Endpoint(endpoint)
	if err != nil {
		return nil, err
	}

	parsedURL, err := url.Parse(normalizedEndpoint)
	if err != nil {
		return nil, fmt.Errorf("invalid S3 endpoint: %w", err)
	}

	opts := &minio.Options{
		Creds:  miniocred.NewStaticV2(accessKey, secretKey, ""),
		Secure: parsedURL.Scheme == "https",
		Region: normalizeS3Region(region),
	}
	if forcePathStyle {
		opts.BucketLookup = minio.BucketLookupPath
	}

	client, err := minio.New(parsedURL.Host, opts)
	if err != nil {
		return nil, fmt.Errorf("error initializing storage config: %w", err)
	}

	return client, nil
}

func normalizeS3Endpoint(endpoint string) (string, error) {
	cleaned := strings.TrimSpace(endpoint)
	if cleaned == "" {
		return "", fmt.Errorf("invalid S3 endpoint: endpoint is empty")
	}

	if !strings.Contains(cleaned, "://") {
		cleaned = "https://" + cleaned
	}

	parsedURL, err := url.Parse(cleaned)
	if err != nil {
		return "", fmt.Errorf("invalid S3 endpoint: %w", err)
	}
	if parsedURL.Scheme == "" || parsedURL.Host == "" {
		return "", fmt.Errorf("invalid S3 endpoint: expected host and scheme")
	}

	return parsedURL.String(), nil
}

func normalizeS3Region(region string) string {
	cleaned := strings.TrimSpace(region)
	if cleaned == "" || strings.EqualFold(cleaned, "auto") {
		return "us-east-1"
	}

	return cleaned
}

func normalizeS3SignatureVersion(signatureVersion string) string {
	if strings.EqualFold(strings.TrimSpace(signatureVersion), s3SignatureV2) {
		return s3SignatureV2
	}

	return s3SignatureV4
}

// S3Test tests the connection to S3
func (Client) S3Test(
	accessKey, secretKey, region, endpoint, bucketName string,
	forcePathStyle bool,
	signatureVersion string,
) error {
	if normalizeS3SignatureVersion(signatureVersion) == s3SignatureV2 {
		s3Client, err := createS3ClientV2(
			accessKey, secretKey, region, endpoint, forcePathStyle,
		)
		if err != nil {
			return err
		}

		exists, err := s3Client.BucketExists(context.TODO(), bucketName)
		if err != nil {
			return fmt.Errorf("failed to test S3 bucket: %w", err)
		}
		if !exists {
			return fmt.Errorf("failed to test S3 bucket: bucket does not exist")
		}
		return nil
	}

	s3Client, err := createS3ClientV4(
		accessKey, secretKey, region, endpoint, forcePathStyle,
	)
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
}

// S3Upload uploads a file to S3 from a reader.
//
// Returns the file size, in bytes.
func (Client) S3Upload(
	accessKey, secretKey, region, endpoint, bucketName, key string,
	forcePathStyle bool,
	signatureVersion string,
	fileReader io.Reader,
) (int64, error) {
	key = strutil.RemoveLeadingSlash(key)
	contentType := strutil.GetContentTypeFromFileName(key)

	if normalizeS3SignatureVersion(signatureVersion) == s3SignatureV2 {
		s3Client, err := createS3ClientV2(
			accessKey, secretKey, region, endpoint, forcePathStyle,
		)
		if err != nil {
			return 0, err
		}

		uploadInfo, err := s3Client.PutObject(
			context.TODO(),
			bucketName,
			key,
			fileReader,
			-1,
			minio.PutObjectOptions{ContentType: contentType},
		)
		if err != nil {
			return 0, fmt.Errorf("failed to upload file to S3: %w", err)
		}

		return uploadInfo.Size, nil
	}

	s3Client, err := createS3ClientV4(
		accessKey, secretKey, region, endpoint, forcePathStyle,
	)
	if err != nil {
		return 0, err
	}

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
}

// S3Delete deletes a file from S3
func (Client) S3Delete(
	accessKey, secretKey, region, endpoint, bucketName, key string,
	forcePathStyle bool,
	signatureVersion string,
) error {
	key = strutil.RemoveLeadingSlash(key)

	if normalizeS3SignatureVersion(signatureVersion) == s3SignatureV2 {
		s3Client, err := createS3ClientV2(
			accessKey, secretKey, region, endpoint, forcePathStyle,
		)
		if err != nil {
			return err
		}

		err = s3Client.RemoveObject(
			context.TODO(),
			bucketName,
			key,
			minio.RemoveObjectOptions{},
		)
		if err != nil {
			return fmt.Errorf("failed to delete file from S3: %w", err)
		}

		return nil
	}

	s3Client, err := createS3ClientV4(
		accessKey, secretKey, region, endpoint, forcePathStyle,
	)
	if err != nil {
		return err
	}

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
}

// S3GetDownloadLink generates a presigned URL for downloading a file from S3
func (Client) S3GetDownloadLink(
	accessKey, secretKey, region, endpoint, bucketName, key string,
	forcePathStyle bool,
	signatureVersion string,
	expiration time.Duration,
) (string, error) {
	if normalizeS3SignatureVersion(signatureVersion) == s3SignatureV2 {
		s3Client, err := createS3ClientV2(
			accessKey, secretKey, region, endpoint, forcePathStyle,
		)
		if err != nil {
			return "", fmt.Errorf("failed to create S3 client: %w", err)
		}

		presignedURL, err := s3Client.PresignedGetObject(
			context.TODO(),
			bucketName,
			strutil.RemoveLeadingSlash(key),
			expiration,
			nil,
		)
		if err != nil {
			return "", fmt.Errorf("failed to generate presigned URL: %w", err)
		}

		return presignedURL.String(), nil
	}

	s3Client, err := createS3ClientV4(
		accessKey, secretKey, region, endpoint, forcePathStyle,
	)
	if err != nil {
		return "", fmt.Errorf("failed to create S3 client: %w", err)
	}

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
}
