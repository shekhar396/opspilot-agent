package identity

import (
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	identityLength = 36
	loadRetries    = 5
	retryDelay     = 20 * time.Millisecond
)

var errIncompleteIdentity = errors.New("identity content is incomplete")

type Identity struct {
	id string
}

// ID returns the immutable identity value.
func (i Identity) ID() string {
	return i.id
}

// Parse validates a UUIDv4-compatible identity and returns its immutable value.
func Parse(value string) (Identity, error) {
	id := strings.TrimSpace(value)
	if err := validate(id); err != nil {
		return Identity{}, err
	}
	return Identity{id: id}, nil
}

func LoadOrCreate(path string) (Identity, error) {
	if err := validatePath(path); err != nil {
		return Identity{}, err
	}
	path = filepath.Clean(strings.TrimSpace(path))

	loaded, err := loadWithRetry(path)
	if err == nil {
		return loaded, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return Identity{}, err
	}

	parent := filepath.Dir(path)
	parentExisted := true
	if _, err := os.Stat(parent); errors.Is(err, os.ErrNotExist) {
		parentExisted = false
	} else if err != nil {
		return Identity{}, fmt.Errorf("inspect identity directory: %w", err)
	}
	if err := os.MkdirAll(parent, 0o700); err != nil {
		return Identity{}, fmt.Errorf("create identity directory: %w", err)
	}
	if !parentExisted {
		if err := os.Chmod(parent, 0o700); err != nil {
			return Identity{}, fmt.Errorf("set identity directory permissions: %w", err)
		}
	}

	candidate, err := generate()
	if err != nil {
		return Identity{}, fmt.Errorf("generate identity: %w", err)
	}

	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if errors.Is(err, os.ErrExist) {
		return loadWithRetry(path)
	}
	if err != nil {
		return Identity{}, fmt.Errorf("create identity file: %w", err)
	}

	removeOnError := true
	defer func() {
		if removeOnError {
			_ = os.Remove(path)
		}
	}()

	if err := file.Chmod(0o600); err != nil {
		_ = file.Close()
		return Identity{}, fmt.Errorf("set identity file permissions: %w", err)
	}
	content := candidate.id + "\n"
	if n, err := io.WriteString(file, content); err != nil {
		_ = file.Close()
		return Identity{}, fmt.Errorf("write identity file: %w", err)
	} else if n != len(content) {
		_ = file.Close()
		return Identity{}, fmt.Errorf("write identity file: %w", io.ErrShortWrite)
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return Identity{}, fmt.Errorf("sync identity file: %w", err)
	}
	if err := file.Close(); err != nil {
		return Identity{}, fmt.Errorf("close identity file: %w", err)
	}

	removeOnError = false
	return candidate, nil
}

func loadWithRetry(path string) (Identity, error) {
	for attempt := 0; ; attempt++ {
		loaded, err := load(path)
		if !errors.Is(err, errIncompleteIdentity) || attempt == loadRetries {
			return loaded, err
		}
		time.Sleep(retryDelay)
	}
}

func load(path string) (Identity, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return Identity{}, fmt.Errorf("read identity file: %w", err)
	}

	id := strings.TrimSpace(string(content))
	if len(id) < identityLength {
		return Identity{}, fmt.Errorf("validate identity: %w", errIncompleteIdentity)
	}
	parsed, err := Parse(id)
	if err != nil {
		return Identity{}, fmt.Errorf("validate identity: %w", err)
	}
	return parsed, nil
}

func generate() (Identity, error) {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		return Identity{}, err
	}
	value[6] = (value[6] & 0x0f) | 0x40
	value[8] = (value[8] & 0x3f) | 0x80

	id := fmt.Sprintf("%x-%x-%x-%x-%x", value[0:4], value[4:6], value[6:8], value[8:10], value[10:16])
	return Identity{id: id}, nil
}

func validate(id string) error {
	if len(id) != identityLength {
		return fmt.Errorf("identity must be exactly %d characters", identityLength)
	}
	for position, character := range []byte(id) {
		switch position {
		case 8, 13, 18, 23:
			if character != '-' {
				return fmt.Errorf("identity has an invalid separator")
			}
		default:
			if !isLowerHex(character) {
				return fmt.Errorf("identity contains a non-lowercase-hexadecimal character")
			}
		}
	}
	if id[14] != '4' {
		return fmt.Errorf("identity must be UUID version 4")
	}
	if !strings.ContainsRune("89ab", rune(id[19])) {
		return fmt.Errorf("identity has an invalid UUID variant")
	}
	return nil
}

func validatePath(path string) error {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return fmt.Errorf("identity path is required")
	}
	if strings.ContainsRune(trimmed, '\x00') {
		return fmt.Errorf("identity path must not contain a null byte")
	}
	if !filepath.IsAbs(trimmed) {
		return fmt.Errorf("identity path must be absolute")
	}
	if strings.HasSuffix(trimmed, string(os.PathSeparator)) {
		return fmt.Errorf("identity path must not end with a path separator")
	}
	cleaned := filepath.Clean(trimmed)
	if cleaned == "." || cleaned == string(os.PathSeparator) {
		return fmt.Errorf("identity path must identify a file")
	}
	return nil
}

func isLowerHex(character byte) bool {
	return character >= '0' && character <= '9' || character >= 'a' && character <= 'f'
}
