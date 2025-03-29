package postgres

import (
	"database/sql"
	"time"
)

type Organization struct {
	ID                 int
	Name               string
	DefaultPermissions string
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type User struct {
	ID                  int
	OrgID               int
	Name                string
	Email               string
	HashedBearerToken   string
	OverridePermissions sql.NullString
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

type Log struct {
	ID             int
	UserID         int
	Query          string
	Decision       string // Allowed, Blocked, Rewritten
	RewrittenQuery string
	CreatedAt      time.Time
}
