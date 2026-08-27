package paseto

import (
	"bytes"
	"crypto/ed25519"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/require"
)

const vectorLocalKeyHex = "707172737475767778797a7b7c7d7e7f808182838485868788898a8b8c8d8e8f"

const vectorPublicKeyHex = "1eb9dbbbbc047c03fd70604e0071f0987e16b28b757225c11f00415d0e20b1a2"

const vectorSecretSeedHex = "b4cbfb43df4ce210727d953e4a713307fa19bb7d9f85041438d9e11b942a3774"

const (
	v4LocalVectorSource   = "https://github.com/paseto-standard/test-vectors/blob/f7cfbe02e4e069dddefc9001cd6303b045595ea3/v4.json#L4-L93"
	v4PublicVectorSource  = "https://github.com/paseto-standard/test-vectors/blob/f7cfbe02e4e069dddefc9001cd6303b045595ea3/v4.json#L94-L132"
	v4FailureVectorSource = "https://github.com/paseto-standard/test-vectors/blob/f7cfbe02e4e069dddefc9001cd6303b045595ea3/v4.json#L133-L185"
)

type localVector struct {
	name              string
	nonceHex          string
	token             string
	payload           string
	footer            string
	implicitAssertion string
}

// TestV4Local_OfficialVectors covers all nine v4.local vectors, 4-E-1 through
// 4-E-9, from the official PASETO test-vector repository.
//
// Source: https://github.com/paseto-standard/test-vectors/blob/f7cfbe02e4e069dddefc9001cd6303b045595ea3/v4.json#L4-L93
func TestV4Local_OfficialVectors(t *testing.T) {
	t.Logf("test vector source: %s", v4LocalVectorSource)
	key := decodeHex(t, vectorLocalKeyHex)
	vectors := []localVector{
		{
			name:              "4-E-1",
			nonceHex:          "0000000000000000000000000000000000000000000000000000000000000000",
			token:             "v4.local.AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAr68PS4AXe7If_ZgesdkUMvSwscFlAl1pk5HC0e8kApeaqMfGo_7OpBnwJOAbY9V7WU6abu74MmcUE8YWAiaArVI8XJ5hOb_4v9RmDkneN0S92dx0OW4pgy7omxgf3S8c3LlQg",
			payload:           `{"data":"this is a secret message","exp":"2022-01-01T00:00:00+00:00"}`,
			footer:            "",
			implicitAssertion: "",
		},
		{
			name:              "4-E-2",
			nonceHex:          "0000000000000000000000000000000000000000000000000000000000000000",
			token:             "v4.local.AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAr68PS4AXe7If_ZgesdkUMvS2csCgglvpk5HC0e8kApeaqMfGo_7OpBnwJOAbY9V7WU6abu74MmcUE8YWAiaArVI8XIemu9chy3WVKvRBfg6t8wwYHK0ArLxxfZP73W_vfwt5A",
			payload:           `{"data":"this is a hidden message","exp":"2022-01-01T00:00:00+00:00"}`,
			footer:            "",
			implicitAssertion: "",
		},
		{
			name:              "4-E-3",
			nonceHex:          "df654812bac492663825520ba2f6e67cf5ca5bdc13d4e7507a98cc4c2fcc3ad8",
			token:             "v4.local.32VIErrEkmY4JVILovbmfPXKW9wT1OdQepjMTC_MOtjA4kiqw7_tcaOM5GNEcnTxl60WkwMsYXw6FSNb_UdJPXjpzm0KW9ojM5f4O2mRvE2IcweP-PRdoHjd5-RHCiExR1IK6t6-tyebyWG6Ov7kKvBdkrrAJ837lKP3iDag2hzUPHuMKA",
			payload:           `{"data":"this is a secret message","exp":"2022-01-01T00:00:00+00:00"}`,
			footer:            "",
			implicitAssertion: "",
		},
		{
			name:              "4-E-4",
			nonceHex:          "df654812bac492663825520ba2f6e67cf5ca5bdc13d4e7507a98cc4c2fcc3ad8",
			token:             "v4.local.32VIErrEkmY4JVILovbmfPXKW9wT1OdQepjMTC_MOtjA4kiqw7_tcaOM5GNEcnTxl60WiA8rd3wgFSNb_UdJPXjpzm0KW9ojM5f4O2mRvE2IcweP-PRdoHjd5-RHCiExR1IK6t4gt6TiLm55vIH8c_lGxxZpE3AWlH4WTR0v45nsWoU3gQ",
			payload:           `{"data":"this is a hidden message","exp":"2022-01-01T00:00:00+00:00"}`,
			footer:            "",
			implicitAssertion: "",
		},
		{
			name:              "4-E-5",
			nonceHex:          "df654812bac492663825520ba2f6e67cf5ca5bdc13d4e7507a98cc4c2fcc3ad8",
			token:             "v4.local.32VIErrEkmY4JVILovbmfPXKW9wT1OdQepjMTC_MOtjA4kiqw7_tcaOM5GNEcnTxl60WkwMsYXw6FSNb_UdJPXjpzm0KW9ojM5f4O2mRvE2IcweP-PRdoHjd5-RHCiExR1IK6t4x-RMNXtQNbz7FvFZ_G-lFpk5RG3EOrwDL6CgDqcerSQ.eyJraWQiOiJ6VmhNaVBCUDlmUmYyc25FY1Q3Z0ZUaW9lQTlDT2NOeTlEZmdMMVc2MGhhTiJ9",
			payload:           `{"data":"this is a secret message","exp":"2022-01-01T00:00:00+00:00"}`,
			footer:            `{"kid":"zVhMiPBP9fRf2snEcT7gFTioeA9COcNy9DfgL1W60haN"}`,
			implicitAssertion: "",
		},
		{
			name:              "4-E-6",
			nonceHex:          "df654812bac492663825520ba2f6e67cf5ca5bdc13d4e7507a98cc4c2fcc3ad8",
			token:             "v4.local.32VIErrEkmY4JVILovbmfPXKW9wT1OdQepjMTC_MOtjA4kiqw7_tcaOM5GNEcnTxl60WiA8rd3wgFSNb_UdJPXjpzm0KW9ojM5f4O2mRvE2IcweP-PRdoHjd5-RHCiExR1IK6t6pWSA5HX2wjb3P-xLQg5K5feUCX4P2fpVK3ZLWFbMSxQ.eyJraWQiOiJ6VmhNaVBCUDlmUmYyc25FY1Q3Z0ZUaW9lQTlDT2NOeTlEZmdMMVc2MGhhTiJ9",
			payload:           `{"data":"this is a hidden message","exp":"2022-01-01T00:00:00+00:00"}`,
			footer:            `{"kid":"zVhMiPBP9fRf2snEcT7gFTioeA9COcNy9DfgL1W60haN"}`,
			implicitAssertion: "",
		},
		{
			name:              "4-E-7",
			nonceHex:          "df654812bac492663825520ba2f6e67cf5ca5bdc13d4e7507a98cc4c2fcc3ad8",
			token:             "v4.local.32VIErrEkmY4JVILovbmfPXKW9wT1OdQepjMTC_MOtjA4kiqw7_tcaOM5GNEcnTxl60WkwMsYXw6FSNb_UdJPXjpzm0KW9ojM5f4O2mRvE2IcweP-PRdoHjd5-RHCiExR1IK6t40KCCWLA7GYL9KFHzKlwY9_RnIfRrMQpueydLEAZGGcA.eyJraWQiOiJ6VmhNaVBCUDlmUmYyc25FY1Q3Z0ZUaW9lQTlDT2NOeTlEZmdMMVc2MGhhTiJ9",
			payload:           `{"data":"this is a secret message","exp":"2022-01-01T00:00:00+00:00"}`,
			footer:            `{"kid":"zVhMiPBP9fRf2snEcT7gFTioeA9COcNy9DfgL1W60haN"}`,
			implicitAssertion: `{"test-vector":"4-E-7"}`,
		},
		{
			name:              "4-E-8",
			nonceHex:          "df654812bac492663825520ba2f6e67cf5ca5bdc13d4e7507a98cc4c2fcc3ad8",
			token:             "v4.local.32VIErrEkmY4JVILovbmfPXKW9wT1OdQepjMTC_MOtjA4kiqw7_tcaOM5GNEcnTxl60WiA8rd3wgFSNb_UdJPXjpzm0KW9ojM5f4O2mRvE2IcweP-PRdoHjd5-RHCiExR1IK6t5uvqQbMGlLLNYBc7A6_x7oqnpUK5WLvj24eE4DVPDZjw.eyJraWQiOiJ6VmhNaVBCUDlmUmYyc25FY1Q3Z0ZUaW9lQTlDT2NOeTlEZmdMMVc2MGhhTiJ9",
			payload:           `{"data":"this is a hidden message","exp":"2022-01-01T00:00:00+00:00"}`,
			footer:            `{"kid":"zVhMiPBP9fRf2snEcT7gFTioeA9COcNy9DfgL1W60haN"}`,
			implicitAssertion: `{"test-vector":"4-E-8"}`,
		},
		{
			name:              "4-E-9",
			nonceHex:          "df654812bac492663825520ba2f6e67cf5ca5bdc13d4e7507a98cc4c2fcc3ad8",
			token:             "v4.local.32VIErrEkmY4JVILovbmfPXKW9wT1OdQepjMTC_MOtjA4kiqw7_tcaOM5GNEcnTxl60WiA8rd3wgFSNb_UdJPXjpzm0KW9ojM5f4O2mRvE2IcweP-PRdoHjd5-RHCiExR1IK6t6tybdlmnMwcDMw0YxA_gFSE_IUWl78aMtOepFYSWYfQA.YXJiaXRyYXJ5LXN0cmluZy10aGF0LWlzbid0LWpzb24",
			payload:           `{"data":"this is a hidden message","exp":"2022-01-01T00:00:00+00:00"}`,
			footer:            "arbitrary-string-that-isn't-json",
			implicitAssertion: `{"test-vector":"4-E-9"}`,
		},
	}

	for _, vector := range vectors {
		t.Run(vector.name, func(t *testing.T) {
			nonce := decodeHex(t, vector.nonceHex)
			token, err := encryptToken(
				key,
				[]byte(vector.payload),
				[]byte(vector.footer),
				[]byte(vector.implicitAssertion),
				bytes.NewReader(nonce),
			)
			require.NoError(t, err)
			require.Equal(t, vector.token, token)

			payload, footer, err := decryptToken(key, vector.token, []byte(vector.implicitAssertion))
			require.NoError(t, err)
			require.Equal(t, vector.payload, string(payload))
			require.Equal(t, vector.footer, string(footer))
			if vector.implicitAssertion != "" {
				_, _, err = decryptToken(key, vector.token, nil)
				require.ErrorIs(t, err, ErrInvalidToken)
			}
		})
	}
}

