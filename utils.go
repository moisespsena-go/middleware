package middleware

import (
	"net"
	"net/http"
	"strings"
)

var xForwardedFor = http.CanonicalHeaderKey("X-Forwarded-For")
var xRealIP = http.CanonicalHeaderKey("X-Real-IP")

func GetRealIP(r *http.Request) string {
	var ip string

	if xff := r.Header.Get(xForwardedFor); xff != "" {
		i := strings.Index(xff, ", ")
		if i == -1 {
			i = len(xff)
		}
		ip = xff[:i]
	} else if xrip := r.Header.Get(xRealIP); xrip != "" {
		ip = xrip
	} else {
		ip, _, _ = net.SplitHostPort(r.RemoteAddr)
	}

	return ip
}

type Extensions map[string]bool

func (this Extensions) Update(news ...Extensions) Extensions {
	this = this.Copy()
	for _, v := range news {
		if v != nil {
			for k, v := range v {
				this[k] = v
			}
		}
	}
	return this
}

func (this *Extensions) PtrUpdate(news ...Extensions) *Extensions {
	*this = this.Update(news...)
	return this
}

func (this Extensions) Copy() (m Extensions) {
	m = Extensions{}
	if m != nil {
		for k, v := range this {
			m[k] = v
		}
	}
	return
}

func (this Extensions) UpdateStrings(values ...string) Extensions {
	this = this.Copy()
	for _, v := range values {
		if v != "" {
			switch v[0] {
			case '+':
				this[v[1:]] = true
			case '-':
				this[v[1:]] = false
			default:
				this[v] = true
			}
		}
	}
	return this
}

func (this *Extensions) PtrUpdateStrings(values ...string) *Extensions {
	*this = this.UpdateStrings(values...)
	return this
}

func StringsToExtensions(values ...string) (m Extensions) {
	return m.UpdateStrings(values...)
}
