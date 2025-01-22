package main

import (
	"fmt"
	"regexp"
)

func main() {

	r := regexp.MustCompile(`"key":\s*"([a-zA-Z0-9_]+)"`)

	out := r.ReplaceAllSu([]byte("Hello World\"key\":   \"key_123\" asdasd"), func(match []byte) []byte {
		submatches := r.(match)
		for i := 0; i < len(submatches); i++ {

			fmt.Printf("%d: %s\n", i, submatches[i])
		}
		return []byte(string(submatches[1]) + "XXX" + string(submatches[3]))

	})

	fmt.Println(string(out))
}
