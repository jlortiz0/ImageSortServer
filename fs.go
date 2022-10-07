package main

import (
	"errors"
	"io"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"github.com/microsoftarchive/ttlcache"
)

type FileReadOnlyHandler string

func NewFileReadOnlyHandler(rootDir string) http.Handler {
	return FileReadOnlyHandler(rootDir)
}

func (h FileReadOnlyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	loc := path.Join(string(h), r.URL.Path)
	SpecificFileHandler(loc).ServeHTTP(w, r)
}

type SpecificFileHandler string

func NewSpecificFileHandler(path string) http.Handler {
	return SpecificFileHandler(path)
}

func (h SpecificFileHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Add("Allow", "GET")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	f, err := os.Open(string(h))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			w.WriteHeader(http.StatusNotFound)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			writeAll(w, []byte(err.Error()))
		}
		return
	}
	defer f.Close()
	stat, err := f.Stat()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		writeAll(w, []byte(err.Error()))
		return
	}
	w.Header().Add("Last-Modified", stat.ModTime().UTC().Format(time.RFC1123))
	modSince := r.Header.Get("If-Modified-Since")
	if modSince != "" {
		ts, err := time.Parse(time.RFC1123, modSince)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if ts.Equal(stat.ModTime().UTC()) {
			w.WriteHeader(http.StatusNotModified)
			return
		}
	}
	_, err = io.Copy(w, f)
	if err != nil {
		logError(err, OP_COPY, string(h)+"-http")
	}
}

type ImageSortRootMount struct {
	*ttlcache.Cache
	rootDir string
}

func NewImageSortRootMount(path string) http.Handler {
	cache := ttlcache.NewCache(time.Minute * 10)
	return ImageSortRootMount{cache, path}
}

func (i ImageSortRootMount) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path[0] == '/' {
		r.URL.Path = r.URL.Path[1:]
	}
	loc := path.Join(i.rootDir, r.URL.Path)
	switch r.Method {
	case http.MethodGet:
		ind := strings.IndexByte(r.URL.Path, os.PathSeparator)
		for ind != -1 {
			redirect, ok := i.Get(r.URL.Path[:ind])
			if ok {
				w.Header().Add("Location", path.Join(redirect, r.URL.Path[ind+1:]))
				w.WriteHeader(http.StatusMovedPermanently)
				return
			}
			ind2 := strings.IndexByte(r.URL.Path[ind+1:], os.PathSeparator)
			if ind2 == -1 {
				break
			}
			ind += ind2
		}
		SpecificFileHandler(loc).ServeHTTP(w, r)
	case http.MethodPost:
		targetB, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			writeAll(w, []byte(err.Error()))
			logError(err, OP_READ, "http")
			return
		}
		target := path.Join(i.rootDir, string(targetB))
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
		stat2, err := os.Stat(target)
		if stat.IsDir() {
			if err == nil {
				// Should I do a merge in this case?
				w.WriteHeader(http.StatusConflict)
				writeAll(w, []byte("exists"))
			} else if !errors.Is(err, os.ErrNotExist) {
				w.WriteHeader(http.StatusInternalServerError)
				writeAll(w, []byte(err.Error()))
				logError(err, OP_STAT, loc)
			} else {
				err = os.Rename(loc, target)
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					writeAll(w, []byte(err.Error()))
					logError(err, OP_MOVE, loc+" "+target)
				} else {
					i.Set(r.URL.Path, string(targetB))
					w.Header().Add("Location", target)
					w.WriteHeader(http.StatusCreated)
				}
			}
		} else {
			if !stat2.IsDir() {
				w.WriteHeader(http.StatusConflict)
				writeAll(w, []byte("not a directory"))
				return
			}
			// TODO: Name conflict resolution
			os.Rename(loc, target)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				writeAll(w, []byte(err.Error()))
				logError(err, OP_MOVE, loc+" "+target)
			} else {
				i.Set(r.URL.Path, string(targetB))
				w.Header().Add("Location", target)
				w.WriteHeader(http.StatusCreated)
			}
		}
	case "CREATE":
		// A bit odd that this isn't defined by http...
		err := os.Mkdir(loc, 0600)
		if err != nil && !errors.Is(err, os.ErrExist) {
			w.WriteHeader(http.StatusInternalServerError)
			writeAll(w, []byte(err.Error()))
			logError(err, OP_CREATE, loc)
		} else {
			w.WriteHeader(http.StatusNoContent)
		}
	case http.MethodDelete:
		if r.URL.Path == "Trash" {
			err := os.RemoveAll(loc)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				writeAll(w, []byte(err.Error()))
				logError(err, OP_RECURSIVEREMOVE, loc)
			} else {
				w.WriteHeader(http.StatusNoContent)
			}
			// On the off chance that it did actually get removed, I should try to remake it
			os.Mkdir(loc, 0600)
			return
		}
		stat, err := os.Stat(loc)
		if err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				w.WriteHeader(http.StatusInternalServerError)
				writeAll(w, []byte(err.Error()))
				logError(err, OP_STAT, loc)
			} else {
				w.WriteHeader(http.StatusNoContent)
			}
			return
		}
		if stat.IsDir() {
			f, err := os.Open(loc)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				writeAll(w, []byte(err.Error()))
				logError(err, OP_OPEN, loc)
				return
			}
			_, err = f.ReadDir(1)
			if err != io.EOF {
				w.WriteHeader(http.StatusPreconditionFailed)
				f.Close()
				return
			}
			f.Close()
			os.Remove(loc)
			w.WriteHeader(http.StatusNoContent)
			return
		}
		b := path.Base(r.URL.Path)
		err = os.Rename(loc, path.Join(i.rootDir, "Trash", b))
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			writeAll(w, []byte(err.Error()))
			logError(err, OP_MOVE, loc+" Trash")
		} else {
			w.WriteHeader(http.StatusNoContent)
		}
	default:
		w.Header().Add("Allow", "DELETE, POST, GET, CREATE")
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}
