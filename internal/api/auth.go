package api

import (
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
		return nil, fmt.Errorf("Missing user-email header")
	}

	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return nil, fmt.Errorf("Missing authorization header")
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		return nil, fmt.Errorf("Invalid authorization header format")
	}

	token := parts[1]

	user, err := postgres.GetUserByEmail(r.Context(), dbpool, email)
	if err != nil {
		return nil, fmt.Errorf("User not found")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.HashedBearerToken), []byte(token)); err != nil {
		return nil, fmt.Errorf("Invalid token")
	}

	return user, nil
}
