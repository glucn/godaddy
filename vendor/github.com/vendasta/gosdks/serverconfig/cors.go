package serverconfig

import (
	"net/http"
	"strings"
)

// CORS wraps a given http.Handler and enables CORS requests for all its methods
func CORS(h http.Handler, allowedMethods ...string) http.Handler {
	if len(allowedMethods) == 0 {
		allowedMethods = []string{http.MethodPost}
	}
	return cors{h, strings.ToUpper(strings.Join(allowedMethods, ","))}
}

type cors struct {
	handler        http.Handler
	allowedMethods string
}

func (c cors) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Access-Control-Allow-Origin", r.Header.Get("Origin"))
	w.Header().Add("Access-Control-Allow-Credentials", "true")
	w.Header().Add("Access-Control-Allow-Headers", "Accept,Authorization,Cache-Control,Content-Type,DNT,If-Modified-Since,Keep-Alive,Origin,User-Agent,X-Requested-With")
	w.Header().Add("Access-Control-Allow-Methods", c.allowedMethods)

	if r.Method == http.MethodOptions {
		w.Header().Add("Access-Control-Max-Age", "1728000")
		w.Header().Add("Content-Type", "text/plain charset=UTF-8")
		w.Header().Add("Content-Length", "0")
		w.WriteHeader(http.StatusNoContent)
		return
	}
	c.handler.ServeHTTP(w, r)
}
