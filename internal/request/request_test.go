package request

import (
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type chunkReader struct {
	data            string
	numBytesPerRead int
	pos             int
}

func (cr *chunkReader) Read(p []byte) (n int, err error) {
	if cr.pos >= len(cr.data) {
		return 0, io.EOF
	}
	endIndex := cr.pos + cr.numBytesPerRead
	if endIndex > len(cr.data) {
		endIndex = len(cr.data)
	}
	n = copy(p, cr.data[cr.pos:endIndex])
	cr.pos += n

	return n, nil
}

func TestRequestFromReader(t *testing.T) {
	rawRequest := "GET / HTTP/1.1\r\nHost: localhost:42069\r\nUser-Agent: curl/7.81.0\r\nAccept: */*\r\n\r\n"
	rawRequestPath := "GET /coffee HTTP/1.1\r\nHost: localhost:42069\r\nUser-Agent: curl/7.81.0\r\nAccept: */*\r\n\r\n"
	rawPostRequest := "POST /coffee HTTP/1.1\r\nHost: localhost:42069\r\nContent-Type: application/json\r\n\r\n{\"flavor\":\"dark mode\"}"

	// Test across varying chunk sizes (from 1 byte up to full length)
	chunkSizes := []int{1, 2, 3, 5, 8, 16, len(rawRequest)}

	for _, chunkSize := range chunkSizes {
		t.Run("Good GET Request line", func(t *testing.T) {
			reader := &chunkReader{
				data:            rawRequest,
				numBytesPerRead: chunkSize,
			}
			r, err := RequestFromReader(reader)
			require.NoError(t, err)
			require.NotNil(t, r)
			assert.Equal(t, "GET", r.RequestLine.Method)
			assert.Equal(t, "/", r.RequestLine.RequestTarget)
			assert.Equal(t, "1.1", r.RequestLine.HttpVersion)
		})
		t.Run("Incomplete request line (EOF before CRLF)", func(t *testing.T) {
			reader := &chunkReader{
				data:            "GET /coffee HTTP/1.1", // no \r\n
				numBytesPerRead: 3,
			}
			_, err := RequestFromReader(reader)
			require.Error(t, err)
		})
		t.Run("Good GET Request line with path", func(t *testing.T) {
			reader := &chunkReader{
				data:            rawRequestPath,
				numBytesPerRead: chunkSize,
			}
			r, err := RequestFromReader(reader)
			require.NoError(t, err)
			require.NotNil(t, r)
			assert.Equal(t, "GET", r.RequestLine.Method)
			assert.Equal(t, "/coffee", r.RequestLine.RequestTarget)
			assert.Equal(t, "1.1", r.RequestLine.HttpVersion)
		})

		t.Run("Good POST Request with path", func(t *testing.T) {
			reader := &chunkReader{
				data:            rawPostRequest,
				numBytesPerRead: chunkSize,
			}
			r, err := RequestFromReader(reader)
			require.NoError(t, err)
			require.NotNil(t, r)
			assert.Equal(t, "POST", r.RequestLine.Method)
			assert.Equal(t, "/coffee", r.RequestLine.RequestTarget)
			assert.Equal(t, "1.1", r.RequestLine.HttpVersion)
		})
	}

	t.Run("Invalid number of parts in request line", func(t *testing.T) {
		reader := &chunkReader{
			data:            "/coffee HTTP/1.1\r\nHost: localhost:42069\r\n\r\n",
			numBytesPerRead: 3,
		}
		_, err := RequestFromReader(reader)
		require.Error(t, err)
	})

	t.Run("Invalid method (out of order) Request line", func(t *testing.T) {
		reader := &chunkReader{
			data:            "/coffee GET HTTP/1.1\r\nHost: localhost:42069\r\n\r\n",
			numBytesPerRead: 4,
		}
		_, err := RequestFromReader(reader)
		require.Error(t, err)
	})

	t.Run("Invalid version in Request line", func(t *testing.T) {
		reader := &chunkReader{
			data:            "GET /coffee HTTP/2.0\r\nHost: localhost:42069\r\n\r\n",
			numBytesPerRead: 2,
		}
		_, err := RequestFromReader(reader)
		require.Error(t, err)
	})
}
