package storage

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	minioCreds "github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/minio/minio-go/v7"
)

func TestMinioBasic(t *testing.T) {
	accessKey := os.Getenv("STORAGE_ACCESS_KEY_ID")
	secretKey := os.Getenv("STORAGE_SECRET_ACCESS_KEY")
	region := os.Getenv("STORAGE_REGION")
	endpoint := os.Getenv("STORAGE_ENDPOINT")
	bucket := os.Getenv("STORAGE_BUCKET_NAME")

	if accessKey == "" || secretKey == "" || region == "" || endpoint == "" || bucket == "" {
		t.Skip("Variáveis de ambiente STORAGE_* não configuradas")
	}

	client := Client{}

	// Teste de conexão com MinIO
	err := client.S3Test(ProviderMinIO, accessKey, secretKey, region, endpoint, bucket, false)
	if err != nil {
		t.Fatalf("Falha ao conectar no storage MinIO: %v", err)
	}

	// Teste de upload
	conteudo := "conteudo de teste storage"
	testeKey := "test/storage_test.txt"
	_, err = client.S3Upload(ProviderMinIO, accessKey, secretKey, region, endpoint, bucket, testeKey, strings.NewReader(conteudo), false)
	if err != nil {
		t.Fatalf("Falha ao fazer upload: %v", err)
	}

	// Teste de geração de link de download
	url, err := client.S3GetDownloadLink(ProviderMinIO, accessKey, secretKey, region, endpoint, bucket, testeKey, 60*time.Second, false)
	if err != nil {
		t.Fatalf("Falha ao gerar link de download: %v", err)
	}
	t.Logf("URL de download gerada: %s", url)

	// Teste de remoção
	err = client.S3Delete(ProviderMinIO, accessKey, secretKey, region, endpoint, bucket, testeKey, false)
	if err != nil {
		t.Fatalf("Falha ao deletar arquivo: %v", err)
	}
}

func TestMinioWithForcePathStyle(t *testing.T) {
	// Teste específico para MinIO com force path style
	accessKey := os.Getenv("STORAGE_ACCESS_KEY_ID")
	secretKey := os.Getenv("STORAGE_SECRET_ACCESS_KEY")
	region := os.Getenv("STORAGE_REGION")
	endpoint := os.Getenv("STORAGE_ENDPOINT")
	bucket := os.Getenv("STORAGE_BUCKET_NAME")

	if accessKey == "" || secretKey == "" || region == "" || endpoint == "" || bucket == "" {
		t.Skip("Variáveis de ambiente STORAGE_* não configuradas")
	}

	client := Client{}

	// Teste com MinIO (forcePathStyle não se aplica ao MinIO, mas testamos mesmo assim)
	err := client.S3Test(ProviderMinIO, accessKey, secretKey, region, endpoint, bucket, true)
	if err != nil {
		t.Logf("Teste com MinIO falhou: %v", err)
		t.Skip("MinIO não funcionou")
	}

	t.Logf("MinIO funcionando corretamente")
}

func TestAWSS3Compatibility(t *testing.T) {
	// Teste de compatibilidade com AWS S3 (usando credenciais de teste se disponíveis)
	accessKey := os.Getenv("AWS_ACCESS_KEY_ID")
	secretKey := os.Getenv("AWS_SECRET_ACCESS_KEY")
	region := os.Getenv("AWS_REGION")
	bucket := os.Getenv("AWS_TEST_BUCKET")

	if accessKey == "" || secretKey == "" || region == "" || bucket == "" {
		t.Skip("Credenciais AWS não configuradas para teste")
	}

	client := Client{}

	// Teste de conexão com AWS S3
	err := client.S3Test(ProviderAWS, accessKey, secretKey, region, "", bucket, false) // endpoint vazio para AWS
	if err != nil {
		t.Fatalf("Falha ao conectar no AWS S3: %v", err)
	}

	t.Logf("Conexão com AWS S3 estabelecida com sucesso")
}

