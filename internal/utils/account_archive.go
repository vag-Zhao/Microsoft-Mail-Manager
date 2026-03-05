package utils

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"
)

var accountArchiveMagic = []byte{'Z', 'G', 'S', 'A', 'C', 'C'}

const accountArchiveVersion byte = 1

type accountArchiveHeader struct {
	ExportedAt string `json:"exportedAt"`
	LineCount  int    `json:"lineCount"`
	Codec      string `json:"codec"`
}

func EncodeAccountArchive(plain string) ([]byte, error) {
	header := accountArchiveHeader{
		ExportedAt: time.Now().Format(time.RFC3339),
		LineCount:  countNonEmptyLines(plain),
		Codec:      "gzip",
	}
	headerBytes, err := json.Marshal(header)
	if err != nil {
		return nil, err
	}

	var payload bytes.Buffer
	gz := gzip.NewWriter(&payload)
	if _, err := gz.Write([]byte(plain)); err != nil {
		_ = gz.Close()
		return nil, err
	}
	if err := gz.Close(); err != nil {
		return nil, err
	}

	var out bytes.Buffer
	out.Write(accountArchiveMagic)
	out.WriteByte(accountArchiveVersion)

	lenBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(lenBuf, uint32(len(headerBytes)))
	out.Write(lenBuf)
	out.Write(headerBytes)
	out.Write(payload.Bytes())
	return out.Bytes(), nil
}

func DecodeAccountArchive(data []byte) (string, error) {
	if len(data) < len(accountArchiveMagic)+1+4 {
		return "", fmt.Errorf("账号数据文件格式无效或版本不支持")
	}
	if !bytes.Equal(data[:len(accountArchiveMagic)], accountArchiveMagic) {
		return "", fmt.Errorf("账号数据文件格式无效或版本不支持")
	}
	if data[len(accountArchiveMagic)] != accountArchiveVersion {
		return "", fmt.Errorf("账号数据文件格式无效或版本不支持")
	}

	headerLenOffset := len(accountArchiveMagic) + 1
	headerLen := binary.BigEndian.Uint32(data[headerLenOffset : headerLenOffset+4])
	payloadOffset := headerLenOffset + 4 + int(headerLen)
	if payloadOffset > len(data) {
		return "", fmt.Errorf("账号数据文件已损坏")
	}

	var header accountArchiveHeader
	if err := json.Unmarshal(data[headerLenOffset+4:payloadOffset], &header); err != nil {
		return "", fmt.Errorf("账号数据文件已损坏")
	}
	if header.Codec != "gzip" {
		return "", fmt.Errorf("账号数据文件格式无效或版本不支持")
	}

	gz, err := gzip.NewReader(bytes.NewReader(data[payloadOffset:]))
	if err != nil {
		return "", fmt.Errorf("账号数据文件已损坏")
	}
	defer gz.Close()

	plain, err := io.ReadAll(gz)
	if err != nil {
		return "", fmt.Errorf("账号数据文件已损坏")
	}
	return string(plain), nil
}

func IsAccountArchive(data []byte) bool {
	if len(data) < len(accountArchiveMagic)+1 {
		return false
	}
	if !bytes.Equal(data[:len(accountArchiveMagic)], accountArchiveMagic) {
		return false
	}
	return data[len(accountArchiveMagic)] == accountArchiveVersion
}

func countNonEmptyLines(text string) int {
	if strings.TrimSpace(text) == "" {
		return 0
	}
	lines := strings.Split(text, "\n")
	count := 0
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			count++
		}
	}
	return count
}
