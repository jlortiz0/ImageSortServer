/*
Copyright (C) 2019-2022 jlortiz

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

package main

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"io"
	"os"
	"path"
	"sort"
	"strings"

	"github.com/devedge/imagehash"
)

var hashes map[string]hashEntry

type hashEntry struct {
	hash    []byte
	modTime int64
}

// If cap is 0, the folder really was empty
func getDedupList(fldr string) []string {
	f, err := os.Open(fldr)
	if err != nil {
		panic(err)
	}
	entries, err := f.ReadDir(0)
	if err != nil {
		panic(err)
	}
	ls := make([]string, 0, len(entries))
	for _, v := range entries {
		if !v.IsDir() {
			ind := strings.LastIndexByte(v.Name(), '.')
			if ind == -1 {
				continue
			}
			switch strings.ToLower(v.Name()[ind+1:]) {
			case "mp4":
				fallthrough
			case "webm":
				fallthrough
			case "gif":
				fallthrough
			case "mov":
				fallthrough
			case "bmp":
				fallthrough
			case "jpg":
				fallthrough
			case "png":
				fallthrough
			case "jpeg":
				ls = append(ls, v.Name())
			}
		}
	}
	f.Close()
	sort.Strings(ls)
	return ls
}

func initDiff(rootDir string, ls []string, fldr string) [][2]string {
	diffLs := make([][]byte, len(ls))
	for k, v := range ls {
		if fldr == "" {
			diffLs[k] = getHash(path.Join(rootDir, v))
		} else {
			diffLs[k] = getHash(path.Join(rootDir, fldr, v))
		}
	}
	diffList := make([][2]string, len(diffLs)/32)
	for i, v := range diffLs {
		j := i + 1
		for j < len(diffLs) {
			if compareBits(v, diffLs[j]) <= config.HashDiff {
				diffList = append(diffList, [2]string{ls[i], ls[j]})
			}
			j++
		}
	}
	return diffList
}

func loadHashes() error {
	f, err := os.Open(path.Join(rootDir, "imgSort.cache"))
	if err != nil && errors.Is(err, os.ErrNotExist) {
		hashes = make(map[string]hashEntry, 128)
		return nil
	} else if err != nil {
		return err
	}
	defer f.Close()
	reader := bufio.NewReader(f)
	sz, _ := reader.ReadByte()
	if sz&128 != 0 || sz != config.HashSize {
		hashes = make(map[string]hashEntry, 128)
		return nil
	}
	size := uint16(sz)
	size *= size
	size /= 8
	temp := make([]byte, 4)
	_, err = reader.Read(temp)
	if err != nil {
		return err
	}
	entries := binary.BigEndian.Uint32(temp)
	hashes = make(map[string]hashEntry, entries)
	var s string
	for {
		s, err = reader.ReadString(0)
		if err != nil {
			break
		}
		s = s[:len(s)-1]
		_, err = io.ReadFull(reader, temp)
		if err != nil {
			break
		}
		lModify := int64(binary.BigEndian.Uint32(temp))
		temp2 := make([]byte, size)
		_, err = io.ReadFull(reader, temp2)
		if err != nil {
			break
		}
		hashes[s] = hashEntry{temp2, lModify}
	}
	if err == io.EOF {
		return nil
	}
	return err
}

func saveHashes() error {
	f, err := os.OpenFile(path.Join(rootDir, "imgSort.cache"), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	writer := bufio.NewWriter(f)
	writer.WriteByte(byte(config.HashSize))
	temp := make([]byte, 4)
	binary.BigEndian.PutUint32(temp, uint32(len(hashes)))
	_, err = writer.Write(temp)
	if err != nil {
		return err
	}
	for k, v := range hashes {
		if v.hash == nil {
			continue
		}
		_, err = writer.WriteString(k)
		if err != nil {
			return err
		}
		writer.WriteByte(0)
		binary.BigEndian.PutUint32(temp, uint32(v.modTime))
		_, err = writer.Write(temp)
		if err != nil {
			return err
		}
		_, err = writer.Write(v.hash)
		if err != nil {
			return err
		}
	}
	writer.Flush()
	return nil
}

func getHash(path string) []byte {
	hash, ok := hashes[path]
	if ok {
		info, err := os.Stat(path)
		if err == nil && info.ModTime().Unix() == hash.modTime {
			return hash.hash
		}
	}
	var err error
	var img image.Image
	switch strings.ToLower(path[strings.LastIndexByte(path, '.')+1:]) {
	case "mp4":
		fallthrough
	case "webm":
		fallthrough
	case "gif":
		fallthrough
	case "mov":
		img, err = getVideoFrame(path)
	default:
		img, err = imagehash.OpenImg(path)
	}
	if err != nil {
		fmt.Printf("Could not open %s: %s\n", path, err.Error())
		return nil
	}
	hsh, err := imagehash.DhashHorizontal(img, int(config.HashSize))
	if err != nil {
		fmt.Printf("Could not hash %s: %s\n", path, err.Error())
		return nil
	}
	info, err := os.Stat(path)
	if err != nil {
		return nil
	}
	hashes[path] = hashEntry{hsh, info.ModTime().Unix()}
	return hsh
}

var bitsTable [16]uint16 = [16]uint16{
	0, 1, 1, 2, 1, 2, 2, 3,
	1, 2, 2, 3, 2, 3, 3, 4,
}

func compareBits(x, y []byte) uint16 {
	if len(x) != len(y) {
		return 0xFFFF
	}
	var c uint16
	for i := 0; i < len(x); i++ {
		temp := x[i] ^ y[i]
		c += bitsTable[temp&0xf]
		c += bitsTable[temp>>4]
		if c > config.HashDiff {
			break
		}
	}
	return c
}
