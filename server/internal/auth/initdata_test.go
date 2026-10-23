package auth

import (
	"encoding/hex"
	"net/url"
	"sort"
	"strconv"
	"testing"
	"time"
)

func sign(t *testing.T, token string, v url.Values) string {
	t.Helper()
	data := ""
	for _, k := range sortedKeys(v) {
		if data != "" {
			data += "\n"
		}
		data += k + "=" + v.Get(k)
	}
	secret := hmacSHA256([]byte("WebAppData"), []byte(token))
	v.Set("hash", hex.EncodeToString(hmacSHA256(secret, []byte(data))))
	return v.Encode()
}

func sortedKeys(v url.Values) []string {
	keys := make([]string, 0, len(v))
	for k := range v {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func TestValidate(t *testing.T) {
	const token = "123:ABC"
	v := url.Values{}
	v.Set("auth_date", strconv.FormatInt(time.Now().Unix(), 10))
	v.Set("query_id", "AAA")
	v.Set("user", `{"id":42,"first_name":"Ali","username":"ali"}`)
	data := sign(t, token, v)

	u, err := Validate(data, token, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if u.ID != 42 || u.Username != "ali" {
		t.Fatalf("unexpected user %+v", u)
	}

	if _, err := Validate(data, "123:WRONG", time.Hour); err != ErrBadHash {
		t.Fatalf("want ErrBadHash, got %v", err)
	}
}

func TestValidateExpired(t *testing.T) {
	const token = "123:ABC"
	v := url.Values{}
	v.Set("auth_date", strconv.FormatInt(time.Now().Add(-48*time.Hour).Unix(), 10))
	v.Set("user", `{"id":1}`)
	if _, err := Validate(sign(t, token, v), token, 24*time.Hour); err != ErrExpired {
		t.Fatalf("want ErrExpired, got %v", err)
	}
}
