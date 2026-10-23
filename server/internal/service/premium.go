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

// ParsePlan admin buyrug'idagi rejani tushunadi: "month", "year", "lifetime" yoki "45d".
func ParsePlan(s string) (Plan, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	if p, ok := PlanByCode(s); ok {
		return p, nil
	}
	if n, err := strconv.Atoi(strings.TrimSuffix(s, "d")); err == nil && n > 0 && strings.HasSuffix(s, "d") {
		return Plan{Code: "custom", Title: fmt.Sprintf("%d kun", n), Days: n}, nil
	}
	return Plan{}, fmt.Errorf("noma'lum reja %q (month | year | lifetime | 30d)", s)
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
