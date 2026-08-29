package headers

import (
	"bytes"
	"fmt"
	"strings"
)

type Headers map[string]string

func NewHeaders() Headers {
	return make(Headers)
}

// isTokenChar reports whether b is a valid RFC 9112 tchar,
// i.e. a character allowed in a field-name.
func isTokenChar(b byte) bool {
	switch {
	case b >= 'A' && b <= 'Z':
		return true
	case b >= 'a' && b <= 'z':
		return true
	case b >= '0' && b <= '9':
		return true
	}
	switch b {
	case '!', '#', '$', '%', '&', '\'', '*', '+', '-', '.', '^', '_', '`', '|', '~':
		return true
	}
	return false
}

func isValidFieldName(name []byte) bool {
	if len(name) == 0 {
		return false
	}
	for _, b := range name {
		if !isTokenChar(b) {
			return false
		}
	}
	return true
}

// Get returns the value for key, treating the header name as
// case-insensitive. Returns "" if the header isn't present.
func (h Headers) Get(key string) string {
	return h[strings.ToLower(key)]
}

// Set assigns value to key, overwriting any existing value for that key
// (unlike the comma-appending behavior used when parsing duplicate
// headers off the wire). Case-insensitive, like Get.
func (h Headers) Set(key, value string) {
	h[strings.ToLower(key)] = value
}
func (h Headers) Parse(data []byte) (n int, done bool, err error) {
	crlfIdx := bytes.Index(data, []byte("\r\n"))
	if crlfIdx == -1 {
		// Not enough data yet
		return 0, false, nil
	}

	if crlfIdx == 0 {
		// Empty line -> end of headers
		return 2, true, nil
	}

	line := data[:crlfIdx]

	colonIdx := bytes.IndexByte(line, ':')
	if colonIdx == -1 {
		return 0, false, fmt.Errorf("invalid header field-line: missing colon")
	}

	fieldNameBytes := line[:colonIdx]
	// The field-name must have NO whitespace anywhere in it (no OWS before the colon).
	// If trimming changes it, there was leading/trailing space around the name.
	if !bytes.Equal(fieldNameBytes, bytes.TrimSpace(fieldNameBytes)) {
		return 0, false, fmt.Errorf("invalid header field-name: %q", fieldNameBytes)
	}

	if !isValidFieldName(fieldNameBytes) {
		return 0, false, fmt.Errorf("invalid character in header field-name: %q", fieldNameBytes)
	}

	fieldValueBytes := bytes.TrimSpace(line[colonIdx+1:])

	// Field names are case-insensitive; normalize to lowercase for storage/lookup.
	fieldName := strings.ToLower(string(fieldNameBytes))
	fieldValue := string(fieldValueBytes)

	if existing, ok := h[fieldName]; ok {
		h[fieldName] = existing + ", " + fieldValue
	} else {
		h[fieldName] = fieldValue
	}

	return crlfIdx + 2, false, nil
}
