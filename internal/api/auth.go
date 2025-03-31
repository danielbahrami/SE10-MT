package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/danielbahrami/se10-mt/internal/postgres"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

func AuthenticateUser(r *http.Request, dbpool *pgxpool.Pool) (*postgres.User, error) {
	email := r.Header.Get("User-Email")
	if email == "" {
		return nil, fmt.Errorf("missing user-email header")
	}

	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return nil, fmt.Errorf("missing authorization header")
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		return nil, fmt.Errorf("invalid authorization header format")
	}

	token := parts[1]

	user, err := postgres.GetUserByEmail(context.Background(), dbpool, email)
	if err != nil {
		return nil, fmt.Errorf("user not found")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.HashedBearerToken), []byte(token)); err != nil {
		return nil, fmt.Errorf("invalid token")
	}

	return user, nil
}
