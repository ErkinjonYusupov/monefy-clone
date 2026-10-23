// Package auth Telegram Mini App initData imzosini tekshiradi.
// https://core.telegram.org/bots/webapps#validating-data-received-via-the-mini-app
package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

var (
	ErrNoHash  = errors.New("initData: hash missing")
	ErrBadHash = errors.New("initData: hash mismatch")
	ErrExpired = errors.New("initData: expired")
	ErrNoUser  = errors.New("initData: user missing")
)

type TelegramUser struct {
	ID           int64  `json:"id"`
	FirstName    string `json:"first_name"`
	LastName     string `json:"last_name"`
	Username     string `json:"username"`
	LanguageCode string `json:"language_code"`
}

// Validate imzoni tekshiradi va foydalanuvchini qaytaradi. maxAge <= 0 bo'lsa muddat tekshirilmaydi.
func Validate(initData, botToken string, maxAge time.Duration) (*TelegramUser, error) {
	values, err := verify(initData, botToken, maxAge)
	if err != nil {
		return nil, err
	}
	raw := values.Get("user")
	if raw == "" {
		return nil, ErrNoUser
	}
	var u TelegramUser
	if err := json.Unmarshal([]byte(raw), &u); err != nil {
		return nil, err
	}
	if u.ID == 0 {
		return nil, ErrNoUser
	}
	return &u, nil
}

type Contact struct {
	UserID      int64  `json:"user_id"`
	PhoneNumber string `json:"phone_number"`
	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name"`
}

// ValidateContact WebApp.requestContact() javobini tekshiradi. Telegram uni initData kabi
// bot tokeni bilan imzolaydi, shuning uchun kontakt soxtalashtirilmaganiga ishonsa bo'ladi.
func ValidateContact(response, botToken string, maxAge time.Duration) (*Contact, error) {
	values, err := verify(response, botToken, maxAge)
	if err != nil {
		return nil, err
	}
	var c Contact
	if err := json.Unmarshal([]byte(values.Get("contact")), &c); err != nil {
		return nil, fmt.Errorf("contact: %w", err)
	}
	if c.UserID == 0 || c.PhoneNumber == "" {
		return nil, errors.New("contact: incomplete")
	}
	return &c, nil
}

// verify "k=v&...&hash=..." ko'rinishidagi Telegram ma'lumotining HMAC imzosini tekshiradi.
func verify(data, botToken string, maxAge time.Duration) (url.Values, error) {
	values, err := url.ParseQuery(data)
	if err != nil {
		return nil, err
	}
	hash := values.Get("hash")
	if hash == "" {
		return nil, ErrNoHash
	}

	keys := make([]string, 0, len(values))
	for k := range values {
		if k != "hash" {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	pairs := make([]string, len(keys))
	for i, k := range keys {
		pairs[i] = k + "=" + values.Get(k)
	}
	dataCheck := strings.Join(pairs, "\n")

	secret := hmacSHA256([]byte("WebAppData"), []byte(botToken))
	expected := hex.EncodeToString(hmacSHA256(secret, []byte(dataCheck)))
	if !hmac.Equal([]byte(expected), []byte(hash)) {
		return nil, ErrBadHash
	}

	if maxAge > 0 {
		ts, err := strconv.ParseInt(values.Get("auth_date"), 10, 64)
		if err != nil || time.Since(time.Unix(ts, 0)) > maxAge {
			return nil, ErrExpired
		}
	}
	return values, nil
}

func hmacSHA256(key, msg []byte) []byte {
	m := hmac.New(sha256.New, key)
	m.Write(msg)
	return m.Sum(nil)
}
