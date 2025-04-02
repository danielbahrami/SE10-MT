package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func ConnectPostgres() (*pgxpool.Pool, error) {
	DATABASE_URL := fmt.Sprintf("postgres://%s:%s@%s:%s/%s",
		os.Getenv("POSTGRES_USER"),
		os.Getenv("POSTGRES_PASSWORD"),
		os.Getenv("POSTGRES_HOST"),
		os.Getenv("POSTGRES_PORT"),
		os.Getenv("POSTGRES_DB"),
	)

	dbpool, err := pgxpool.New(context.Background(), DATABASE_URL)
	if err != nil {
		return nil, fmt.Errorf("Unable to create connection pool: %w", err)
	}

	fmt.Println("Connected to Postgres")
	return dbpool, nil
}

func GetUserByEmail(ctx context.Context, dbpool *pgxpool.Pool, email string) (*User, error) {
	sql := `
        SELECT id, org_id, name, email, hashed_bearer_token, override_permissions, created_at, updated_at FROM users WHERE email = $1
	`
	row := dbpool.QueryRow(ctx, sql, email)

	var user User
	err := row.Scan(&user.ID, &user.OrgID, &user.Name, &user.Email, &user.HashedBearerToken, &user.OverridePermissions, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("GetUserByEmail: %w", err)
	}

	return &user, nil
}

func GetOrganizationById(ctx context.Context, dbpool *pgxpool.Pool, id int) (*Organization, error) {
	sql := `
        SELECT id, name, default_permissions, created_at, updated_at FROM organizations WHERE id = $1
	`
	row := dbpool.QueryRow(ctx, sql, id)

	var org Organization
	err := row.Scan(&org.ID, &org.Name, &org.DefaultPermissions, &org.CreatedAt, &org.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("GetOrganizationById: %w", err)
	}

	return &org, nil
}

func GetUserPermissions(ctx context.Context, dbpool *pgxpool.Pool, user *User) (*Permissions, error) {
	var effectivePermissions string
	if user.OverridePermissions.Valid && user.OverridePermissions.String != "" {
		effectivePermissions = user.OverridePermissions.String
	} else {
		org, err := GetOrganizationById(ctx, dbpool, user.OrgID)
		if err != nil {
			return nil, fmt.Errorf("GetUserPermissions: %w", err)
		}
		effectivePermissions = org.DefaultPermissions
	}

	var permissions Permissions
	if err := json.Unmarshal([]byte(effectivePermissions), &permissions); err != nil {
		return nil, fmt.Errorf("GetUserPermissions: %w", err)
	}

	return &permissions, nil
}

func LogQuery(ctx context.Context, dbpool *pgxpool.Pool, userId int, query, decision, rewrittenQuery string) error {
	sql := `
        INSERT INTO logs (user_id, query, decision, rewritten_query, created_at) VALUES ($1, $2, $3, $4, $5)
	`
	_, err := dbpool.Exec(ctx, sql, userId, query, decision, rewrittenQuery, time.Now())

	return err
}
