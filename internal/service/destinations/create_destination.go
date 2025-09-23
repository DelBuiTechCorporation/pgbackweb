package destinations

import (
	"context"
	"strings"

	"github.com/eduardolat/pgbackweb/internal/database/dbgen"
)

func (s *Service) CreateDestination(
	ctx context.Context, params dbgen.DestinationsServiceCreateDestinationParams,
) (dbgen.Destination, error) {
	if !strings.HasPrefix(params.Endpoint, "https://") && !strings.HasPrefix(params.Endpoint, "http://") {
		params.Endpoint = "https://" + params.Endpoint
	}

	err := s.TestDestination(
		params.AccessKey, params.SecretKey, params.Region, params.Endpoint,
		params.BucketName, params.Provider, params.ForcePathStyle,
	)
	if err != nil {
		return dbgen.Destination{}, err
	}

	params.EncryptionKey = s.env.PBW_ENCRYPTION_KEY
	dest, err := s.dbgen.DestinationsServiceCreateDestination(ctx, params)

	_ = s.TestDestinationAndStoreResult(ctx, dest.ID)

	return dest, err
}

