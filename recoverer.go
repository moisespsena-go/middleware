package middleware

// The original work was derived from Goji's middleware, source:
// https://github.com/zenazn/goji/tree/master/web/middleware

import (
	"net/http"
	"runtime/debug"

	"github.com/moisespsena-go/tracederror"

	error_utils "github.com/unapu-go/error-utils"
)

// Recoverer is a middleware that recovers from panics, logs the panic (and a
// backtrace), and returns a HTTP 500 (Internal Server Error) status if
// possible. Recoverer prints a request BID if one is provided.
func Recoverer(f ...PanicFormatter) func(next http.Handler) http.Handler {
	var gpe func(r *http.Request) PanicEntry
	for _, f := range f {
		gpe = f.NewPanicEntry
		break
	}
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rvr := recover(); rvr != nil {
					http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
					var panicEntry PanicEntry
					for _, gpe := range []func(r *http.Request) PanicEntry{gpe, GetPanicEntry, NewPanicEntry} {
						if gpe == nil {
							continue
						}
						if panicEntry = gpe(r); panicEntry != nil {
							break
						}
					}
					if err, ok := rvr.(error); ok {
						var trace = error_utils.TraceOf(err)
						if trace != nil {
							recovererPanic(err, trace)
							panicEntry.Write(err, trace)
						} else {
							recovererPanic(err, debug.Stack())
							panicEntry.Write(err, debug.Stack())
						}
					} else {
						if te, ok := rvr.(tracederror.TracedError); ok {
							recovererPanic(te.Error(), te.Trace())
							panicEntry.Write(te.Error(), te.Trace())
						} else {
							recovererPanic(rvr, debug.Stack())
							panicEntry.Write(rvr, debug.Stack())
						}
					}
				}
			}()

			next.ServeHTTP(w, r)
		}
		return http.HandlerFunc(fn)
	}
}
