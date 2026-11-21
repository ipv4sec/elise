//go:build cgo

package main

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"elise/internal/totp"
)

func TestLocalAccountStorage(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	path := filepath.Join(home, "Library", "Application Support", "Elise", "accounts.json")

	saved, err := loadAccounts()
	if err != nil || len(saved) != 0 {
		t.Fatalf("loading an empty installation: got %d accounts, error %v", len(saved), err)
	}

	account, err := totp.Parse("MZXW6", "Test")
	if err != nil {
		t.Fatal(err)
	}
	if err := saveAccounts([]totp.Account{account}); err != nil {
		t.Fatal(err)
	}
	for location, want := range map[string]os.FileMode{path: 0600, filepath.Dir(path): 0700} {
		info, err := os.Stat(location)
		if err != nil {
			t.Fatal(err)
		}
		if got := info.Mode().Perm(); got != want {
			t.Errorf("%q has permissions %04o, want %04o", location, got, want)
		}
	}
	saved, err = loadAccounts()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(saved, []totp.Account{account}) {
		t.Fatal("reloaded account differs from saved account")
	}

	if err := os.WriteFile(path, []byte("invalid"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := loadAccounts(); err == nil {
		t.Error("accepted a malformed accounts file")
	}
	if err := saveAccounts(nil); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("removing the last account left the file in place: %v", err)
	}
}
