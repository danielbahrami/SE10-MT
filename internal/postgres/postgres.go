package postgres

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

func ConnectPostgres() (*pgxpool.Pool, error) {
	DATABASE_URL := "postgres://" + os.Getenv("POSTGRES_USER") + ":" + os.Getenv("POSTGRES_PASSWORD") + "@" +
		os.Getenv("POSTGRES_HOST") + ":" + os.Getenv("POSTGRES_PORT") + "/" + os.Getenv("POSTGRES_DB")
	fmt.Println("DATABASE_URL: " + DATABASE_URL)

	dbpool, err := pgxpool.New(context.Background(), DATABASE_URL)
	if err != nil {
		return nil, fmt.Errorf("unable to create connection pool: %w", err)
	}
	return dbpool, nil
}