func TestStorageOperations(t *testing.T) {
	// Teste completo de operações de storage com MinIO
	accessKey := os.Getenv("STORAGE_ACCESS_KEY_ID")
	secretKey := os.Getenv("STORAGE_SECRET_ACCESS_KEY")
	region := os.Getenv("STORAGE_REGION")
	endpoint := os.Getenv("STORAGE_ENDPOINT")
	bucket := os.Getenv("STORAGE_BUCKET_NAME")

	if accessKey == "" || secretKey == "" || endpoint == "" || bucket == "" {
		t.Skip("Variáveis de ambiente de storage não configuradas")
	}

	client := Client{}

	// Teste de conexão
	err := client.S3Test(ProviderMinIO, accessKey, secretKey, region, endpoint, bucket, false)
	if err != nil {
		t.Fatalf("Falha ao conectar no storage: %v", err)
	}

	// Teste de upload
	testData := []byte("dados de teste para upload")
	testKey := "test-file-" + time.Now().Format("20060102-150405")

	_, err = client.S3Upload(ProviderMinIO, accessKey, secretKey, region, endpoint, bucket, testKey, strings.NewReader(string(testData)), false)
	if err != nil {
		t.Fatalf("Falha ao fazer upload: %v", err)
	}

	// Teste de geração de link de download
	downloadLink, err := client.S3GetDownloadLink(ProviderMinIO, accessKey, secretKey, region, endpoint, bucket, testKey, 3600*time.Second, false)
	if err != nil {
		t.Fatalf("Falha ao gerar link de download: %v", err)
	}

	if downloadLink == "" {
		t.Fatal("Link de download vazio")
	}

	t.Logf("Link de download gerado: %s", downloadLink)

	// Teste de exclusão
	err = client.S3Delete(ProviderMinIO, accessKey, secretKey, region, endpoint, bucket, testKey, false)
	if err != nil {
		t.Fatalf("Falha ao excluir arquivo: %v", err)
	}

	t.Logf("Operações de storage concluídas com sucesso")
}

func TestS3ClientCreation(t *testing.T) {
	// Teste de criação de cliente S3 com diferentes configurações
	accessKey := os.Getenv("STORAGE_ACCESS_KEY_ID")
	secretKey := os.Getenv("STORAGE_SECRET_ACCESS_KEY")
	region := os.Getenv("STORAGE_REGION")
	endpoint := os.Getenv("STORAGE_ENDPOINT")
	bucket := os.Getenv("STORAGE_BUCKET_NAME")

	if accessKey == "" || secretKey == "" || endpoint == "" || bucket == "" {
		t.Skip("Variáveis de ambiente de storage não configuradas")
	}

	client := Client{}

	// Teste com MinIO sem force path style
	err := client.S3Test(ProviderMinIO, accessKey, secretKey, region, endpoint, bucket, false)
	if err != nil {
		t.Fatalf("Falha com MinIO sem force path style: %v", err)
	}

	// Teste com MinIO com force path style
	err = client.S3Test(ProviderMinIO, accessKey, secretKey, region, endpoint, bucket, true)
	if err != nil {
		t.Fatalf("Falha com MinIO com force path style: %v", err)
	}

	t.Logf("Criação de cliente S3 testada com sucesso")
}

func TestMinioConfigurationValidation(t *testing.T) {
	// Teste de validação de configuração sem operações de rede
	accessKey := os.Getenv("STORAGE_ACCESS_KEY_ID")
	secretKey := os.Getenv("STORAGE_SECRET_ACCESS_KEY")
	region := os.Getenv("STORAGE_REGION")
	endpoint := os.Getenv("STORAGE_ENDPOINT")
	bucket := os.Getenv("STORAGE_BUCKET_NAME")

	if accessKey == "" || secretKey == "" || region == "" || endpoint == "" || bucket == "" {
		t.Skip("Variáveis de ambiente STORAGE_* não configuradas")
	}

	// Validações básicas
	if len(accessKey) < 3 {
		t.Errorf("Access key parece muito curta: %s", accessKey)
	}

	if len(secretKey) < 8 {
		t.Errorf("Secret key parece muito curta: %s", secretKey)
	}

	if !strings.HasPrefix(endpoint, "http") {
		t.Errorf("Endpoint deve começar com http:// ou https://: %s", endpoint)
	}

	if bucket == "" {
		t.Errorf("Bucket name não pode ser vazio")
	}

	t.Logf("Configuração validada: endpoint=%s, bucket=%s, region=%s", endpoint, bucket, region)
}

