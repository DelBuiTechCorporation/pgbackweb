package executions

import (
	"context"
	"fmt"
	"time"

	"github.com/eduardolat/pgbackweb/internal/database/dbgen"
	"github.com/eduardolat/pgbackweb/internal/util/strutil"
	"github.com/google/uuid"
)

// GetAllExecutionLinksOrPaths returns all download links or paths for the files
// associated with the given execution. Supports both single-file (legacy) and
// multi-part backups.
//
// Returns a boolean indicating if the files are locally stored and the list of
// download links/paths.
func (s *Service) GetAllExecutionLinksOrPaths(
	ctx context.Context, executionID uuid.UUID,
) (bool, []string, error) {
	data, err := s.dbgen.ExecutionsServiceGetDownloadLinkOrPathData(
		ctx, dbgen.ExecutionsServiceGetDownloadLinkOrPathDataParams{
			ExecutionID:   executionID,
			DecryptionKey: s.env.PBW_ENCRYPTION_KEY,
		},
	)
	if err != nil {
		return false, nil, err
	}

	if !data.Path.Valid {
		return false, nil, fmt.Errorf("execution has no file associated")
	}

	paths := strutil.ParseJSONStringArray(data.Path.String)

	if data.IsLocal {
		var fullPaths []string
		for _, p := range paths {
			fullPaths = append(fullPaths, s.ints.StorageClient.LocalGetFullPath(p))
		}
		return true, fullPaths, nil
	}

	var links []string
	for _, p := range paths {
		link, err := s.ints.StorageClient.S3GetDownloadLink(
			data.DecryptedAccessKey, data.DecryptedSecretKey, data.Region.String,
			data.Endpoint.String, data.BucketName.String, p,
			data.ForcePathStyle.Bool, data.SignatureVersion.String,
			time.Hour*12,
		)
		if err != nil {
			return false, nil, err
		}
		links = append(links, link)
	}
	return false, links, nil
}

// GetExecutionDownloadLinkOrPath returns the download link or path for the
// first file associated with the given execution. For multi-part backups,
// returns only the first part.
//
// Returns a boolean indicating if the file is locally stored and the download
// link/path.
func (s *Service) GetExecutionDownloadLinkOrPath(
	ctx context.Context, executionID uuid.UUID,
) (bool, string, error) {
	isLocal, links, err := s.GetAllExecutionLinksOrPaths(ctx, executionID)
	if err != nil {
		return false, "", err
	}
	if len(links) == 0 {
		return false, "", fmt.Errorf("execution has no file associated")
	}
	return isLocal, links[0], nil
}

