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
	Decision       string // "Allowed", "Blocked", or "Rewritten"
	RewrittenQuery string
	CreatedAt      time.Time
}

// Defines what CRUD operations are allowed for an entity
type OperationPermissions struct {
	Read   bool `json:"read"`
	Create bool `json:"create"`
	Update bool `json:"update"`
	Delete bool `json:"delete"`
}

// The overall structure representing the access rules
// AllowedLabels: The list of node labels a user can access
// AllowedRelationships: The allowed relationship types
// AllowedProperties: A mapping from an entity (like a node label) to a list of accessible properties
// OperationPermissions: Which CRUD operations are permitted for different entities
type Permissions struct {
	AllowedLabels        []string                        `json:"allowed_labels"`
	AllowedRelationships []string                        `json:"allowed_relationships"`
	AllowedProperties    map[string][]string             `json:"allowed_properties"`
	OperationPermissions map[string]OperationPermissions `json:"operation_permissions,omitempty"`
}
