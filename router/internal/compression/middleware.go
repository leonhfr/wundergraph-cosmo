package compression

import (
	"compress/flate"
	"compress/gzip"
	"io"
	"net/http"

	"github.com/andybalholm/brotli"
	"github.com/go-chi/chi/v5/middleware"
)

const (
	compressionLevel = 5
)

// TODO: compressible types parameter?
// TODO: config parameter?
func NewMiddleware() func(next http.Handler) http.Handler {
	compressor := middleware.NewCompressor(compressionLevel)
	compressor.SetEncoder("deflate", encoderDeflate)
	compressor.SetEncoder("gzip", encoderGzip)
	compressor.SetEncoder("br", encoderBrotli)
	return compressor.Handler
}

func encoderBrotli(w io.Writer, level int) io.Writer {
	return brotli.NewWriterV2(w, level)
}

func encoderGzip(w io.Writer, level int) io.Writer {
	gw, err := gzip.NewWriterLevel(w, level)
	if err != nil {
		return nil
	}
	return gw
}

func encoderDeflate(w io.Writer, level int) io.Writer {
	dw, err := flate.NewWriter(w, level)
	if err != nil {
		return nil
	}
	return dw
}