type publicVector struct {
	name              string
	token             string
	payload           string
	footer            string
	implicitAssertion string
}

// TestV4Public_OfficialVectors covers all three v4.public vectors, 4-S-1
// through 4-S-3, from the official PASETO test-vector repository. Ed25519
// signatures are deterministic, so the complete token must match.
//
// Source: https://github.com/paseto-standard/test-vectors/blob/f7cfbe02e4e069dddefc9001cd6303b045595ea3/v4.json#L94-L132
func TestV4Public_OfficialVectors(t *testing.T) {
	t.Logf("test vector source: %s", v4PublicVectorSource)
	privateKey := ed25519.NewKeyFromSeed(decodeHex(t, vectorSecretSeedHex))
	publicKey := ed25519.PublicKey(decodeHex(t, vectorPublicKeyHex))
	vectors := []publicVector{
		{
			name:              "4-S-1",
			token:             "v4.public.eyJkYXRhIjoidGhpcyBpcyBhIHNpZ25lZCBtZXNzYWdlIiwiZXhwIjoiMjAyMi0wMS0wMVQwMDowMDowMCswMDowMCJ9bg_XBBzds8lTZShVlwwKSgeKpLT3yukTw6JUz3W4h_ExsQV-P0V54zemZDcAxFaSeef1QlXEFtkqxT1ciiQEDA",
			payload:           `{"data":"this is a signed message","exp":"2022-01-01T00:00:00+00:00"}`,
			footer:            "",
			implicitAssertion: "",
		},
		{
			name:              "4-S-2",
			token:             "v4.public.eyJkYXRhIjoidGhpcyBpcyBhIHNpZ25lZCBtZXNzYWdlIiwiZXhwIjoiMjAyMi0wMS0wMVQwMDowMDowMCswMDowMCJ9v3Jt8mx_TdM2ceTGoqwrh4yDFn0XsHvvV_D0DtwQxVrJEBMl0F2caAdgnpKlt4p7xBnx1HcO-SPo8FPp214HDw.eyJraWQiOiJ6VmhNaVBCUDlmUmYyc25FY1Q3Z0ZUaW9lQTlDT2NOeTlEZmdMMVc2MGhhTiJ9",
			payload:           `{"data":"this is a signed message","exp":"2022-01-01T00:00:00+00:00"}`,
			footer:            `{"kid":"zVhMiPBP9fRf2snEcT7gFTioeA9COcNy9DfgL1W60haN"}`,
			implicitAssertion: "",
		},
		{
			name:              "4-S-3",
			token:             "v4.public.eyJkYXRhIjoidGhpcyBpcyBhIHNpZ25lZCBtZXNzYWdlIiwiZXhwIjoiMjAyMi0wMS0wMVQwMDowMDowMCswMDowMCJ9NPWciuD3d0o5eXJXG5pJy-DiVEoyPYWs1YSTwWHNJq6DZD3je5gf-0M4JR9ipdUSJbIovzmBECeaWmaqcaP0DQ.eyJraWQiOiJ6VmhNaVBCUDlmUmYyc25FY1Q3Z0ZUaW9lQTlDT2NOeTlEZmdMMVc2MGhhTiJ9",
			payload:           `{"data":"this is a signed message","exp":"2022-01-01T00:00:00+00:00"}`,
			footer:            `{"kid":"zVhMiPBP9fRf2snEcT7gFTioeA9COcNy9DfgL1W60haN"}`,
			implicitAssertion: `{"test-vector":"4-S-3"}`,
		},
	}

	for _, vector := range vectors {
		t.Run(vector.name, func(t *testing.T) {
			token := signToken(
				privateKey,
				[]byte(vector.payload),
				[]byte(vector.footer),
				[]byte(vector.implicitAssertion),
			)
			require.Equal(t, vector.token, token)

			payload, footer, err := verifyToken(publicKey, vector.token, []byte(vector.implicitAssertion))
			require.NoError(t, err)
			require.Equal(t, vector.payload, string(payload))
			require.Equal(t, vector.footer, string(footer))
			if vector.implicitAssertion != "" {
				_, _, err = verifyToken(publicKey, vector.token, nil)
				require.ErrorIs(t, err, ErrInvalidToken)
			}
		})
	}
}

