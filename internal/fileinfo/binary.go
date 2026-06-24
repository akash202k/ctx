package fileinfo

import (
	"bytes"
	"os"
)

var magicNumbers = [][]byte{
	{0xFF, 0xD8, 0xFF},       // JPEG
	{0x89, 0x50, 0x4E, 0x47}, // PNG
	{0x47, 0x49, 0x46, 0x38}, // GIF
	{0x25, 0x50, 0x44, 0x46}, // PDF
	{0x50, 0x4B, 0x03, 0x04}, // ZIP
	{0x52, 0x61, 0x72, 0x21}, // RAR
	{0x1F, 0x8B, 0x08},       // GZIP
}

func IsBinary(path string) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		return false, err
	}
	if info.IsDir() {
		return false, nil
	}

	f, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer f.Close()

	buf := make([]byte, 4096)
	n, err := f.Read(buf)
	if err != nil && err.Error() != "EOF" {
		return false, err
	}
	buf = buf[:n]

	if bytes.IndexByte(buf, 0x00) != -1 {
		return true, nil
	}

	for _, sig := range magicNumbers {
		if len(buf) >= len(sig) && bytes.Equal(buf[:len(sig)], sig) {
			return true, nil
		}
	}

	return false, nil
}
