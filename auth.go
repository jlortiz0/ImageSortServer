package main

import (
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
)

type AuthRequired struct {
	hndlr  http.Handler
	token  uint64
	logged bool
}

func NewAuthRequired(hndlr http.Handler) http.Handler {
	return &AuthRequired{hndlr, uint64(rand.Int31()), false}
}

func (a *AuthRequired) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	user, pass, ok := r.BasicAuth()
	if !ok {
		if a.logged {
			log.Printf("[WARN] Forgotten token from %s\n", r.RemoteAddr[:strings.IndexByte(r.RemoteAddr, ':')])
			w.WriteHeader(http.StatusForbidden)
			return
		}
		log.Printf("Username: anonymous   Password: %08x", a.token)
		w.Header().Add("WWW-Authenticate", "Basic realm=\"See server console for login\", charset=\"UTF-8\"")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	token, err := strconv.ParseUint(pass, 16, 64)
	if user != "anonymous" || err != nil || token != a.token {
		if a.logged {
			log.Printf("[WARN] Double login FAILED from %s\n", r.RemoteAddr[:strings.IndexByte(r.RemoteAddr, ':')])
			w.WriteHeader(http.StatusForbidden)
			return
		}
		log.Printf("[WARN] Login FAILED from %s\n", r.RemoteAddr[:strings.IndexByte(r.RemoteAddr, ':')])
		log.Printf("Username: anonymous   Password: %08x", a.token)
		w.Header().Add("WWW-Authenticate", "Basic realm=\"See server console for login\", charset=\"UTF-8\"")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	if !a.logged {
		a.logged = true
		log.Printf("[INFO] Login from %s\n", r.RemoteAddr[:strings.IndexByte(r.RemoteAddr, ':')])
	}
	a.hndlr.ServeHTTP(w, r)
}
