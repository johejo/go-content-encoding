package contentencoding_test

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"

	contentencoding "github.com/johejo/go-content-encoding"
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
	decode := contentencoding.Decode()
	mux.Handle("/", decode(http.HandlerFunc(handler)))
}

func ExampleWithDecoder() {
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
	errHandler := contentencoding.ErrorHandler(func(w http.ResponseWriter, r *http.Request, err error) {
		w.WriteHeader(999) // custom error code
	})
	dm := contentencoding.Decode(contentencoding.WithErrorHandler(errHandler))
	mux.Handle("/", dm(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("test")) // not compressed
	req.Header.Set("Content-Encoding", "gzip")
	mux.ServeHTTP(rec, req)
	fmt.Println(rec.Code)

	// Output:
	// 999
}
