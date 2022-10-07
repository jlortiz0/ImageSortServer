package main

import (
	"encoding/json"
	"errors"
	"io"
	"math/rand"
	"mime"
	"net/http"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

var dupOpers map[uint64]chan [][2]string = make(map[uint64]chan [][2]string)
var dupLock *sync.Mutex = new(sync.Mutex)

func apiHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodDelete || r.Method == http.MethodPost {
		w.Header().Add("Allow", "GET, PUT")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodPut {
		w.Header().Add("Allow", "GET, PUT, DELETE, POST, CREATE")
		w.WriteHeader(http.StatusNotImplemented)
		return
	}
	url := strings.Split(r.URL.Path, "/")
	if url[0] == "" {
		url = url[3:]
	} else {
		url = url[2:]
	}
	switch url[0] {
	case "list":
		if r.Method != http.MethodGet {
			w.Header().Add("Allow", "GET")
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if len(url) == 1 {
			url = append(url, ".")
		}
		loc := path.Join(rootDir, path.Join(url[1:]...))
		f, err := os.Open(loc)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				w.WriteHeader(http.StatusNotFound)
			} else {
				w.WriteHeader(http.StatusInternalServerError)
				writeAll(w, []byte(err.Error()))
				logError(err, OP_OPEN, loc)
			}
			return
		}
		stat, err := f.Stat()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			writeAll(w, []byte(err.Error()))
			logError(err, OP_STAT, loc)
			return
		}
		if !stat.IsDir() {
			w.WriteHeader(http.StatusBadRequest)
			writeAll(w, []byte("not a directory"))
			return
		}
		fList, _ := f.ReadDir(0)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			writeAll(w, []byte(err.Error()))
			logError(err, OP_READ, loc)
			return
		}
		f.Close()
		dList := make([]string, 0, len(fList))
		if url[1] == "." {
			for _, v := range fList {
				if v.IsDir() {
					s := v.Name()
					if s != "Sort" && s != "Trash" && s != "System Volume Information" && s[0] != '$' && s[0] != '.' {
						dList = append(dList, s)
					}
				}
			}
		} else {
			accept := r.Header.Get("Accept")
			allowed := make([]string, 0, 16)
			for _, x := range strings.Split(accept, ",") {
				if strings.HasPrefix(x, "image/") || strings.HasPrefix(x, "video/") {
					ind := strings.LastIndexByte(x, ';')
					if ind != -1 {
						x = x[:ind]
					}
					tmp, _ := mime.ExtensionsByType(x)
					allowed = append(allowed, tmp...)
				}
			}
			sort.Strings(allowed)
			for _, v := range fList {
				if !v.IsDir() {
					ind := strings.LastIndexByte(v.Name(), '.')
					if ind == -1 {
						continue
					}
					ext := v.Name()[ind:]
					_, ok := sort.Find(len(allowed), func(i int) int {
						return strings.Compare(ext, allowed[i])
					})
					if ok {
						dList = append(dList, v.Name())
					}
				}
			}
		}
		if len(dList) == 0 && len(fList) != 0 {
			w.WriteHeader(http.StatusNotAcceptable)
		} else {
			d, _ := json.Marshal(dList)
			writeAll(w, d)
		}
	case "info":
		if r.Method != http.MethodGet {
			w.Header().Add("Allow", "GET")
			w.WriteHeader(http.StatusMethodNotAllowed)
		} else if len(url) > 1 {
			loc := path.Join(rootDir, path.Join(url[1:]...))
			stat, err := os.Stat(loc)
			if err != nil {
				if errors.Is(err, os.ErrNotExist) {
					w.WriteHeader(http.StatusNotFound)
				} else {
					w.WriteHeader(http.StatusInternalServerError)
					writeAll(w, []byte(err.Error()))
					logError(err, OP_STAT, loc)
				}
				return
			}
			if stat.IsDir() {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			writeAll(w, []byte(strconv.FormatInt(stat.Size(), 10)))
		} else {
			w.WriteHeader(http.StatusBadRequest)
		}
	case "dedup":
		if r.Method != http.MethodGet {
			w.Header().Add("Allow", "GET")
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		tk := r.URL.Query().Get("token")
		if tk != "" {
			token, err := strconv.ParseUint(tk, 16, 64)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				writeAll(w, []byte("invalid token"))
				return
			}
			dupLock.Lock()
			ch, ok := dupOpers[token]
			dupLock.Unlock()
			if !ok {
				w.WriteHeader(http.StatusBadRequest)
				writeAll(w, []byte("unknown token"))
				return
			}
			select {
			case v := <-ch:
				d, _ := json.Marshal(v)
				dupLock.Lock()
				delete(dupOpers, token)
				dupLock.Unlock()
				writeAll(w, d)
			default:
				w.WriteHeader(http.StatusAccepted)
				writeAll(w, []byte(strconv.FormatUint(token, 16)))
			}
			return
		}
		var ls []string
		if len(url) > 1 {
			ls = getDedupList(rootDir + path.Join(url[1:]...))
		} else {
			f, err := os.Open(rootDir)
			if err != nil {
				panic(err)
			}
			entries, err := f.ReadDir(0)
			if err != nil {
				panic(err)
			}
			ls = make([]string, 0, len(entries)<<7)
			for _, fldr := range entries {
				name := fldr.Name()
				if fldr.IsDir() && name != "Trash" && name[0] != '.' && name[0] != '$' {
					for _, x := range getDedupList(rootDir + fldr.Name()) {
						ls = append(ls, path.Join(name, x))
					}
				}
			}
			url = append(url, ".")
		}
		ch := make(chan [][2]string, 1)
		go func() {
			ch <- initDiff(rootDir, ls, path.Join(url[1:]...))
			close(ch)
		}()
		t := time.NewTicker(time.Second * 5)
		select {
		case <-t.C:
			var token uint64
			ok := true
			dupLock.Lock()
			for ok {
				token = uint64(rand.Int63())
				_, ok = dupOpers[token]
			}
			dupOpers[token] = ch
			dupLock.Unlock()
			w.WriteHeader(http.StatusAccepted)
			writeAll(w, []byte(strconv.FormatUint(token, 16)))
		case v := <-ch:
			d, _ := json.Marshal(v)
			writeAll(w, d)
		}
		t.Stop()
	case "settings":
		switch r.Method {
		case http.MethodGet:
			d, _ := json.Marshal(config)
			writeAll(w, d)
		case http.MethodPut:
			data, err := io.ReadAll(r.Body)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				writeAll(w, []byte(err.Error()))
				logError(err, OP_READ, "http")
				return
			}
			// TODO: Sanity check, return 422 if failed
			// Maybe some way to get the constraints?
			err = json.Unmarshal(data, &config)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				writeAll(w, []byte(err.Error()))
			} else {
				w.WriteHeader(http.StatusNoContent)
			}
		default:
			w.Header().Add("Allow", "GET, PUT")
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	default:
		w.WriteHeader(http.StatusNotFound)
		return
	}
}
