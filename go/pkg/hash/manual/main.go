package main

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"

	"github.com/unkeyed/unkey/go/pkg/hash"
)

func main() {

	//{
	//  shortToken: "8UQ3nYGa",
	//  longToken: "33oTRV6YCmL97sbZgbAmGZEM",
	//  longTokenHash: "48a99769ca64e9153ed4ad956865ede54aca1424a305f5928891e58be46bd8fa",
	//  token: "resend_8UQ3nYGa_33oTRV6YCmL97sbZgbAmGZEM",
	//}

	const (
		shortToken    = "8UQ3nYGa"
		longToken     = "33oTRV6YCmL97sbZgbAmGZEM"
		longTokenHash = "48a99769ca64e9153ed4ad956865ede54aca1424a305f5928891e58be46bd8fa"
		token         = "resend_8UQ3nYGa_33oTRV6YCmL97sbZgbAmGZEM"
	)

	longTokenDec, err := hex.DecodeString(longTokenHash)
	if err != nil {
		panic(err)
	}

	fmt.Printf("longTokenB64=%s\n", base64.StdEncoding.EncodeToString(longTokenDec))

	sha256 := hash.Sha256(longToken)

	fmt.Printf("sha256(token)=%s\n", sha256)

}
