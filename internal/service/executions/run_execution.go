package executions

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/eduardolat/pgbackweb/internal/database/dbgen"
	"github.com/eduardolat/pgbackweb/internal/integration/postgres"
	"github.com/eduardolat/pgbackweb/internal/logger"
	"github.com/eduardolat/pgbackweb/internal/util/strutil"
	"github.com/eduardolat/pgbackweb/internal/util/timeutil"
	"github.com/google/uuid"
)

// RunExecution runs a backup execution
func (s *Service) RunExecution(ctx context.Context, backupID uuid.UUID) error {
	updateExec := func(params dbgen.ExecutionsServiceUpdateExecutionParams) error {
		if params.Status.String == "success" {
			s.webhooksService.RunExecutionSuccess(backupID)
		}

		if params.Status.String == "failed" {
			s.webhooksService.RunExecutionFailed(backupID)
		}

		_, err := s.dbgen.ExecutionsServiceUpdateExecution(
			ctx, params,
		)
		return err
	}

	logError := func(err error) {
		logger.Error("error running backup", logger.KV{
			"backup_id": backupID.String(),
			"error":     err.Error(),
		})
	}

	back, err := s.dbgen.ExecutionsServiceGetBackupData(
		ctx, dbgen.ExecutionsServiceGetBackupDataParams{
			BackupID:      backupID,
			EncryptionKey: s.env.PBW_ENCRYPTION_KEY,
		},
	)
	if err != nil {
		logError(err)
		return err
	}

	ex, err := s.CreateExecution(ctx, dbgen.ExecutionsServiceCreateExecutionParams{
		BackupID: backupID,
		Status:   "running",
	})
	if err != nil {
		logError(err)
		return err
	}

	if !back.BackupIsLocal {
		err = s.ints.StorageClient.S3Test(
			back.DecryptedDestinationAccessKey, back.DecryptedDestinationSecretKey,
			back.DestinationRegion.String, back.DestinationEndpoint.String,
			back.DestinationBucketName.String, back.DestinationForcePathStyle.Bool,
			back.DestinationSignatureVersion.String,
		)
		if err != nil {
			logError(err)
			return updateExec(dbgen.ExecutionsServiceUpdateExecutionParams{
				ID:         ex.ID,
				Status:     sql.NullString{Valid: true, String: "failed"},
				Message:    sql.NullString{Valid: true, String: err.Error()},
				FinishedAt: sql.NullTime{Valid: true, Time: time.Now()},
			})
		}
	}

	pgVersion, err := s.ints.PGClient.ParseVersion(back.DatabasePgVersion)
	if err != nil {
		logError(err)
		return updateExec(dbgen.ExecutionsServiceUpdateExecutionParams{
			ID:         ex.ID,
			Status:     sql.NullString{Valid: true, String: "failed"},
			Message:    sql.NullString{Valid: true, String: err.Error()},
			FinishedAt: sql.NullTime{Valid: true, Time: time.Now()},
		})
	}

	err = s.ints.PGClient.Test(pgVersion, back.DecryptedDatabaseConnectionString)
	if err != nil {
		logError(err)
		return updateExec(dbgen.ExecutionsServiceUpdateExecutionParams{
			ID:         ex.ID,
			Status:     sql.NullString{Valid: true, String: "failed"},
			Message:    sql.NullString{Valid: true, String: err.Error()},
			FinishedAt: sql.NullTime{Valid: true, Time: time.Now()},
		})
	}

	dumpParams := postgres.DumpParams{
		DataOnly:   back.BackupOptDataOnly,
		SchemaOnly: back.BackupOptSchemaOnly,
		Clean:      back.BackupOptClean,
		IfExists:   back.BackupOptIfExists,
		Create:     back.BackupOptCreate,
		NoComments: back.BackupOptNoComments,
	}

	compressionLevel := 9 // default: best compression
	if back.BackupCompressionLevel.Valid {
		compressionLevel = int(back.BackupCompressionLevel.Int16)
	}

	var parts []postgres.ZipPart
	var tempDir string
	var dumpErr error

	if back.BackupMaxPartSizeMb.Valid && back.BackupMaxPartSizeMb.Int32 > 0 {
		maxSize := int64(back.BackupMaxPartSizeMb.Int32) * 1024 * 1024
		parts, tempDir, dumpErr = s.ints.PGClient.DumpZipParts(
			pgVersion, back.DecryptedDatabaseConnectionString,
			maxSize, compressionLevel, dumpParams,
		)
	} else {
		var dir string
		dir, dumpErr = os.MkdirTemp("", "pbw-single-*")
		if dumpErr == nil {
			tempDir = dir
			filePath := strutil.CreatePath(true, dir, "dump.zip")
			var f *os.File
			f, dumpErr = os.Create(filePath)
			if dumpErr == nil {
				reader := s.ints.PGClient.DumpZip(
					pgVersion, back.DecryptedDatabaseConnectionString,
					compressionLevel, dumpParams,
				)
				_, dumpErr = io.Copy(f, reader)
				f.Close()
				if dumpErr == nil {
					var fi os.FileInfo
					fi, dumpErr = os.Stat(filePath)
					if dumpErr == nil {
						parts = []postgres.ZipPart{{FilePath: filePath, Size: fi.Size()}}
					}
				}
			}
		}
	}
	defer os.RemoveAll(tempDir)

	if dumpErr != nil {
		logError(dumpErr)
		return updateExec(dbgen.ExecutionsServiceUpdateExecutionParams{
			ID:         ex.ID,
			Status:     sql.NullString{Valid: true, String: "failed"},
			Message:    sql.NullString{Valid: true, String: dumpErr.Error()},
			FinishedAt: sql.NullTime{Valid: true, Time: time.Now()},
		})
	}

	date := time.Now().Format(timeutil.LayoutSlashYYYYMMDD)
	baseFile := fmt.Sprintf(
		"dump-%s-%s",
		time.Now().Format(timeutil.LayoutYYYYMMDDHHMMSS),
		uuid.NewString(),
	)

	var uploadedPaths []string
	totalFileSize := int64(0)

	for i, part := range parts {
		var fileName string
		if len(parts) == 1 {
			fileName = baseFile + ".zip"
		} else {
			fileName = fmt.Sprintf("%s-%03d.zip", baseFile, i+1)
		}
		partDestPath := strutil.CreatePath(false, back.BackupDestDir, date, fileName)

		partFile, openErr := os.Open(part.FilePath)
		if openErr != nil {
			logError(openErr)
			return updateExec(dbgen.ExecutionsServiceUpdateExecutionParams{
				ID:         ex.ID,
				Status:     sql.NullString{Valid: true, String: "failed"},
				Message:    sql.NullString{Valid: true, String: openErr.Error()},
				FinishedAt: sql.NullTime{Valid: true, Time: time.Now()},
			})
		}

		var partSize int64
		var uploadErr error
		if back.BackupIsLocal {
			partSize, uploadErr = s.ints.StorageClient.LocalUpload(partDestPath, partFile)
		} else {
			partSize, uploadErr = s.ints.StorageClient.S3Upload(
				back.DecryptedDestinationAccessKey, back.DecryptedDestinationSecretKey,
				back.DestinationRegion.String, back.DestinationEndpoint.String,
				back.DestinationBucketName.String, partDestPath,
				back.DestinationForcePathStyle.Bool,
				back.DestinationSignatureVersion.String,
				partFile,
			)
		}
		partFile.Close()

		if uploadErr != nil {
			logError(uploadErr)
			return updateExec(dbgen.ExecutionsServiceUpdateExecutionParams{
				ID:         ex.ID,
				Status:     sql.NullString{Valid: true, String: "failed"},
				Message:    sql.NullString{Valid: true, String: uploadErr.Error()},
				FinishedAt: sql.NullTime{Valid: true, Time: time.Now()},
			})
		}

		uploadedPaths = append(uploadedPaths, partDestPath)
		totalFileSize += partSize
	}

	pathJSON, _ := json.Marshal(uploadedPaths)
	pathStr := string(pathJSON)

	logger.Info("backup created successfully", logger.KV{
		"backup_id":    backupID.String(),
		"execution_id": ex.ID.String(),
		"parts":        len(parts),
	})
	return updateExec(dbgen.ExecutionsServiceUpdateExecutionParams{
		ID:         ex.ID,
		Status:     sql.NullString{Valid: true, String: "success"},
		Message:    sql.NullString{Valid: true, String: "Backup created successfully"},
		Path:       sql.NullString{Valid: true, String: pathStr},
		FinishedAt: sql.NullTime{Valid: true, Time: time.Now()},
		FileSize:   sql.NullInt64{Valid: true, Int64: totalFileSize},
	})
}
