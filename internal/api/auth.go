package api

import (
	"context"
	"fmt"
	"strings"

	"github.com/danielbahrami/se10-mt/internal/postgres"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

func AuthenticateUser(c *gin.Context, dbpool *pgxpool.Pool) (*postgres.User, error) {
	// Retrieve email from custom header
	email := c.GetHeader("User-Email")
	if email == "" {
		return nil, fmt.Errorf("missing user email")
	}

	// Retrieve authorization header and extract token
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		return nil, fmt.Errorf("missing authorization header")
	}

	// Expected header format: "Bearer <token>"
	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		return nil, fmt.Errorf("invalid authorization header format")
	}

	token := parts[1]

	// Retrieve user by email
	user, err := postgres.GetUserByEmail(context.Background(), dbpool, email)
	if err != nil {
		return nil, fmt.Errorf("user not found")
	}

	// Compare provided token with stored hashed token
	if err := bcrypt.CompareHashAndPassword([]byte(user.HashedBearerToken), []byte(token)); err != nil {
		return nil, fmt.Errorf("invalid token")
	}

	return user, nil
}
