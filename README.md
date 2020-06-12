# go-content-encoding

[![ci](https://github.com/johejo/go-content-encoding/workflows/ci/badge.svg?branch=master)](https://github.com/johejo/go-content-encoding/actions?query=workflow%3Aci)
[![go.dev reference](https://img.shields.io/badge/go.dev-reference-007d9c?logo=go&logoColor=white&style=flat-square)](https://pkg.go.dev/github.com/johejo/go-content-encoding)
[![codecov](https://codecov.io/gh/johejo/go-content-encoding/branch/master/graph/badge.svg)](https://codecov.io/gh/johejo/go-content-encoding)
[![Go Report Card](https://goreportcard.com/badge/github.com/johejo/go-content-encoding)](https://goreportcard.com/report/github.com/johejo/go-content-encoding)

## Description

go-content-encoding provides net/http compatible middleware for HTTP Content-Encoding.<br>
It also provides the functionality to customize the decoder.<br>
By default, br(brotli), gzip and zstd(zstandard) are supported.

## Example

```go
package middleware_test

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/johejo/go-content-encoding/middleware"
)

func ExampleDecode() {
	handler := func(w http.ResponseWriter, r *http.Request) {
		b, err := ioutil.ReadAll(r.Body)
		if err != nil {
			panic(err)
		}
		log.Println(b) // decoded body
	}

	mux := http.NewServeMux()
	decode := middleware.Decode()
	mux.Handle("/", decode(http.HandlerFunc(handler)))
}

func ExampleWithDecoder() {
	customDecoder := &middleware.Decoder{
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
	dm := middleware.Decode(middleware.WithDecoder(customDecoder))
	mux.Handle("/", dm(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, err := ioutil.ReadAll(r.Body)
		if err != nil {
			panic(err)
		}
		fmt.Println(string(b))
	})))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("test"))
	req.Header.Set("Content-Encoding", "custom")
	mux.ServeHTTP(rec, req)

	// Output:
	// test-custom
}

func ExampleWithErrorHandler() {
	mux := http.NewServeMux()
	errHandler := middleware.ErrorHandler(func(w http.ResponseWriter, r *http.Request, err error) {
		w.WriteHeader(999) // custom error code
	})
	dm := middleware.Decode(middleware.WithErrorHandler(errHandler))
	mux.Handle("/", dm(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("test")) // not compressed
	req.Header.Set("Content-Encoding", "gzip")
	mux.ServeHTTP(rec, req)
	fmt.Println(rec.Code)

	// Output:
	// 999
}
```


## License

MIT

## Author

Mitsuo Heijo (@johejo)
