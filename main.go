package main

import (
	"context"
	"flag"
	"io"
	"log"
	"math/rand"
	"mime"
	"net/http"
	"os"
	"os/signal"
	"path"
	"strings"
	"time"
)

const defaultRootDir = "root"

var rootDir string

func main() {
	rand.Seed(time.Now().Unix())
	mime.AddExtensionType(".webm", "video/webm")
	flateTypes := flag.String("flate", "flate.types", "file containing list of types to deflate")
	flag.StringVar(&rootDir, "r", defaultRootDir, "root directory that contains your folders")
	www := flag.String("w", "www", "directory to serve html from")
	flag.Parse()
	f, err := os.ReadFile(*flateTypes)
	if err != nil {
		shouldCompress = make(map[string]struct{}, 0)
	} else {
		data := strings.Split(string(f), "\n")
		shouldCompress = make(map[string]struct{}, len(data))
		for _, x := range data {
			if len(x) == 0 {
				continue
			}
			if x[len(x)-1] == '\r' {
				x = x[:len(x)-1]
			}
			shouldCompress[x] = struct{}{}
		}
	}
	loadSettings()
	loadHashes()
	hndlr := http.NewServeMux()
	// hndlr.Handle("/login-test", NewAuthRequired(http.NotFoundHandler()))
	hndlr.HandleFunc("/api/1/", apiHandler)
	hndlr.Handle("/api/", http.NotFoundHandler())
	hndlr.Handle("/www/", NewFileReadOnlyHandler(*www, 1))
	indx := path.Join(*www, "index.html")
	hndlr.Handle("/index.html", http.RedirectHandler(indx, http.StatusMovedPermanently))
	hndlr.Handle("/index.htm", http.RedirectHandler(indx, http.StatusMovedPermanently))
	hndlr.Handle("/index", http.RedirectHandler(indx, http.StatusMovedPermanently))
	hndlr.Handle("/favicon.ico", NewSpecificFileHandler(path.Join(*www, "favicon.ico")))
	hndlr.Handle("/", NewImageSortRootMount(rootDir))
	_, err = os.Stat(path.Join(rootDir, "Sort"))
	if err != nil {
		os.Mkdir(path.Join(rootDir, "Sort"), 0600)
	}
	_, err = os.Stat(path.Join(rootDir, "Trash"))
	if err != nil {
		os.Mkdir(path.Join(rootDir, "Trash"), 0600)
	}
	srv := &http.Server{ReadHeaderTimeout: time.Second * 5, Handler: hndlr}
	go srv.ListenAndServe()
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)
	<-ch
	srv.Shutdown(context.Background())
	saveHashes()
	saveSettings()
}

func writeAll(w io.Writer, b []byte) error {
	for len(b) > 0 {
		c, err := w.Write(b)
		if err != nil {
			return err
		}
		b = b[c:]
	}
	return nil
}

type OperationTypes int

const (
	OP_CREATE OperationTypes = iota
	OP_REMOVE
	OP_RECURSIVEREMOVE
	OP_OPEN
	OP_CLOSE
	OP_STAT
	OP_READ
	OP_WRITE
	OP_MARSHALL
	OP_MOVE
	OP_COPY
)

var operStrings []string = []string{"create", "remove", "recursive remove", "open", "close", "stat", "read", "write", "marshall", "move", "copy"}

func logError(err error, op OperationTypes, path string) {
	log.Printf("[ERROR] error %s %s: %s\n", operStrings[op], path, err.Error())
}
