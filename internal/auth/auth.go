package auth

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/zalando/go-keyring"
)

const (
	EnvToken     = "PLAINCORD_TOKEN"
	EnvNoKeyring = "PLAINCORD_NO_KEYRING"
	keyService   = "plaincord"
	keyUser      = "discord-token"
)

var ErrInvalidToken = errors.New("token looks invalid")

func ConfigDir() (string, error) {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "plaincord"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "plaincord"), nil
}

func TokenPath() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "token"), nil
}

func ConfigPath() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

func LooksLikeToken(token string) bool {
	value := strings.TrimSpace(token)
	return len(value) >= 50 && !strings.ContainsAny(value, " \n\t")
}

func skipKeyring() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv(EnvNoKeyring)))
	return v == "1" || v == "true" || v == "yes"
}

func LoadToken() (string, error) {
	if env := strings.TrimSpace(os.Getenv(EnvToken)); env != "" {
		return env, nil
	}
	if !skipKeyring() {
		if secret, err := keyring.Get(keyService, keyUser); err == nil && strings.TrimSpace(secret) != "" {
			return strings.TrimSpace(secret), nil
		}
	}
	path, err := TokenPath()
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

func SaveToken(token string) error {
	value := strings.TrimSpace(token)
	if !LooksLikeToken(value) {
		return ErrInvalidToken
	}
	if !skipKeyring() {
		if err := keyring.Set(keyService, keyUser, value); err == nil {
			if path, err := TokenPath(); err == nil {
				_ = os.Remove(path)
			}
			return nil
		}
	}
	path, err := TokenPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(value+"\n"), 0o600)
}

func DeleteToken() error {
	if !skipKeyring() {
		_ = keyring.Delete(keyService, keyUser)
	}
	path, err := TokenPath()
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func LoadApplicationID() (string, error) {
	cfg, err := loadConfig()
	if err != nil {
		return "", err
	}
	id, _ := cfg["application_id"].(string)
	return id, nil
}

func SaveApplicationID(id string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	cfg["application_id"] = id
	return saveConfig(cfg)
}

func loadConfig() (map[string]any, error) {
	path, err := ConfigPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]any{}, nil
		}
		return nil, err
	}
	var cfg map[string]any
	if err := json.Unmarshal(data, &cfg); err != nil {
		return map[string]any{}, nil
	}
	if cfg == nil {
		cfg = map[string]any{}
	}
	return cfg, nil
}

func saveConfig(cfg map[string]any) error {
	path, err := ConfigPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o600)
}
