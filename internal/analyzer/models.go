package analyzer

// Defines what CRUD operations are allowed for an entity
type OperationPermissions struct {
	Read   bool `json:"read"`
	Create bool `json:"create"`
	Update bool `json:"update"`
	Delete bool `json:"delete"`
}

// Defines which relationship types a user is allowed to work with along with the direction of the relationship
type RelationshipPermission struct {
	Direction string   `json:"direction"` // "incoming", "outgoing", or "both"
	Types     []string `json:"types"`
}

// The overall structure representing the access rules
// AllowedLabels: The list of node labels a user can access.
// AllowedRelationships: The allowed relationship types along with their permitted directions
// AllowedProperties: A mapping from an entity (like a node label) to a list of accessible properties
// OperationPermissions: Which CRUD operations are permitted for different entities
type Permissions struct {
	AllowedLabels        []string                        `json:"allowed_labels"`
	AllowedRelationships []RelationshipPermission        `json:"allowed_relationships"`
	AllowedProperties    map[string][]string             `json:"allowed_properties"`
	OperationPermissions map[string]OperationPermissions `json:"operation_permissions,omitempty"`
}
