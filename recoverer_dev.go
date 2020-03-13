// +build dev

package middleware

import (
	"github.com/moisespsena-go/logging"
	"github.com/moisespsena-go/path-helpers"
)

var reclog = logging.GetOrCreateLogger(path_helpers.GetCalledDir()+".recoverer_panic")

func recovererPanic(r interface{}, stack []byte) {
	reclog.Errorf("%s:\n%s", r, string(stack))
}
