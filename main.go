package main

import (
	"context"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path"
	"time"
)

const defaultRootDir = "jlortiz_TEST"

var rootDir string = defaultRootDir

func main() {
	// TODO: flags
	loadHashes()
	hndlr := http.NewServeMux()
	hndlr.HandleFunc("/api/1", apiHandler)
	hndlr.Handle("/api", http.NotFoundHandler())
	hndlr.Handle("/www", NewFileReadOnlyHandler(path.Join(rootDir, "www")))
	hndlr.Handle("/index.html", http.RedirectHandler("/www/index.html", http.StatusMovedPermanently))
	hndlr.Handle("/index.htm", http.RedirectHandler("/www/index.html", http.StatusMovedPermanently))
	hndlr.Handle("/index", http.RedirectHandler("/www/index.html", http.StatusMovedPermanently))
	hndlr.Handle("/", NewImageSortRootMount(rootDir))
	_, err := os.Stat(path.Join(rootDir, "Sort"))
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
	log.Printf("[%s ERROR] error %s %s: %s\n", time.Now().Format(time.Stamp), operStrings[op], path, err.Error())
}
