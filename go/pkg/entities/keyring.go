package entities

import "time"

type Keyring struct {
	ID                 string
	WorkspaceID        string
	StoreEncryptedKeys bool
	DefaultPrefix      string
	DefaultBytes       int
	CreatedAt          time.Time
	UpdatedAt          time.Time
	DeletedAt          time.Time
}
