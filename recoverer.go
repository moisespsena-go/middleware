package middleware

// The original work was derived from Goji's middleware, source:
// https://github.com/zenazn/goji/tree/master/web/middleware

import (
	"fmt"
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
					var (
						errb []byte
						panicEntry PanicEntry
					)

					for _, gpe := range []func(r *http.Request) PanicEntry{gpe, GetPanicEntry, NewPanicEntry} {
						if gpe == nil {
							continue
						}
						if panicEntry = gpe(r); panicEntry != nil {
							break
						}
					}
					if err, ok := rvr.(error); ok {
						errb = error_utils.TraceOf(err)
						if len(errb) == 0 {
							errb = debug.Stack()
						}
					} else {
						if te, ok := rvr.(tracederror.TracedError); ok {
							errb = te.Trace()
						} else {
							errb = debug.Stack()
						}
					}

					var msg = http.StatusText(http.StatusInternalServerError)
					if len(errb) > 0 {
						msg = fmt.Sprintf("<pre>%s\n%s\n\n%s</pre>", msg, fmt.Sprint(rvr), string(errb))
					}
					http.Error(w, msg, http.StatusInternalServerError)

					if len(errb) > 0 {
						go func() {
							recovererPanic(rvr, errb)
							panicEntry.Write(rvr, errb)
						}()
					}
				}
			}()

			next.ServeHTTP(w, r)
		}
		return http.HandlerFunc(fn)
	}
}
