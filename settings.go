package main

import (
	"encoding/json"
	"os"
	"path"
)

var config struct {
	HashDiff uint16
	HashSize byte
}

func loadSettings() error {
	data, err := os.ReadFile(path.Join(rootDir, "ImgSort.cfg"))
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &config)
}
