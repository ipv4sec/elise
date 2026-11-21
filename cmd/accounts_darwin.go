//go:build cgo

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"elise/internal/totp"
)

func accountsPath() (string, error) {
	config, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("无法定位本地账户目录：%w", err)
	}
	return filepath.Join(config, "Elise", "accounts.json"), nil
}

func loadAccounts() ([]totp.Account, error) {
	path, err := accountsPath()
	if err != nil {
		return nil, err
	}
	payload, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("无法读取本地账户：%w", err)
	}

	var saved []totp.Account
	if err := json.Unmarshal(payload, &saved); err != nil {
		return nil, fmt.Errorf("本地账户文件格式不正确：%w", err)
	}
	for _, account := range saved {
		if err := account.Validate(); err != nil {
			return nil, fmt.Errorf("本地账户数据无效：%w", err)
		}
	}
	return saved, nil
}

func saveAccounts(accounts []totp.Account) error {
	path, err := accountsPath()
	if err != nil {
		return err
	}
	if len(accounts) == 0 {
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("无法删除本地账户文件：%w", err)
		}
		return nil
	}
	if err := writeAccounts(path, accounts); err != nil {
		return fmt.Errorf("无法保存本地账户：%w", err)
	}
	return nil
}

func writeAccounts(path string, accounts []totp.Account) error {
	parent := filepath.Dir(path)
	if err := os.MkdirAll(parent, 0700); err != nil {
		return err
	}

	// CreateTemp opens the file with mode 0600. Rename publishes it only after
	// the complete update reaches disk.
	file, err := os.CreateTemp(parent, ".accounts-*")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	defer file.Close()
	if err := json.NewEncoder(file).Encode(accounts); err != nil {
		return err
	}
	if err := file.Sync(); err != nil {
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	return os.Rename(file.Name(), path)
}
