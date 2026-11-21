//go:build darwin && cgo

package main

/*
#cgo CFLAGS: -fobjc-arc
#cgo LDFLAGS: -framework AppKit
#include "bridge.h"
*/
import "C"

import (
	"encoding/json"
	"runtime"
	"strings"
	"time"

	"elise/internal/totp"
)

// AppKit invokes all of the callbacks on its main thread.
var (
	accounts []totp.Account
	loaded   bool
)

func init() {
	runtime.LockOSThread()
}

func main() {
	C.EliseRunApp()
}

//export EliseLoad
func EliseLoad() *C.char {
	loaded = false
	saved, err := loadAccounts()
	if err != nil {
		return C.CString(err.Error())
	}
	accounts = saved
	loaded = true
	return nil
}

//export EliseSnapshot
func EliseSnapshot() *C.char {
	type entry struct {
		Name string `json:"name"`
		Code string `json:"code"`
	}
	now := time.Now()
	entries := make([]entry, 0, len(accounts))
	for _, account := range accounts {
		code := account.Code(now)
		entries = append(entries, entry{Name: account.Name, Code: code})
	}
	payload, _ := json.Marshal(entries)
	return C.CString(string(payload))
}

//export EliseCopyCode
func EliseCopyCode(index C.int) *C.char {
	if index < 0 || int(index) >= len(accounts) {
		return nil
	}
	return C.CString(accounts[index].Code(time.Now()))
}

//export EliseAdd
func EliseAdd(input, name *C.char) *C.char {
	account, err := totp.Parse(C.GoString(input), C.GoString(name))
	if err != nil {
		return C.CString(err.Error())
	}
	for _, saved := range accounts {
		if strings.EqualFold(saved.Name, account.Name) {
			return C.CString("这个账户名称已存在。请填写其他名称。")
		}
	}
	next := append(append([]totp.Account(nil), accounts...), account)
	return save(next)
}

//export EliseRemove
func EliseRemove(index C.int) *C.char {
	if index < 0 || int(index) >= len(accounts) {
		return C.CString("找不到此账户。")
	}
	next := make([]totp.Account, 0, len(accounts)-1)
	next = append(next, accounts[:index]...)
	next = append(next, accounts[index+1:]...)
	return save(next)
}

func save(next []totp.Account) *C.char {
	if !loaded {
		return C.CString("请先重新读取本地账户。")
	}
	if err := saveAccounts(next); err != nil {
		return C.CString(err.Error())
	}
	accounts = next
	return nil
}
