package main

import (
	"encoding/json"
	"os"
	"path"
)

type Settings struct {
	HashDiff uint16
	HashSize byte
}

var config Settings

func loadSettings() error {
	data, err := os.ReadFile(path.Join(rootDir, "ImgSort.cfg"))
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &config)
}

func saveSettings() error {
	data, err := json.Marshal(config)
	if err != nil {
		return err
	}
	return os.WriteFile(path.Join(rootDir, "ImgSort.cfg"), data, 0600)
}

func updateSettings(n Settings) {
	n.HashSize &= ^byte(3)
	if n.HashSize > 32 {
		n.HashSize = 32
	} else if n.HashSize < 4 {
		n.HashSize = 4
	}
	if n.HashDiff > (uint16(n.HashSize)*uint16(n.HashSize))/2 {
		n.HashDiff = (uint16(n.HashSize) * uint16(n.HashSize)) / 2
	}
	config = n
}
