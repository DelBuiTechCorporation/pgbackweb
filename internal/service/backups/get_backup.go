package backups

import (
	"context"

	"github.com/eduardolat/pgbackweb/internal/database/dbgen"
	"github.com/google/uuid"
)

func (s *Service) GetBackup(
	ctx context.Context, id uuid.UUID,
) (dbgen.BackupsServiceGetBackupRow, error) {
	return s.dbgen.BackupsServiceGetBackup(ctx, dbgen.BackupsServiceGetBackupParams{
		ID:            id,
		EncryptionKey: s.env.PBW_ENCRYPTION_KEY,
	})
}
