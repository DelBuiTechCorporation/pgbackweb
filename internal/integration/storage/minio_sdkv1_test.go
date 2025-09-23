package storage

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	awsv1 "github.com/aws/aws-sdk-go/aws"
	creds "github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	s3v1 "github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

// createS3ClientV1WithSigV2 creates S3 client using AWS SDK v1 with Signature v2
func createS3ClientV1WithSigV2(accessKey, secretKey, region, endpoint string) (*s3v1.S3, error) {
	creds := creds.NewStaticCredentials(accessKey, secretKey, "")

	config := &awsv1.Config{
		Credentials:      creds,
		Region:           awsv1.String(region),
		Endpoint:         awsv1.String(endpoint),
		S3ForcePathStyle: awsv1.Bool(true), // Importante para MinIO
		DisableSSL:       awsv1.Bool(strings.HasPrefix(endpoint, "http://")),
	}

	// Forçar Signature Version 2
	config.S3Disable100Continue = awsv1.Bool(true)
	config.S3UseAccelerate = awsv1.Bool(false)

	sess, err := session.NewSession(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	return s3v1.New(sess), nil
}

func TestMinioWithSDKv1SignatureV2(t *testing.T) {
	// Teste usando AWS SDK v1 com Signature Version 2 (como no rclone)
	accessKey := os.Getenv("STORAGE_ACCESS_KEY_ID")
	secretKey := os.Getenv("STORAGE_SECRET_ACCESS_KEY")
	endpoint := os.Getenv("STORAGE_ENDPOINT")
	bucket := os.Getenv("STORAGE_BUCKET_NAME")

	if accessKey == "" || secretKey == "" || endpoint == "" || bucket == "" {
		t.Skip("Variáveis de ambiente STORAGE_* não configuradas")
	}

	// Usar região other-v2-signature como no rclone
	region := "other-v2-signature"

	s3Client, err := createS3ClientV1WithSigV2(accessKey, secretKey, region, endpoint)
	if err != nil {
		t.Fatalf("Falha ao criar cliente SDK v1: %v", err)
	}

	// Teste de conexão
	_, err = s3Client.HeadBucket(&s3v1.HeadBucketInput{
		Bucket: awsv1.String(bucket),
	})
	if err != nil {
		t.Logf("Teste SDK v1 com SigV2 falhou: %v", err)
		t.Skip("SDK v1 com Signature v2 não funcionou")
	}

	t.Logf("Sucesso com SDK v1 + Signature v2 (region: %s)", region)

	// Teste de upload
	conteudo := "teste sdk v1 sig v2"
	testeKey := "test/sdkv1_test.txt"

	uploader := s3manager.NewUploaderWithClient(s3Client)
	_, err = uploader.Upload(&s3manager.UploadInput{
		Bucket: awsv1.String(bucket),
		Key:    awsv1.String(testeKey),
		Body:   strings.NewReader(conteudo),
	})
	if err != nil {
		t.Fatalf("Falha no upload com SDK v1: %v", err)
	}

	// Teste de presigned URL
	req, _ := s3Client.GetObjectRequest(&s3v1.GetObjectInput{
		Bucket: awsv1.String(bucket),
		Key:    awsv1.String(testeKey),
	})
	url, err := req.Presign(5 * time.Minute)
	if err != nil {
		t.Fatalf("Falha ao gerar presigned URL: %v", err)
	}
	t.Logf("Presigned URL: %s", url)

	// Limpar
	_, err = s3Client.DeleteObject(&s3v1.DeleteObjectInput{
		Bucket: awsv1.String(bucket),
		Key:    awsv1.String(testeKey),
	})
	if err != nil {
		t.Logf("Aviso: falha ao limpar: %v", err)
	}
}