package githubsecrets

type publicKeysResponse struct {
	PublicKeys []publicKey `json:"public_keys"`
}

type publicKey struct {
	KeyIdentifier string `json:"key_identifier"`
	Key           string `json:"key"`
}
