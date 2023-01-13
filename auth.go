package main

import (
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
)

type AuthRequired struct {
	hndlr http.Handler
	token uint64
}

func NewAuthRequired(hndlr http.Handler) http.Handler {
	return AuthRequired{hndlr, uint64(rand.Int63())}
}

func (a AuthRequired) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	user, pass, ok := r.BasicAuth()
	if !ok {
		log.Printf("Username: anonymous   Password: %08x", a.token)
		w.Header().Add("WWW-Authenticate", "Basic realm=\"See server console for login\", charset=\"UTF-8\"")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	token, err := strconv.ParseUint(pass, 16, 64)
	if user != "anonymous" || err != nil || token != a.token {
		log.Printf("[WARN] Login FAILED from %s\n", r.RemoteAddr[:strings.IndexByte(r.RemoteAddr, ':')])
		log.Printf("Username: anonymous   Password: %08x", a.token)
		w.Header().Add("WWW-Authenticate", "Basic realm=\"See server console for login\", charset=\"UTF-8\"")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	a.hndlr.ServeHTTP(w, r)
}
