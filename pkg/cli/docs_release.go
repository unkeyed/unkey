//go:build cli_release

package cli

// Release binaries omit runtime MDX generation to avoid linking the template engine.
func (*Command) handleMDXGeneration(_ []string) (bool, error) {
	return false, nil
}