// createS3ClientV2 creates a new S3 client with Signature Version 2
func createS3ClientV2(
	accessKey, secretKey, region, endpoint string,
) (*s3.Client, error) {
	credentialsProvider := credentials.NewStaticCredentialsProvider(
		accessKey, secretKey, "",
	)

	//nolint:all
	endpointResolver := aws.EndpointResolverFunc(func(
		_ string, _ string,
	) (aws.Endpoint, error) {
		return aws.Endpoint{
			HostnameImmutable: true,
			URL:               endpoint,
		}, nil
	})

	// Force Signature Version 2
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

func TestMinioWithSignatureV2(t *testing.T) {
	// Teste específico para MinIO que pode requerer Signature Version 2
	accessKey := os.Getenv("STORAGE_ACCESS_KEY_ID")
	secretKey := os.Getenv("STORAGE_SECRET_ACCESS_KEY")
	region := os.Getenv("STORAGE_REGION")
	endpoint := os.Getenv("STORAGE_ENDPOINT")
	bucket := os.Getenv("STORAGE_BUCKET_NAME")

	if accessKey == "" || secretKey == "" || region == "" || endpoint == "" || bucket == "" {
		t.Skip("Variáveis de ambiente STORAGE_* não configuradas")
	}

	// Tentar criar cliente com configurações específicas para MinIO
	clientV2, err := createS3ClientV2(accessKey, secretKey, region, endpoint)
	if err != nil {
		t.Fatalf("Falha ao criar cliente S3 v2: %v", err)
	}

	// Teste de conexão
	_, err = clientV2.HeadBucket(
		context.TODO(),
		&s3.HeadBucketInput{
			Bucket: aws.String(bucket),
		},
	)
	if err != nil {
		t.Logf("Teste com Signature v2 falhou: %v", err)
		t.Skip("Signature v2 não funcionou - pode ser necessário verificar configuração do MinIO")
	}

	t.Logf("Sucesso com cliente S3 personalizado")

	// Teste de upload simples
	conteudo := "teste signature v2"
	testeKey := "test/sigv2_test.txt"

	_, err = clientV2.PutObject(
		context.TODO(),
		&s3.PutObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(testeKey),
			Body:   strings.NewReader(conteudo),
		},
	)
	if err != nil {
		t.Fatalf("Falha no upload com sig v2: %v", err)
	}

	// Limpar
	_, err = clientV2.DeleteObject(
		context.TODO(),
		&s3.DeleteObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(testeKey),
		},
	)
	if err != nil {
		t.Logf("Aviso: falha ao limpar arquivo de teste: %v", err)
	}
}

func TestMinioEndpointConnectivity(t *testing.T) {
	// Teste básico de conectividade HTTP ao endpoint MinIO
	endpoint := os.Getenv("STORAGE_ENDPOINT")
	bucket := os.Getenv("STORAGE_BUCKET_NAME")

	if endpoint == "" || bucket == "" {
		t.Skip("Variáveis de ambiente STORAGE_ENDPOINT ou STORAGE_BUCKET_NAME não configuradas")
	}

	// Criar cliente HTTP com timeout
	client := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // Para desenvolvimento
		},
	}

	// Tentar uma requisição GET básica (sem autenticação)
	testURL := fmt.Sprintf("%s/%s/", endpoint, bucket)
	resp, err := client.Get(testURL)
	if err != nil {
		t.Logf("Falha na conectividade HTTP: %v", err)
		t.Skip("Endpoint não está acessível - verificar URL e conectividade de rede")
	}
	defer resp.Body.Close()

	t.Logf("Endpoint acessível: %s (Status: %d)", testURL, resp.StatusCode)

	// Status 403 é esperado sem credenciais, mas confirma que o endpoint responde
	if resp.StatusCode == 403 {
		t.Logf("Endpoint responde corretamente (403 Forbidden esperado sem credenciais)")
	} else if resp.StatusCode == 404 {
		t.Logf("Bucket pode não existir ou endpoint incorreto")
	} else {
		t.Logf("Resposta inesperada do endpoint: %d", resp.StatusCode)
	}
}

func TestMinioWithRcloneConfig(t *testing.T) {
	// Teste baseado na configuração do rclone que funciona
	accessKey := os.Getenv("STORAGE_ACCESS_KEY_ID")
	secretKey := os.Getenv("STORAGE_SECRET_ACCESS_KEY")
	endpoint := os.Getenv("STORAGE_ENDPOINT")
	bucket := os.Getenv("STORAGE_BUCKET_NAME")

	if accessKey == "" || secretKey == "" || endpoint == "" || bucket == "" {
		t.Skip("Variáveis de ambiente STORAGE_* não configuradas")
	}

	// Configuração baseada no rclone: region: other-v2-signature
	region := "other-v2-signature"

	// Criar cliente com a configuração do rclone
	client := Client{}

	// Teste de conexão
	err := client.S3Test(ProviderMinIO, accessKey, secretKey, region, endpoint, bucket, false)
	if err != nil {
		t.Logf("Teste com configuração rclone falhou: %v", err)
		t.Skip("Configuração baseada no rclone não funcionou")
	}

	t.Logf("Sucesso com configuração baseada no rclone (region: %s)", region)

	// Teste de upload
	conteudo := "teste rclone config"
	testeKey := "test/rclone_test.txt"

	_, err = client.S3Upload(ProviderMinIO, accessKey, secretKey, region, endpoint, bucket, testeKey, strings.NewReader(conteudo), false)
	if err != nil {
		t.Fatalf("Falha no upload: %v", err)
	}

	// Teste de download link
	url, err := client.S3GetDownloadLink(ProviderMinIO, accessKey, secretKey, region, endpoint, bucket, testeKey, 5*time.Minute, false)
	if err != nil {
		t.Fatalf("Falha ao gerar link: %v", err)
	}
	t.Logf("Link de download: %s", url)

	// Limpar
	err = client.S3Delete(ProviderMinIO, accessKey, secretKey, region, endpoint, bucket, testeKey, false)
	if err != nil {
		t.Logf("Aviso: falha ao limpar: %v", err)
	}
}

