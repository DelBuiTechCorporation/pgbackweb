package auth

import (
	"context"
	"fmt"

	"github.com/eduardolat/pgbackweb/internal/database/dbgen"
	"github.com/eduardolat/pgbackweb/internal/util/cryptoutil"
	"github.com/google/uuid"
)

func (s *Service) Login(
	ctx context.Context, email, password, ip, userAgent string,
) (dbgen.AuthServiceLoginCreateSessionRow, error) {
	user, err := s.dbgen.AuthServiceLoginGetUserByEmail(ctx, email)
	if err != nil {
		return dbgen.AuthServiceLoginCreateSessionRow{}, err
	}

	if err := cryptoutil.VerifyBcryptHash(password, user.Password); err != nil {
		return dbgen.AuthServiceLoginCreateSessionRow{}, fmt.Errorf("invalid password")
	}

	session, err := s.dbgen.AuthServiceLoginCreateSession(
		ctx, dbgen.AuthServiceLoginCreateSessionParams{
			UserID:        user.ID,
			Ip:            ip,
			UserAgent:     userAgent,
			Token:         uuid.NewString(),
			EncryptionKey: s.env.PBW_ENCRYPTION_KEY,
		},
	)
	if err != nil {
		return dbgen.AuthServiceLoginCreateSessionRow{}, err
	}

	return session, nil
}
