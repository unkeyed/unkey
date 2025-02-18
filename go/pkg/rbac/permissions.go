package rbac

import (
	"errors"
	"fmt"
	"strings"
)

type ActionType string

const (
	// API Actions
	ReadAPI    ActionType = "read_api"
	CreateAPI  ActionType = "create_api"
	DeleteAPI  ActionType = "delete_api"
	UpdateAPI  ActionType = "update_api"
	CreateKey  ActionType = "create_key"
	UpdateKey  ActionType = "update_key"
	DeleteKey  ActionType = "delete_key"
	EncryptKey ActionType = "encrypt_key"
	DecryptKey ActionType = "decrypt_key"
	ReadKey    ActionType = "read_key"

	// Ratelimit Actions
	Limit           ActionType = "limit"
	CreateNamespace ActionType = "create_namespace"
	ReadNamespace   ActionType = "read_namespace"
	UpdateNamespace ActionType = "update_namespace"
	DeleteNamespace ActionType = "delete_namespace"
	SetOverride     ActionType = "set_override"
	ReadOverride    ActionType = "read_override"
	DeleteOverride  ActionType = "delete_override"

	// RBAC Actions
	CreatePermission         ActionType = "create_permission"
	UpdatePermission         ActionType = "update_permission"
	DeletePermission         ActionType = "delete_permission"
	ReadPermission           ActionType = "read_permission"
	CreateRole               ActionType = "create_role"
	UpdateRole               ActionType = "update_role"
	DeleteRole               ActionType = "delete_role"
	ReadRole                 ActionType = "read_role"
	AddPermissionToKey       ActionType = "add_permission_to_key"
	RemovePermissionFromKey  ActionType = "remove_permission_from_key"
	AddRoleToKey             ActionType = "add_role_to_key"
	RemoveRoleFromKey        ActionType = "remove_role_from_key"
	AddPermissionToRole      ActionType = "add_permission_to_role"
	RemovePermissionFromRole ActionType = "remove_permission_from_role"

	// Identity Actions
	CreateIdentity ActionType = "create_identity"
	ReadIdentity   ActionType = "read_identity"
	UpdateIdentity ActionType = "update_identity"
	DeleteIdentity ActionType = "delete_identity"
)

type Tuple struct {
	ResourceType string
	ResourceID   string
	Action       string
}

func (t Tuple) String() string {
	return fmt.Sprintf("%s:%s:%s", t.ResourceType, t.ResourceID, t.Action)
}

func TupleFromString(s string) (Tuple, error) {
	parts := strings.Split(s, ":")
	if len(parts) != 3 {
		return Tuple{}, errors.New("invalid tuple format")

	}
	tuple := Tuple{
		ResourceType: parts[0],
		ResourceID:   parts[1],
		Action:       parts[2],
	}
	return tuple, nil
}
