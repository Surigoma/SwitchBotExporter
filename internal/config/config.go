package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

func LoadNames(path string) (map[string]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	token, err := decoder.Token()
	if err != nil {
		return nil, err
	}
	if token != json.Delim('{') {
		return nil, errors.New("expected one JSON object")
	}
	names := make(map[string]string)
	for decoder.More() {
		token, err := decoder.Token()
		if err != nil {
			return nil, err
		}
		original, ok := token.(string)
		if !ok {
			return nil, errors.New("expected MAC address")
		}
		mac, valid := configMAC(original)
		if !valid {
			return nil, fmt.Errorf("invalid MAC address %q", original)
		}
		if _, duplicate := names[mac]; duplicate {
			return nil, fmt.Errorf("duplicate MAC address %q", original)
		}
		var name *string
		if err := decoder.Decode(&name); err != nil {
			return nil, err
		}
		if name == nil {
			return nil, fmt.Errorf("expected a display name for %q", original)
		}
		names[mac] = *name
	}
	if _, err := decoder.Token(); err != nil {
		return nil, err
	}
	if _, err := decoder.Token(); !errors.Is(err, io.EOF) {
		if err != nil {
			return nil, err
		}
		return nil, errors.New("unexpected data after name table")
	}
	if len(names) == 0 {
		return nil, errors.New("no devices configured")
	}
	return names, nil
}

func configMAC(input string) (string, bool) {
	value := strings.ToLower(input)
	if len(value) != 17 {
		return "", false
	}
	for i, c := range value {
		if i%3 == 2 {
			if c != ':' {
				return "", false
			}
		} else if !strings.ContainsRune("0123456789abcdef", c) {
			return "", false
		}
	}
	return value, true
}
