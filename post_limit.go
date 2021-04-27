package middleware

import (
	"net/http"

	post_limit "github.com/moisespsena-go/http-post-limit"
)

type PostLimitFailedHandler = http.HandlerFunc

var ParseForm = post_limit.ParseForm

var DefaultPostLimitFailedHandler = PostLimitFailedHandler(func(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "max post size exceeded", http.StatusBadRequest)
})

func PostLimit(maxPostSize int64, failedHandler ...PostLimitFailedHandler) func(next http.Handler) http.Handler {
	var fh PostLimitFailedHandler
	for _, fh = range failedHandler {
	}
	return func(next http.Handler) http.Handler {
		return post_limit.New(next, &post_limit.Opts{
			MaxPostSize:  maxPostSize,
			ErrorHandler: fh,
		})
	}
}
