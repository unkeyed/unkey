package array

// Chunk splits a slice into chunks of size n.
func Chunk[T any](slice []T, n int) [][]T {
	if n <= 0 {
		panic("n must be greater than 0")
	}

	var chunks [][]T
	for i := 0; i < len(slice); i += n {
		end := min(i+n, len(slice))
		chunks = append(chunks, slice[i:end])
	}

	return chunks
}
