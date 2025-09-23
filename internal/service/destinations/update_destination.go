package destinations

import (
	"context"

	"github.com/eduardolat/pgbackweb/internal/database/dbgen"
)

func (s *Service) UpdateDestination(
	ctx context.Context, params dbgen.DestinationsServiceUpdateDestinationParams,
) (dbgen.Destination, error) {
	// Get current destination to fill in missing values for testing
	currentDest, err := s.GetDestination(ctx, params.ID)
	if err != nil {
		return dbgen.Destination{}, err
	}

	// Use new values if provided, otherwise use current values
	accessKey := currentDest.DecryptedAccessKey
	if params.AccessKey.Valid {
		accessKey = params.AccessKey.String
	}

	secretKey := currentDest.DecryptedSecretKey
	if params.SecretKey.Valid {
		secretKey = params.SecretKey.String
	}

	region := currentDest.Region
	if params.Region.Valid {
		region = params.Region.String
	}

	endpoint := currentDest.Endpoint
	if params.Endpoint.Valid {
		endpoint = params.Endpoint.String
	}

	bucketName := currentDest.BucketName
	if params.BucketName.Valid {
		bucketName = params.BucketName.String
	}

	provider := currentDest.Provider
	if params.Provider.Valid {
		provider = params.Provider.String
	}

	forcePathStyle := currentDest.ForcePathStyle
	if params.ForcePathStyle.Valid {
		forcePathStyle = params.ForcePathStyle.Bool
	}

	err = s.TestDestination(
		accessKey, secretKey, region, endpoint, bucketName, provider, forcePathStyle,
	)
	if err != nil {
		return dbgen.Destination{}, err
	}

	params.EncryptionKey = s.env.PBW_ENCRYPTION_KEY
	dest, err := s.dbgen.DestinationsServiceUpdateDestination(ctx, params)

	_ = s.TestDestinationAndStoreResult(ctx, dest.ID)

	return dest, err
}
