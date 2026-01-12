package vault

import "github.com/unkeyed/unkey/pkg/assert"

type Config struct {
	// InstanceID is the unique identifier for this instance of the API server
	InstanceID string

	// HttpPort defines the HTTP port for the API server to listen on (default: 7070)
	HttpPort int

	// S3Bucket is the bucket to store secrets in
	S3Bucket string
	// S3Url is the url to store secrets in
	S3Url string
	// S3AccessKeyID is the access key id to use for s3
	S3AccessKeyID string
	// S3AccessKeySecret is the access key secret to use for s3
	S3AccessKeySecret string
	// MasterKeys
	// The first key is used for encryption, additional keys may be provided for backwards compatible decryption
	//
	// If multiple keys are provided, vault will start a rekey process to migrate all secrets to the new key
	MasterKeys []string
	// BearerToken is the authentication token for securing vault operations
	BearerToken string
}

func (c Config) Validate() error {

	return assert.All(
		assert.NotEmpty(c.InstanceID, "instanceID must not be empty"),
		assert.Greater(c.HttpPort, 0, "httpPort must be greater than 0"),
		assert.NotEmpty(c.S3Bucket, "s3Bucket must not be empty"),
		assert.NotEmpty(c.S3Url, "s3Url must not be empty"),
		assert.NotEmpty(c.S3AccessKeyID, "s3AccessKeyID must not be empty"),
		assert.NotEmpty(c.S3AccessKeySecret, "s3AccessKeySecret must not be empty"),
		assert.NotEmpty(c.MasterKeys, "masterKeys must not be empty"),
		assert.NotEmpty(c.BearerToken, "bearerToken must not be empty"),
	)

}
