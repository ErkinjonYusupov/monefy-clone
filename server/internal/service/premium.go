// Package service biznes qoidalari: premium rejalar va formatlash.
package service

import (
	"fmt"
	"strconv"
	"strings"
)

// Plan premium obuna rejasi. Narxlar UZS tiyinda; Click ulanganda shu jadvaldan olinadi.
type Plan struct {
	Code  string `json:"code"`
	Title string `json:"title"`
	Days  int    `json:"days"` // 0 = umrbod
	Price int64  `json:"price"`
}

var Plans = []Plan{
	{Code: "month", Title: "1 oy", Days: 30, Price: 15_000_00},
	{Code: "year", Title: "1 yil", Days: 365, Price: 120_000_00},
	{Code: "lifetime", Title: "Umrbod", Days: 0, Price: 250_000_00},
}

// GrantOptions admin panelidan beriladigan muddatlar. Narx ommaviy rejadan olinadi
// (bo'lmasa 0); admin to'langan summani o'zi o'zgartirishi mumkin.
var GrantOptions = []Plan{
	{Code: "week", Title: "1 hafta", Days: 7},
	{Code: "month", Title: "1 oy", Days: 30, Price: 15_000_00},
	{Code: "3month", Title: "3 oy", Days: 90},
	{Code: "6month", Title: "6 oy", Days: 180},
	{Code: "year", Title: "1 yil", Days: 365, Price: 120_000_00},
}

// PremiumFeatures Monefy Pro'dagi kabi faqat premium uchun ochiq imkoniyatlar.
var PremiumFeatures = []string{
	"Ko'p valyutali hisoblar va valyuta kurslari",
	"Kategoriyalarni qo'shish, tahrirlash va o'chirish",
	"Ma'lumotlarni CSV faylga eksport qilish",
}

func PlanByCode(code string) (Plan, bool) {
	for _, p := range Plans {
		if p.Code == code {
			return p, true
		}
	}
	return Plan{}, false
}

// GrantByCode admin beradigan muddat yoki ommaviy reja (lifetime ham).
func GrantByCode(code string) (Plan, bool) {
	for _, p := range GrantOptions {
		if p.Code == code {
			return p, true
		}
	}
	return PlanByCode(code)
}

// ParsePlan admin buyrug'idagi rejani tushunadi: "month", "year", "lifetime" yoki "45d".
func ParsePlan(s string) (Plan, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	if p, ok := GrantByCode(s); ok {
		return p, nil
	}
	if n, err := strconv.Atoi(strings.TrimSuffix(s, "d")); err == nil && n > 0 && strings.HasSuffix(s, "d") {
		return Plan{Code: "custom", Title: fmt.Sprintf("%d kun", n), Days: n}, nil
	}
	return Plan{}, fmt.Errorf("noma'lum reja %q (week | month | 3month | 6month | year | lifetime | 30d)", s)
}

// FormatMoney tiyindagi summani "1 250 000,50" ko'rinishiga keltiradi.
func FormatMoney(minor int64) string {
	neg := minor < 0
	if neg {
		minor = -minor
	}
	whole, frac := minor/100, minor%100
	digits := strconv.FormatInt(whole, 10)
	var b strings.Builder
	for i, d := range digits {
		if i > 0 && (len(digits)-i)%3 == 0 {
			b.WriteByte(' ')
		}
		b.WriteRune(d)
	}
	out := b.String()
	if frac != 0 {
		out += fmt.Sprintf(",%02d", frac)
	}
	if neg {
		out = "-" + out
	}
	return out
}

// NormalizePhone raqamni "+998901234567" ko'rinishiga keltiradi.
func NormalizePhone(p string) string {
	digits := strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, p)
	if digits == "" {
		return ""
	}
	return "+" + digits
}
