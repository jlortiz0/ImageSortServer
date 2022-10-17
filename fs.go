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

const ENABLE_CACHE = false
const DEFLATE_MIN = 1024
const DEFLATE_USE = true

var shouldCompress map[string]bool

type FileReadOnlyHandler struct {
	rootDir string
	strip   int
}

func NewFileReadOnlyHandler(rootDir string, strip int) http.Handler {
	return FileReadOnlyHandler{rootDir, strip}
}

func (h FileReadOnlyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p := r.URL.Path
	if p[0] == '/' {
		p = p[1:]
	}
	if h.strip != 0 {
		tmp := strings.SplitAfterN(p, "/", h.strip+1)
		p = tmp[len(tmp)-1]
	}
	loc := path.Join(h.rootDir, p)
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
	if stat.IsDir() || !stat.Mode().IsRegular() {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	compress := false
	ind := strings.LastIndexByte(string(h), '.')
	if ind != -1 {
		mtype := mime.TypeByExtension(string(h)[ind:])
		if mtype != "" {
			w.Header().Add("Content-Type", mtype)
			if DEFLATE_USE {
				ind = strings.LastIndexByte(mtype, ';')
				if ind != -1 {
					mtype = mtype[:ind]
				}
				if stat.Size() > DEFLATE_MIN && shouldCompress[mtype] && strings.Contains(r.Header.Get("Accept-Encoding"), "deflate") {
					compress = true
					w.Header().Add("Content-Encoding", "deflate")
				}
			}
		}
	}
	if ENABLE_CACHE {
		w.Header().Add("Cache-Control", "public, max-age=604800")
	} else {
		w.Header().Add("Cache-Control", "no-cache")
	}
	w.Header().Add("Last-Modified", stat.ModTime().UTC().Format(time.RFC1123))
	modSince := r.Header.Get("If-Modified-Since")
	if modSince != "" {
		ts, err := time.Parse(time.RFC1123, modSince)
		if err != nil {
			w.Header().Del("Content-Encoding")
			w.Header().Del("Cache-Control")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if ts.Equal(stat.ModTime().UTC()) {
			w.Header().Del("Content-Encoding")
			w.WriteHeader(http.StatusNotModified)
			return
		}
	}
	if r.Method == http.MethodGet {
		if DEFLATE_USE && compress {
			w2, _ := flate.NewWriter(w, flate.DefaultCompression)
			_, err = io.Copy(w2, f)
			if err == nil {
				err = w2.Flush()
			}
		} else {
			w.Header().Add("Content-Length", strconv.FormatInt(stat.Size(), 10))
			_, err = io.Copy(w, f)
		}
		if err != nil {
			logError(err, OP_COPY, string(h)+"-http")
		}
	} else {
		if !DEFLATE_USE || !compress {
			w.Header().Add("Content-Length", strconv.FormatInt(stat.Size(), 10))
		}
		w.WriteHeader(http.StatusOK)
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
	if r.URL.Path == "" {
		w.Header().Add("Location", "index.html")
		w.WriteHeader(http.StatusMovedPermanently)
		return
	}
	loc := path.Join(i.rootDir, r.URL.Path)
	switch r.Method {
	case http.MethodHead:
		fallthrough
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
			base := path.Base(loc)
			ext := base[strings.LastIndexByte(base, '.'):]
			_, err = os.Stat(path.Join(target, base))
			base = base[:len(base)-len(ext)]
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
			hashes[target] = hashes[r.URL.Path]
			err = os.Rename(loc, path.Join(i.rootDir, target))
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				writeAll(w, []byte(err.Error()))
				logError(err, OP_MOVE, loc+" "+target)
			} else {
				w.Header().Add("Location", "/"+target)
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
		delete(hashes, r.URL.Path)
		ext := b[strings.LastIndexByte(b, '.'):]
		b2 := b[:len(b)-len(ext)]
		_, err = os.Stat(path.Join("Trash", b))
		j := -1
		for err == nil || !errors.Is(err, os.ErrNotExist) {
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				writeAll(w, []byte(err.Error()))
				logError(err, OP_MOVE, loc+" Trash")
				return
			}
			j++
			_, err = os.Stat(fmt.Sprintf("Trash%c%s_%d%s", os.PathSeparator, b2, j, ext))
		}
		if j != -1 {
			b = fmt.Sprintf("%s_%d%s", b2, j, ext)
		}
		err = os.Rename(loc, path.Join(i.rootDir, "Trash", b))
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			writeAll(w, []byte(err.Error()))
			logError(err, OP_MOVE, loc+" Trash")
		} else {
			w.WriteHeader(http.StatusNoContent)
		}
	default:
		w.Header().Add("Allow", "DELETE, POST, GET, CREATE, HEAD")
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}
