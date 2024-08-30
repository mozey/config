package cmdconfig

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"

	"github.com/pkg/errors"
)

// UnmarshalENV .env file bytes to key value map.
// Syntax rules as per this comment
// https://github.com/mozey/config/issues/24#issue-1091975787
func UnmarshalENV(b []byte) (m map[string]string, err error) {
	m = make(map[string]string)

	// Using multi-line mode regex
	// https://stackoverflow.com/a/62996933/639133
	expr := "(?m)^\\s*([_a-zA-Z0-9]+)\\s*=\\s*(.+)\\s*$"
	r, _ := regexp.Compile(expr)
	lines := r.FindAllString(string(b), -1)

	for _, line := range lines {
		key, value, found := strings.Cut(line, "=")
		if !found {
			return m, errors.Errorf("regexp error %s", line)
		}

		// Trim surrounding white space
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)

		// Remove surrounding, quotes inside the value is kept
		value = strings.TrimPrefix(value, "\"")
		value = strings.TrimSuffix(value, "\"")

		m[key] = value
	}

	return m, nil
}

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
