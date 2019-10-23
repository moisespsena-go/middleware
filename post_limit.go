package middleware

import (
	"context"
	"errors"
	"net/http"
)

var (
	PostSizeKey = contextKey{"post-size"}

	MaxPostSize int64 = 1024 * 1024 // 1Mb
)

type FormParser func(r *http.Request) (err error)
type PostLimitFailedHandler = http.HandlerFunc

func ParseForm(r *http.Request) (err error) {
	var maxPostSize = MaxPostSize
	if v := r.Context().Value(PostSizeKey); v != nil {
		if vi := v.(int64); vi > 0 {
			maxPostSize = vi
		}
	}
	switch r.Header.Get("Content-Type") {
	case "application/x-www-form-urlencoded":
		err = r.ParseForm()
	case "multipart/form-data":
		err = r.ParseMultipartForm(maxPostSize)
	default:
		err = errors.New("bad content-type")
	}
	return nil
}

var DefaultPostLimitFailedHandler = PostLimitFailedHandler(func(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "max post size exceeded", http.StatusBadRequest)
})

func PostLimit(maxPostSize int64, failedHandler ...PostLimitFailedHandler) func(next http.Handler) http.Handler {
	if maxPostSize == 0 {
		maxPostSize = MaxPostSize
	}
	var fh PostLimitFailedHandler
	for _, fh = range failedHandler {
	}
	if fh == nil {
		fh = DefaultPostLimitFailedHandler
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodPost, http.MethodPut:
				if r.ContentLength > maxPostSize {
					fh(w, r)
					return
				}
			}
			r = r.WithContext(context.WithValue(r.Context(), PostSizeKey, maxPostSize))
			next.ServeHTTP(w, r)
		})
	}
}
