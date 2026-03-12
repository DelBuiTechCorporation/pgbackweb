package executions

import (
	"context"
	"database/sql"
	"errors"

	"github.com/eduardolat/pgbackweb/internal/database/dbgen"
	"github.com/eduardolat/pgbackweb/internal/util/strutil"
	"github.com/google/uuid"
)

func (s *Service) SoftDeleteExecution(
	ctx context.Context, executionID uuid.UUID,
) error {
	execution, err := s.dbgen.ExecutionsServiceGetExecutionForSoftDelete(
		ctx, dbgen.ExecutionsServiceGetExecutionForSoftDeleteParams{
			ExecutionID:   executionID,
			EncryptionKey: s.env.PBW_ENCRYPTION_KEY,
		},
	)
	if err != nil && errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}

	if execution.ExecutionPath.Valid {
		for _, p := range strutil.ParseJSONStringArray(execution.ExecutionPath.String) {
			if !execution.BackupIsLocal {
				err := s.ints.StorageClient.S3Delete(
					execution.DecryptedDestinationAccessKey, execution.DecryptedDestinationSecretKey,
					execution.DestinationRegion.String, execution.DestinationEndpoint.String,
					execution.DestinationBucketName.String, p,
					execution.DestinationForcePathStyle.Bool,
					execution.DestinationSignatureVersion.String,
				)
				if err != nil {
					return err
				}
			} else {
				err := s.ints.StorageClient.LocalDelete(p)
				if err != nil {
					return err
				}
			}
		}
	}

	return s.dbgen.ExecutionsServiceSoftDeleteExecution(ctx, executionID)
}
