package headers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHeadersParse(t *testing.T) {
	t.Run("Valid single header", func(t *testing.T) {
		headers := NewHeaders()
		data := []byte("Host: localhost:42069\r\n\r\n")
		n, done, err := headers.Parse(data)
		require.NoError(t, err)
		require.NotNil(t, headers)
		assert.Equal(t, "localhost:42069", headers["host"])
		assert.Equal(t, 23, n)
		assert.False(t, done)
	})

	t.Run("Valid single header with extra whitespace", func(t *testing.T) {
		headers := NewHeaders()
		data := []byte("Host:      localhost:42069       \r\n\r\n")
		n, done, err := headers.Parse(data)
		require.NoError(t, err)
		require.NotNil(t, headers)
		assert.Equal(t, "localhost:42069", headers["host"])
		assert.True(t, n > 0)
		assert.False(t, done)
	})

	t.Run("Valid 2 headers with existing headers", func(t *testing.T) {
		headers := NewHeaders()
		headers["host"] = "localhost:42069"
		data := []byte("Content-Length: 42\r\n\r\n")
		n, done, err := headers.Parse(data)
		require.NoError(t, err)
		require.NotNil(t, headers)
		assert.Equal(t, "localhost:42069", headers["host"])
		assert.Equal(t, "42", headers["content-length"])
		assert.True(t, n > 0)
		assert.False(t, done)
	})

	t.Run("Valid done", func(t *testing.T) {
		headers := NewHeaders()
		data := []byte("\r\n\r\n")
		n, done, err := headers.Parse(data)
		require.NoError(t, err)
		assert.Equal(t, 2, n)
		assert.True(t, done)
	})

	t.Run("Invalid spacing header", func(t *testing.T) {
		headers := NewHeaders()
		data := []byte("       Host: localhost:42069\r\n\r\n")
		n, done, err := headers.Parse(data)
		require.Error(t, err)
		assert.Equal(t, 0, n)
		assert.False(t, done)
	})

	t.Run("Valid header with mixed case key normalizes to lowercase", func(t *testing.T) {
		headers := NewHeaders()
		data := []byte("HoSt: localhost:42069\r\n\r\n")
		n, done, err := headers.Parse(data)
		require.NoError(t, err)
		require.NotNil(t, headers)
		assert.Equal(t, "localhost:42069", headers["host"])
		_, capitalizedKeyExists := headers["HoSt"]
		assert.False(t, capitalizedKeyExists)
		assert.True(t, n > 0)
		assert.False(t, done)
	})

	t.Run("Valid 2 headers with different capitalization combine correctly", func(t *testing.T) {
		headers := NewHeaders()
		data := []byte("Set-Person: lane-loves-go\r\n\r\n")
		n, done, err := headers.Parse(data)
		require.NoError(t, err)
		assert.Equal(t, "lane-loves-go", headers["set-person"])
		assert.True(t, n > 0)
		assert.False(t, done)

		data = []byte("SET-PERSON: prime-loves-zig\r\n\r\n")
		n, done, err = headers.Parse(data)
		require.NoError(t, err)
		assert.Equal(t, "lane-loves-go, prime-loves-zig", headers["set-person"])
		assert.True(t, n > 0)
		assert.False(t, done)
	})

	t.Run("Valid header that matches an existing starting header appends comma-separated value", func(t *testing.T) {
		headers := NewHeaders()
		headers["set-person"] = "lane-loves-go"
		data := []byte("Set-Person: prime-loves-zig\r\n\r\n")
		n, done, err := headers.Parse(data)
		require.NoError(t, err)
		require.NotNil(t, headers)
		assert.Equal(t, "lane-loves-go, prime-loves-zig", headers["set-person"])
		assert.True(t, n > 0)
		assert.False(t, done)
	})

	t.Run("Invalid character in header key", func(t *testing.T) {
		headers := NewHeaders()
		data := []byte("H©st: localhost:42069\r\n\r\n")
		n, done, err := headers.Parse(data)
		require.Error(t, err)
		assert.Equal(t, 0, n)
		assert.False(t, done)
	})
}
