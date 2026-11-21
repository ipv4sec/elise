package totp

import (
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"hash"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode"
)

type Account struct {
	Name      string
	Secret    []byte
	Algorithm string
	Digits    int
	Period    int
}

// Parse accepts a Base32 key or an authenticator's otpauth://totp URI.
func Parse(input, name string) (Account, error) {
	input = strings.TrimSpace(input)
	account := Account{Name: strings.TrimSpace(name), Algorithm: "SHA1", Digits: 6, Period: 30}
	secret := input

	if strings.Contains(input, "://") {
		uri, err := url.Parse(input)
		if err != nil || !strings.EqualFold(uri.Scheme, "otpauth") || uri.User != nil ||
			!strings.EqualFold(uri.Host, "totp") || uri.Fragment != "" {
			return Account{}, fmt.Errorf("请输入 otpauth://totp 链接或 Base32 密钥")
		}

		params, err := url.ParseQuery(uri.RawQuery)
		if err != nil {
			return Account{}, fmt.Errorf("TOTP 链接的参数格式不正确")
		}
		for _, key := range []string{"secret", "issuer", "algorithm", "digits", "period"} {
			if len(params[key]) > 1 {
				return Account{}, fmt.Errorf("TOTP 链接中的 %s 参数重复", key)
			}
		}

		if account.Name == "" {
			label := strings.TrimSpace(strings.TrimPrefix(uri.Path, "/"))
			issuer := strings.TrimSpace(params.Get("issuer"))
			prefix, _, qualified := strings.Cut(label, ":")
			if issuer != "" && label != "" && !(qualified && strings.TrimSpace(prefix) == issuer) {
				label = issuer + ": " + label
			}
			if label == "" {
				label = issuer
			}
			account.Name = label
		}

		secret = params.Get("secret")
		if algorithm := params.Get("algorithm"); algorithm != "" {
			account.Algorithm = strings.ToUpper(algorithm)
		}
		for key, target := range map[string]*int{"digits": &account.Digits, "period": &account.Period} {
			if params.Has(key) {
				number, err := strconv.Atoi(params.Get(key))
				if err != nil {
					return Account{}, fmt.Errorf("TOTP 链接中的 %s 必须是整数", key)
				}
				*target = number
			}
		}
	}

	secret = strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		if r >= 'a' && r <= 'z' {
			return r - ('a' - 'A')
		}
		return r
	}, secret)

	encoding := base32.StdEncoding.WithPadding(base32.NoPadding)
	unpadded := strings.TrimRight(secret, "=")
	decoded, err := encoding.DecodeString(unpadded)
	if err != nil || len(decoded) == 0 || encoding.EncodeToString(decoded) != unpadded ||
		(secret != unpadded && base32.StdEncoding.EncodeToString(decoded) != secret) {
		return Account{}, fmt.Errorf("请输入有效的 Base32 密钥")
	}
	account.Secret = decoded
	if err := account.Validate(); err != nil {
		return Account{}, err
	}
	return account, nil
}

func (a Account) Validate() error {
	if strings.TrimSpace(a.Name) == "" || strings.IndexFunc(a.Name, unicode.IsControl) >= 0 {
		return fmt.Errorf("请输入有效的账户名称")
	}
	if len(a.Secret) == 0 {
		return fmt.Errorf("账户缺少密钥")
	}
	if a.hash() == nil {
		return fmt.Errorf("仅支持 SHA1、SHA256 和 SHA512")
	}
	if a.Digits != 6 && a.Digits != 8 {
		return fmt.Errorf("验证码位数必须为 6 或 8")
	}
	if a.Period <= 0 || a.Period > 86400 {
		return fmt.Errorf("刷新周期必须为 1 到 86400 秒")
	}
	return nil
}

func (a Account) hash() func() hash.Hash {
	switch a.Algorithm {
	case "SHA1":
		return sha1.New
	case "SHA256":
		return sha256.New
	case "SHA512":
		return sha512.New
	default:
		return nil
	}
}

// Code returns the current code.
// The account must have passed Validate.
func (a Account) Code(now time.Time) string {
	seconds := now.Unix()
	if seconds < 0 {
		return ""
	}
	period := int64(a.Period)

	var counter [8]byte
	binary.BigEndian.PutUint64(counter[:], uint64(seconds/period))
	mac := hmac.New(a.hash(), a.Secret)
	mac.Write(counter[:])
	digest := mac.Sum(nil)
	offset := digest[len(digest)-1] & 0x0f
	value := binary.BigEndian.Uint32(digest[offset:offset+4]) & 0x7fffffff
	modulus := uint32(1000000)
	if a.Digits == 8 {
		modulus = 100000000
	}

	return fmt.Sprintf("%0*d", a.Digits, value%modulus)
}