// TestV4_OfficialFailureVectors covers all five failure vectors, 4-F-1 through
// 4-F-5, from the official PASETO test-vector repository. These vectors protect
// purpose and version separation, authentication, and unpadded Base64url.
//
// Source: https://github.com/paseto-standard/test-vectors/blob/f7cfbe02e4e069dddefc9001cd6303b045595ea3/v4.json#L133-L185
func TestV4_OfficialFailureVectors(t *testing.T) {
	t.Logf("test vector source: %s", v4FailureVectorSource)
	localKey := decodeHex(t, vectorLocalKeyHex)
	publicKey := ed25519.PublicKey(decodeHex(t, vectorPublicKeyHex))
	tests := []struct {
		name  string
		token string
		check func(string) error
	}{
		{
			name:  "4-F-1 public key cannot verify a local token",
			token: "v4.local.vngXfCISbnKgiP6VWGuOSlYrFYU300fy9ijW33rznDYgxHNPwWluAY2Bgb0z54CUs6aYYkIJ-bOOOmJHPuX_34Agt_IPlNdGDpRdGNnBz2MpWJvB3cttheEc1uyCEYltj7wBQQYX.YXJiaXRyYXJ5LXN0cmluZy10aGF0LWlzbid0LWpzb24",
			check: func(token string) error {
				_, _, err := verifyToken(publicKey, token, nil)
				return err
			},
		},
		{
			name:  "4-F-2 local key cannot decrypt a public token",
			token: "v4.public.eyJpbnZhbGlkIjoidGhpcyBzaG91bGQgbmV2ZXIgZGVjb2RlIn22Sp4gjCaUw0c7EH84ZSm_jN_Qr41MrgLNu5LIBCzUr1pn3Z-Wukg9h3ceplWigpoHaTLcwxj0NsI1vjTh67YB.eyJraWQiOiJ6VmhNaVBCUDlmUmYyc25FY1Q3Z0ZUaW9lQTlDT2NOeTlEZmdMMVc2MGhhTiJ9",
			check: func(token string) error {
				_, _, err := decryptToken(localKey, token, nil)
				return err
			},
		},
		{
			name:  "4-F-3 v3 token cannot be decrypted as v4",
			token: "v3.local.23e_2PiqpQBPvRFKzB0zHhjmxK3sKo2grFZRRLM-U7L0a8uHxuF9RlVz3Ic6WmdUUWTxCaYycwWV1yM8gKbZB2JhygDMKvHQ7eBf8GtF0r3K0Q_gF1PXOxcOgztak1eD1dPe9rLVMSgR0nHJXeIGYVuVrVoLWQ.YXJiaXRyYXJ5LXN0cmluZy10aGF0LWlzbid0LWpzb24",
			check: func(token string) error {
				_, _, err := decryptToken(localKey, token, nil)
				return err
			},
		},
		{
			name:  "4-F-4 changed authentication tag",
			token: "v4.local.AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAr68PS4AXe7If_ZgesdkUMvSwscFlAl1pk5HC0e8kApeaqMfGo_7OpBnwJOAbY9V7WU6abu74MmcUE8YWAiaArVI8XJ5hOb_4v9RmDkneN0S92dx0OW4pgy7omxgf3S8c3LlQh",
			check: func(token string) error {
				_, _, err := decryptToken(localKey, token, nil)
				return err
			},
		},
		{
			name:  "4-F-5 padded Base64url",
			token: "v4.local.32VIErrEkmY4JVILovbmfPXKW9wT1OdQepjMTC_MOtjA4kiqw7_tcaOM5GNEcnTxl60WkwMsYXw6FSNb_UdJPXjpzm0KW9ojM5f4O2mRvE2IcweP-PRdoHjd5-RHCiExR1IK6t4x-RMNXtQNbz7FvFZ_G-lFpk5RG3EOrwDL6CgDqcerSQ==.eyJraWQiOiJ6VmhNaVBCUDlmUmYyc25FY1Q3Z0ZUaW9lQTlDT2NOeTlEZmdMMVc2MGhhTiJ9",
			check: func(token string) error {
				_, _, err := decryptToken(localKey, token, nil)
				return err
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.ErrorIs(t, test.check(test.token), ErrInvalidToken)
		})
	}
}

func decodeHex(t *testing.T, encoded string) []byte {
	t.Helper()
	decoded, err := hex.DecodeString(encoded)
	require.NoError(t, err)
	return decoded
}
