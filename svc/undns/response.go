package undns

import dnswire "codeberg.org/miekg/dns"

func packTruncated(response *dnswire.Msg, size int) error {
	original := response.Answer
	if err := response.Pack(); err != nil {
		return err
	}
	if len(response.Data) <= size {
		return nil
	}

	low, high := 0, len(original)
	for low < high {
		middle := (low + high + 1) / 2
		response.Answer = original[:middle]
		response.Truncated = middle < len(original)
		if err := response.Pack(); err != nil {
			return err
		}
		if len(response.Data) <= size {
			low = middle
		} else {
			high = middle - 1
		}
	}
	response.Answer = original[:low]
	response.Truncated = true
	if err := response.Pack(); err != nil {
		return err
	}
	if len(response.Data) > size {
		response.Ns = nil
		response.Extra = nil
		return response.Pack()
	}
	return nil
}
