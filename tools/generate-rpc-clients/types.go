package main

type serviceInfo struct {
	Name    string // e.g. "VaultServiceClient"
	Methods []methodInfo
}

type methodKind string

const (
	methodKindUnary        methodKind = "unary"
	methodKindServerStream methodKind = "server_stream"
)

type methodInfo struct {
	Name     string     // e.g. "Encrypt"
	ReqType  string     // e.g. "EncryptRequest"
	RespType string     // e.g. "EncryptResponse"
	Kind     methodKind // unary or server_stream
}

type fileData struct {
	PackageName   string // e.g. "vaultrpc"
	ConnectPkg    string // e.g. "vaultv1connect"
	ConnectImport string // e.g. "github.com/unkeyed/unkey/gen/proto/vault/v1/vaultv1connect"
	ProtoAlias    string // e.g. "v1"
	ProtoImport   string // e.g. "github.com/unkeyed/unkey/gen/proto/vault/v1"
	Services      []serviceInfo
}
