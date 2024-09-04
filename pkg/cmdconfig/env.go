package cmdconfig

import (
	"bytes"
	"fmt"
)

// MarshalENV key value map to .env file bytes
func MarshalENV(c *conf) (b []byte, err error) {
	buf := bytes.NewBufferString("")
	// Assuming c.Keys is already sorted
	for _, key := range c.Keys {
		value, ok := c.Map[key]
		if !ok {
			return b, ErrMissingKey(key)
		}
		_, err = buf.WriteString(fmt.Sprintf("%s=%s\n", key, value))
		if err != nil {
			return b, err
		}
	}
	return buf.Bytes(), nil
}
