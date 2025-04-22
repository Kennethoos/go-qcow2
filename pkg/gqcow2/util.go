package gqcow2

import "io"

// readAt is a wrapper to save the bilaporate of checking err == EOF situation.
func readAt(r FileHandler, offset int64, length int64) ([]byte, error) {
	// no op
	if length == 0 {
		return nil, nil
	}

	result := make([]byte, length)
	rc, err := r.ReadAt(result, offset)
	if err != nil {
		if int64(rc) == length && err == io.EOF {
			// this is valid situation
		} else {
			return nil, err
		}
	}

	return result, nil
}
