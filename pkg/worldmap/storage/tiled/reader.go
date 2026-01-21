package tiled

import (
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

// loadMapFile читает файл с диска и парсит JSON
func loadMapFile(path string) (*Map, error) {
	fileBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("error reading tmj file: %w", err)
	}
	var tmj Map
	if err := json.Unmarshal(fileBytes, &tmj); err != nil {
		return nil, fmt.Errorf("error unmarshalling tmj file: %w", err)
	}
	return &tmj, nil
}

// getClass возвращает класс объекта, обеспечивая совместимость с версиями Tiled < 1.9
func (o Object) getClass() string {
	if o.Class != "" {
		return o.Class
	}
	return o.Type
}

// getVersion пытается распарсить версию как строку, если не выходит — как float
func (v *Tileset) getVersion() string {
	if len(v.Version) == 0 {
		return ""
	}
	// Оптимизация: если начинается с кавычки, это строка
	if v.Version[0] == '"' {
		var s string
		if err := json.Unmarshal(v.Version, &s); err == nil {
			return s
		}
	}
	// Иначе пробуем как число
	var n float64
	if err := json.Unmarshal(v.Version, &n); err == nil {
		return strconv.FormatFloat(n, 'f', -1, 64)
	}
	return string(v.Version)
}

// decodeGIDData — универсальная функция для декодирования данных тайлов.
// Принимает RawMessage и метод компрессии (берется из свойств слоя).
func decodeGIDData(data json.RawMessage, compression string) ([]uint32, error) {
	if len(data) == 0 {
		return nil, nil
	}

	// 1. CSV формат (массив чисел)
	if data[0] == '[' {
		var tiles []uint32
		if err := json.Unmarshal(data, &tiles); err != nil {
			return nil, fmt.Errorf("failed to parse CSV data: %w", err)
		}
		return tiles, nil
	}

	// 2. Base64 формат (строка)
	if data[0] == '"' {
		var base64Str string
		if err := json.Unmarshal(data, &base64Str); err != nil {
			return nil, fmt.Errorf("failed to unmarshal base64 string: %w", err)
		}

		base64Str = strings.TrimSpace(base64Str)
		rawBytes, err := base64.StdEncoding.DecodeString(base64Str)
		if err != nil {
			return nil, fmt.Errorf("base64 decode error: %w", err)
		}

		var reader io.ReadCloser
		byteReader := bytes.NewReader(rawBytes)

		switch compression {
		case "gzip":
			z, err := gzip.NewReader(byteReader)
			if err != nil {
				return nil, fmt.Errorf("gzip init error: %w", err)
			}
			reader = z
		case "zlib":
			z, err := zlib.NewReader(byteReader)
			if err != nil {
				return nil, fmt.Errorf("zlib init error: %w", err)
			}
			reader = z
		case "zstd":
			return nil, fmt.Errorf("zstd not supported")
		case "":
			reader = io.NopCloser(byteReader)
		default:
			return nil, fmt.Errorf("unknown compression: %s", compression)
		}
		defer reader.Close()

		decompressedData, err := io.ReadAll(reader)
		if err != nil {
			return nil, fmt.Errorf("decompression error: %w", err)
		}

		if len(decompressedData)%4 != 0 {
			return nil, fmt.Errorf("data length %d is not divisible by 4", len(decompressedData))
		}

		count := len(decompressedData) / 4
		tiles := make([]uint32, count)

		for i := 0; i < count; i++ {
			tiles[i] = binary.LittleEndian.Uint32(decompressedData[i*4 : i*4+4])
		}

		return tiles, nil
	}

	return nil, fmt.Errorf("unknown data format")
}

// getTileData извлекает GID тайлов, обрабатывая CSV, Base64 и компрессию (zlib/gzip)
func (l *Layer) getTileData() ([]uint32, error) {
	return decodeGIDData(l.Data, l.Compression)
}
