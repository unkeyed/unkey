package billingjob

import (
	"context"
	"os"
)

func main() {
	if err := Cmd.Run(context.Background(), os.Args[1:]); err != nil {
		os.Exit(1)
	}
}