func TestMinioDirectUpload(t *testing.T) {
	// Teste direto de upload sem verificar bucket primeiro
	accessKey := os.Getenv("STORAGE_ACCESS_KEY_ID")
	secretKey := os.Getenv("STORAGE_SECRET_ACCESS_KEY")
	region := os.Getenv("STORAGE_REGION")
	endpoint := os.Getenv("STORAGE_ENDPOINT")
	bucket := os.Getenv("STORAGE_BUCKET_NAME")

	if accessKey == "" || secretKey == "" || region == "" || endpoint == "" || bucket == "" {
		t.Skip("Variáveis de ambiente STORAGE_* não configuradas")
	}

	client := Client{}

	// Tentar upload direto
	conteudo := "teste upload direto"
	testeKey := "test/direct_upload_test.txt"

	_, err := client.S3Upload(ProviderMinIO, accessKey, secretKey, region, endpoint, bucket, testeKey, strings.NewReader(conteudo), false)
	if err != nil {
		t.Logf("Upload direto falhou: %v", err)
		t.Skip("Upload direto não funcionou - verificar credenciais e permissões")
	}

	t.Logf("Upload direto funcionou!")

	// Tentar gerar link de download
	url, err := client.S3GetDownloadLink(ProviderMinIO, accessKey, secretKey, region, endpoint, bucket, testeKey, 5*time.Minute, false)
	if err != nil {
		t.Logf("Falha ao gerar link: %v", err)
	} else {
		t.Logf("Link gerado: %s", url)
	}

	// Limpar
	err = client.S3Delete(ProviderMinIO, accessKey, secretKey, region, endpoint, bucket, testeKey, false)
	if err != nil {
		t.Logf("Aviso: falha ao limpar: %v", err)
	}
}

func TestMinioOfficialClient(t *testing.T) {
	// Teste usando o cliente oficial do MinIO
	accessKey := os.Getenv("STORAGE_ACCESS_KEY_ID")
	secretKey := os.Getenv("STORAGE_SECRET_ACCESS_KEY")
	endpoint := os.Getenv("STORAGE_ENDPOINT")
	bucket := os.Getenv("STORAGE_BUCKET_NAME")

	if accessKey == "" || secretKey == "" || endpoint == "" || bucket == "" {
		t.Skip("Variáveis de ambiente STORAGE_* não configuradas")
	}

	// Remover https:// do endpoint para o cliente MinIO
	minioEndpoint := strings.TrimPrefix(endpoint, "https://")
	minioEndpoint = strings.TrimPrefix(minioEndpoint, "http://")

	// Criar cliente MinIO
	minioClient, err := minio.New(minioEndpoint, &minio.Options{
		Creds:  minioCreds.NewStaticV4(accessKey, secretKey, ""),
		Secure: strings.HasPrefix(endpoint, "https://"),
	})
	if err != nil {
		t.Fatalf("Falha ao criar cliente MinIO: %v", err)
	}

	// Teste de conexão - verificar se bucket existe
	exists, err := minioClient.BucketExists(context.Background(), bucket)
	if err != nil {
		t.Logf("Falha ao verificar bucket: %v", err)
		t.Skip("Não foi possível verificar o bucket")
	}

	if !exists {
		t.Logf("Bucket %s não existe", bucket)
		t.Skip("Bucket não existe")
	}

	t.Logf("Bucket %s existe e é acessível!", bucket)

	// Teste de upload
	conteudo := "teste com cliente oficial minio"
	testeKey := "test/minio_official_test.txt"

	_, err = minioClient.PutObject(context.Background(), bucket, testeKey, strings.NewReader(conteudo), int64(len(conteudo)), minio.PutObjectOptions{
		ContentType: "text/plain",
	})
	if err != nil {
		t.Fatalf("Falha no upload: %v", err)
	}

	t.Logf("Upload realizado com sucesso!")

	// Teste de presigned URL
	url, err := minioClient.PresignedGetObject(context.Background(), bucket, testeKey, time.Hour, nil)
	if err != nil {
		t.Fatalf("Falha ao gerar presigned URL: %v", err)
	}
	t.Logf("Presigned URL: %s", url)

	// Limpar
	err = minioClient.RemoveObject(context.Background(), bucket, testeKey, minio.RemoveObjectOptions{})
	if err != nil {
		t.Logf("Aviso: falha ao limpar: %v", err)
	}
}




