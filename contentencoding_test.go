package contentencoding_test

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	contentencoding "github.com/johejo/go-content-encoding"
)

func TestDecode_compress(t *testing.T) {
	tests := []struct {
		name     string
		encoding string
		data     string
	}{
		{"brotli", "br", "testdata/test.txt.br"},
		{"gzip", "gzip", "testdata/test.txt.gz"},
		{"zstd", "zstd", "testdata/test.txt.zst"},
		{"gzip+zstd", "gzip, zstd", "testdata/test.txt.gz.zst"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := http.NewServeMux()
			mux.Handle("/", contentencoding.Decode()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				b, err := ioutil.ReadAll(r.Body)
				if err != nil {
					t.Fatal(err)
				}
				txt := strings.TrimSpace(string(b))
				if txt != "test" {
					t.Errorf("should be test but got='%s'", txt)
				}
			})))

			f, err := os.Open(tt.data)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { f.Close() })

			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/", f)
			req.Header.Set("Content-Encoding", tt.encoding)
			mux.ServeHTTP(rec, req)

			result := rec.Result()
			if result.StatusCode != http.StatusOK {
				t.Errorf("%v", result)
			}
			want := "br, gzip, zstd"
			got := result.Header.Get("Accept-Encoding")
			if want != got {
				t.Errorf("invalid Accept-Encoding, want=%s, got=%s", want, got)
			}
		})
	}
}

func TestDecode_WithDecoder(t *testing.T) {
	customDecoder := &contentencoding.Decoder{
		Encoding: "custom",
		Handler: func(w http.ResponseWriter, r *http.Request) error {
			b, err := ioutil.ReadAll(r.Body)
			if err != nil {
				return err
			}
			r.Body = ioutil.NopCloser(strings.NewReader(string(b) + "-custom"))
			return nil
		},
	}
	mux := http.NewServeMux()
	dm := contentencoding.Decode(contentencoding.WithDecoder(customDecoder))
	mux.Handle("/", dm(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, err := ioutil.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		txt := strings.TrimSpace(string(b))
		if txt != "test-custom" {
			t.Errorf("should be test but got='%s'", txt)
		}
	})))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("test"))
	req.Header.Set("Content-Encoding", "custom")
	mux.ServeHTTP(rec, req)
	result := rec.Result()
	if result.StatusCode != http.StatusOK {
		t.Errorf("%v", result)
	}
	want := "br, gzip, zstd, custom"
	got := result.Header.Get("Accept-Encoding")
	if want != got {
		t.Errorf("invalid Accept-Encoding, want=%s, got=%s", want, got)
	}
}

func TestDecode_WithErrorHandler(t *testing.T) {
	mux := http.NewServeMux()
	errHandler := contentencoding.ErrorHandler(func(w http.ResponseWriter, r *http.Request, err error) {
		w.WriteHeader(999) // custom error code
	})
	dm := contentencoding.Decode(contentencoding.WithErrorHandler(errHandler))
	mux.Handle("/", dm(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("test")) // not compressed
	req.Header.Set("Content-Encoding", "gzip")
	mux.ServeHTTP(rec, req)
	result := rec.Result()
	if result.StatusCode != 999 {
		t.Errorf("invalid Accept-Encoding, %v", result)
	}
}

func TestDecode_mergeAcceptEncoding(t *testing.T) {
	mux := http.NewServeMux()
	outer := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Accept-Encoding", "gzip")
			next.ServeHTTP(w, r)
		})
	}
	dm := contentencoding.Decode()
	mux.Handle("/", outer(dm(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))))
	rec := httptest.NewRecorder()
	f, err := os.Open("testdata/test.txt.gz")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { f.Close() })
	req := httptest.NewRequest(http.MethodPost, "/", f) // not compressed
	req.Header.Set("Content-Encoding", "gzip")
	mux.ServeHTTP(rec, req)
	result := rec.Result()
	if result.Header.Get("Accept-Encoding") != "br, gzip, zstd" {
		t.Errorf("invalid Accept-Encoding, %v", result)
	}
}
