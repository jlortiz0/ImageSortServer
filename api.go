package main

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"time"
)

var dupOpers map[int]chan [][2]string

func apiHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodDelete || r.Method == http.MethodPost {
		w.Header().Add("Allow", "GET, PUT")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodPut {
		w.Header().Add("Allow", "GET, PUT, DELETE, POST")
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
		f, err := os.Open(rootDir + path.Join(url[1:]...))
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				w.WriteHeader(http.StatusNotFound)
			} else {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(err.Error()))
			}
			return
		}
		stat, err := f.Stat()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			writeAll(w, []byte(err.Error()))
			return
		}
		if !stat.IsDir() {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("not a directory"))
			return
		}
		fList, _ := f.ReadDir(0)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			writeAll(w, []byte(err.Error()))
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
			// TODO: Filter images based on Accept header
		}
		d, _ := json.Marshal(dList)
		writeAll(w, d)
	case "info":
		if r.Method != http.MethodGet {
			w.Header().Add("Allow", "GET")
			w.WriteHeader(http.StatusMethodNotAllowed)
		} else if len(url) > 1 {
			stat, err := os.Stat(rootDir + strings.Join(url[1:], ","))
			if err != nil {
				if errors.Is(err, os.ErrNotExist) {
					w.WriteHeader(http.StatusNotFound)
				} else {
					w.WriteHeader(http.StatusInternalServerError)
					w.Write([]byte(err.Error()))
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
		// TODO: retrieve from a token
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
		}()
		t := time.NewTicker(time.Second * 5)
		select {
		case <-t.C:
			// TODO: Generate and send token with 202
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
				return
			}
			// TODO: Sanity check, return 422 if failed
			// Maybe some way to get the constraints?
			err = json.Unmarshal(data, &config)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(err.Error()))
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
