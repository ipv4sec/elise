package totp

import (
	"testing"
	"time"
)

func TestCodeRFC6238(t *testing.T) {
	secrets := []string{
		"12345678901234567890",
		"12345678901234567890123456789012",
		"1234567890123456789012345678901234567890123456789012345678901234",
	}
	algorithms := []string{"SHA1", "SHA256", "SHA512"}
	vectors := []struct {
		unix int64
		want [3]string
	}{
		{59, [3]string{"94287082", "46119246", "90693936"}},
		{1111111109, [3]string{"07081804", "68084774", "25091201"}},
		{1111111111, [3]string{"14050471", "67062674", "99943326"}},
		{1234567890, [3]string{"89005924", "91819424", "93441116"}},
		{2000000000, [3]string{"69279037", "90698825", "38618901"}},
		{20000000000, [3]string{"65353130", "77737706", "47863826"}},
	}
	for i, algorithm := range algorithms {
		account := Account{"Test", []byte(secrets[i]), algorithm, 8, 30}
		for _, vector := range vectors {
			got := account.Code(time.Unix(vector.unix, 0))
			if got != vector.want[i] {
				t.Errorf("%s at %d: got %s, want %s", algorithm, vector.unix, got, vector.want[i])
			}
		}
	}
}

func TestParse(t *testing.T) {
	account, err := Parse("otpauth://totp/Work%3AAlice?issuer=Work&secret=mzxw6%3D%3D%3D&algorithm=sha256&digits=8&period=60", "")
	if err != nil {
		t.Fatal(err)
	}
	if account.Name != "Work:Alice" || string(account.Secret) != "foo" ||
		account.Algorithm != "SHA256" || account.Digits != 8 || account.Period != 60 {
		t.Fatalf("unexpected account: %+v", account)
	}
	account, err = Parse("gezd gnbv gy3t qojq gezd gnbv gy3t qojq", " Personal ")
	if err != nil {
		t.Fatal(err)
	}
	if account.Name != "Personal" || account.Algorithm != "SHA1" || account.Digits != 6 || account.Period != 30 {
		t.Fatalf("unexpected default account: %+v", account)
	}
	code := account.Code(time.Unix(59, 0))
	if code != "287082" {
		t.Fatalf("just before rollover: got %s", code)
	}
	next := account.Code(time.Unix(60, 0))
	if next == code {
		t.Fatalf("at rollover: got %s", next)
	}
}

func TestRejectInvalidImport(t *testing.T) {
	for _, input := range []string{
		"ABC!", "M", "MZXW7", "MZXW6=",
		"otpauth://hotp/Work?secret=MZXW6",
		"otpauth://totp/Work?secret=MZXW6&secret=MZXW6",
		"otpauth://totp/Work?secret=MZXW6&digits=7",
		"otpauth://totp/Work?secret=MZXW6&period=0",
		"otpauth://totp/Work?secret=MZXW6&algorithm=MD5",
		"otpauth://totp/Work?secret=%ZZ",
	} {
		if _, err := Parse(input, "Test"); err == nil {
			t.Errorf("accepted invalid input %q", input)
		}
	}
	if _, err := Parse("MZXW6", ""); err == nil {
		t.Error("accepted a bare secret without an account name")
	}
}
