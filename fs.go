package main

import (
	"compress/flate"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"time"
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
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Add("Allow", "GET, HEAD")
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
	compress := false
	ind := strings.LastIndexByte(string(h), '.')
	if ind != -1 {
		mtype := mime.TypeByExtension(string(h)[ind:])
		if mtype != "" {
			w.Header().Add("Content-Type", mtype)
			if strings.HasPrefix(mtype, "text/") {
				if strings.Contains(r.Header.Get("Accept-Encoding"), "deflate") {
					compress = true
					w.Header().Add("Content-Type", "deflate")
				}
			}
		}
	}
	w.Header().Add("Cache-Control", "public, max-age=604800")
	w.Header().Add("Content-Length", strconv.FormatInt(stat.Size(), 10))
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
	if r.Method == http.MethodGet {
		if compress {
			w2, _ := flate.NewWriter(w, flate.DefaultCompression)
			_, err = io.Copy(w2, f)
			if err == nil {
				err = w2.Flush()
			}
		} else {
			_, err = io.Copy(w, f)
		}
		if err != nil {
			logError(err, OP_COPY, string(h)+"-http")
		}
	}
}

type ImageSortRootMount struct {
	rootDir string
}

func NewImageSortRootMount(path string) http.Handler {
	return ImageSortRootMount{path}
}

func (i ImageSortRootMount) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path[0] == '/' {
		r.URL.Path = r.URL.Path[1:]
	}
	loc := path.Join(i.rootDir, r.URL.Path)
	switch r.Method {
	case http.MethodGet:
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
					w.Header().Add("Location", target)
					w.WriteHeader(http.StatusCreated)
				}
			}
		} else {
			if err != nil || !stat2.IsDir() {
				w.WriteHeader(http.StatusConflict)
				writeAll(w, []byte("not a directory"))
				return
			}
			// TODO: Name conflict resolution
			base := path.Base(loc)
			ext := base[strings.LastIndexByte(base, '.'):]
			base = base[:len(base)-len(ext)]
			_, err = os.Stat(path.Join(target, base))
			j := -1
			for err == nil || !errors.Is(err, os.ErrNotExist) {
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					writeAll(w, []byte(err.Error()))
					logError(err, OP_MOVE, loc+" "+target)
					return
				}
				j++
				_, err = os.Stat(fmt.Sprintf("%s%c%s_%d%s", target, os.PathSeparator, base, j, ext))
			}
			if j == -1 {
				target = path.Join(string(targetB), path.Base(loc))
			} else {
				target = fmt.Sprintf("%s%c%s_%d%s", string(targetB), os.PathSeparator, base, j, ext)
			}
			err = os.Rename(loc, path.Join(i.rootDir, target))
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				writeAll(w, []byte(err.Error()))
				logError(err, OP_MOVE, loc+" "+target)
			} else {
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
