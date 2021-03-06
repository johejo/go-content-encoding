// Package contentencoding provides net/http compatible middleware for HTTP Content-Encoding.
// It also provides the functionality to customize the decoder.
// By default, br(brotli), gzip and zstd(zstandard) are supported.
package contentencoding

import (
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/andybalholm/brotli"
	"github.com/klauspost/compress/gzip"
	"github.com/klauspost/compress/zstd"
)

// Decode returns net/http compatible middleware that automatically decodes body detected by Content-Encoding.
// By default, br(brotli), gzip and zstd(zstandard) are supported.
func Decode(opts ...Option) func(next http.Handler) http.Handler {
	cfg := new(config)
	for _, opt := range append(defaults(), opts...) {
		opt(cfg)
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodGet || r.Method == http.MethodHead {
				next.ServeHTTP(w, r)
				return
			}
			values := splitEncodingHeader(r.Header.Get("Content-Encoding"))
			for i := len(values) - 1; i >= 0; i-- {
				v := values[i]
				switch v {
				case "br":
					decompressBrotli(r)
				case "gzip", "x-gzip":
					if err := decompressGzip(r); err != nil {
						cfg.errHandler(w, r, err)
						return
					}
				case "zstd":
					if err := decompressZstd(r, cfg.dopts...); err != nil {
						cfg.errHandler(w, r, err)
						return
					}
				case "", "identity":
				default:
					for _, decoder := range cfg.decoders {
						if v == decoder.Encoding {
							if err := decoder.Handler(w, r); err != nil {
								cfg.errHandler(w, r, err)
								return
							}
						}
					}
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

func decompressBrotli(r *http.Request) {
	r.Body = ioutil.NopCloser(brotli.NewReader(r.Body))
}

func decompressGzip(r *http.Request) error {
	gr, err := gzip.NewReader(r.Body)
	if err != nil {
		return err
	}
	r.Body = gr
	return nil
}

func decompressZstd(r *http.Request, opts ...zstd.DOption) error {
	zr, err := zstd.NewReader(r.Body, opts...)
	if err != nil {
		return err
	}
	r.Body = ioutil.NopCloser(zr)
	return nil
}

var noSpace = strings.NewReplacer(" ", "")

func splitEncodingHeader(raw string) []string {
	if raw == "" {
		return []string{}
	}
	return strings.Split(noSpace.Replace(raw), ",")
}

// Option is option for Decode.
type Option func(cfg *config)

type config struct {
	errHandler ErrorHandler
	decoders   []*Decoder

	dopts []zstd.DOption
}

// DefaultErrorHandler is ErrorHandler that will used by default.
func DefaultErrorHandler(w http.ResponseWriter, r *http.Request, err error) {
	http.Error(w, err.Error(), http.StatusBadRequest)
}

// ErrorHandler is a type used to customize error handling.
type ErrorHandler func(w http.ResponseWriter, r *http.Request, err error)

// WithErrorHandler returns a Option to customize error handling.
func WithErrorHandler(eh ErrorHandler) Option {
	if eh == nil {
		eh = DefaultErrorHandler
	}
	return func(cfg *config) {
		cfg.errHandler = eh
	}
}

// WithDOptions returns a Option to customize zstd decoder with zstd.DOptions.
// See https://pkg.go.dev/github.com/klauspost/compress/zstd?tab=doc#DOption.
func WithDOptions(dopts ...zstd.DOption) Option {
	return func(cfg *config) {
		cfg.dopts = dopts
	}
}

// Decoder is custom decoder for user defined Content-Encoding.
// If the Content-Encoding matches Encoding, Handler is called.
type Decoder struct {
	// Encoding is a string used for Content-Encoding matching.
	Encoding string
	// Handler will be called when Encoding matches the Content-Encoding.
	Handler func(w http.ResponseWriter, r *http.Request) error
}

// WithDecoder returns a Option to use Decode with Decoder.
func WithDecoder(decoders ...*Decoder) Option {
	return func(cfg *config) {
		cfg.decoders = decoders
	}
}

func defaults() []Option {
	return []Option{
		WithErrorHandler(nil),
	}
}
