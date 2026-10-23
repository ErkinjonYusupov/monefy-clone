package handler

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"monefy/internal/auth"
	"monefy/internal/bot"
	"monefy/internal/db"
	"monefy/internal/service"
)

// ---- me ----

type meResponse struct {
	*db.User
	IsPremium     bool `json:"is_premium"`
	IsAdmin       bool `json:"is_admin"`
	PhoneRequired bool `json:"phone_required"`
}

func (h *Handler) meResp(r *http.Request, u *db.User) meResponse {
	return meResponse{User: u, IsPremium: u.IsPremium(), IsAdmin: h.isAdmin(u), PhoneRequired: u.Phone == "" && !isDev(r)}
}

func (h *Handler) me(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.meResp(r, userFrom(r)))
}

// savePhone WebApp.requestContact() javobini qabul qiladi. Imzo bot tokeni bilan tekshiriladi
// va kontakt aynan shu foydalanuvchiniki ekanligi (user_id) solishtiriladi.
func (h *Handler) savePhone(w http.ResponseWriter, r *http.Request) {
	u := userFrom(r)
	var in struct {
		Response string `json:"response"`
	}
	if !decode(w, r, &in) {
		return
	}
	c, err := auth.ValidateContact(in.Response, h.cfg.BotToken, time.Hour)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "bad_contact", "Kontakt tasdiqlanmadi")
		return
	}
	if c.UserID != u.TelegramID {
		writeErr(w, http.StatusForbidden, "not_own_phone", "Faqat o'zingizning raqamingizni ulashing")
		return
	}
	updated, err := h.store.SetPhone(r.Context(), u.TelegramID, service.NormalizePhone(c.PhoneNumber))
	if err != nil {
		storeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, h.meResp(r, updated))
}

func (h *Handler) updateSettings(w http.ResponseWriter, r *http.Request) {
	u := userFrom(r)
	var in struct {
		BaseCurrency *string `json:"base_currency"`
		BudgetAmount *int64  `json:"budget_amount"`
		ClearBudget  bool    `json:"clear_budget"`
	}
	if !decode(w, r, &in) {
		return
	}
	if in.BaseCurrency != nil && !currencyRe.MatchString(*in.BaseCurrency) {
		badRequest(w, "valyuta kodi noto'g'ri")
		return
	}
	if in.BudgetAmount != nil && *in.BudgetAmount <= 0 {
		badRequest(w, "budjet musbat bo'lishi kerak")
		return
	}
	// Bepul tarifda bitta valyuta: asosiy valyuta o'zgarsa, shu valyutadagi hisoblar ham o'zgaradi.
	if in.BaseCurrency != nil && *in.BaseCurrency != u.BaseCurrency && !u.IsPremium() {
		if err := h.store.RenameBaseCurrency(r.Context(), u.ID, u.BaseCurrency, *in.BaseCurrency); err != nil {
			internalErr(w, err)
			return
		}
	}
	updated, err := h.store.UpdateSettings(r.Context(), u.ID, db.UserSettings{
		BaseCurrency: in.BaseCurrency, BudgetAmount: in.BudgetAmount, ClearBudget: in.ClearBudget,
	})
	if err != nil {
		storeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, h.meResp(r, updated))
}

// ---- accounts ----

func (h *Handler) listAccounts(w http.ResponseWriter, r *http.Request) {
	list, err := h.store.ListAccounts(r.Context(), userFrom(r).ID)
	if err != nil {
		internalErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

// checkAccountInput nom va valyutani tekshiradi; boshqa valyuta faqat premium uchun.
// current — tahrirlanayotgan hisobning hozirgi valyutasi (yangi hisobda bo'sh): valyuta o'zgarmasa,
// premium tugagan foydalanuvchi ham USD hisobini qayta nomlashi yoki arxivlashi mumkin.
func checkAccountInput(w http.ResponseWriter, u *db.User, in *db.AccountInput, current string) bool {
	create := current == ""
	if in.Name != nil {
		n := strings.TrimSpace(*in.Name)
		in.Name = &n
	}
	if create && (in.Name == nil || *in.Name == "") {
		badRequest(w, "nom kiritilmagan")
		return false
	}
	if in.Name != nil && (*in.Name == "" || len([]rune(*in.Name)) > 64) {
		badRequest(w, "nom 1–64 belgidan iborat bo'lishi kerak")
		return false
	}
	if create && in.Currency == nil {
		in.Currency = &u.BaseCurrency
	}
	if in.Currency != nil {
		if !currencyRe.MatchString(*in.Currency) {
			badRequest(w, "valyuta kodi noto'g'ri")
			return false
		}
		if *in.Currency != u.BaseCurrency && *in.Currency != current && !u.IsPremium() {
			writeErr(w, http.StatusPaymentRequired, "premium_required", "Boshqa valyutadagi hisob faqat Premium uchun")
			return false
		}
	}
	return true
}

func (h *Handler) createAccount(w http.ResponseWriter, r *http.Request) {
	u := userFrom(r)
	var in db.AccountInput
	if !decode(w, r, &in) || !checkAccountInput(w, u, &in, "") {
		return
	}
	n, err := h.store.CountAccounts(r.Context(), u.ID)
	if err != nil {
		internalErr(w, err)
		return
	}
	if n >= 50 {
		badRequest(w, "hisoblar soni chegarasiga yetildi")
		return
	}
	a, err := h.store.CreateAccount(r.Context(), u.ID, in)
	if err != nil {
		storeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, a)
}

func (h *Handler) updateAccount(w http.ResponseWriter, r *http.Request) {
	u := userFrom(r)
	id, ok := idParam(w, r)
	var in db.AccountInput
	if !ok || !decode(w, r, &in) {
		return
	}
	existing, err := h.store.GetAccount(r.Context(), u.ID, id)
	if err != nil {
		storeErr(w, err)
		return
	}
	if !checkAccountInput(w, u, &in, existing.Currency) {
		return
	}
	a, err := h.store.UpdateAccount(r.Context(), u.ID, id, in)
	if err != nil {
		storeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, a)
}

func (h *Handler) deleteAccount(w http.ResponseWriter, r *http.Request) {
	u := userFrom(r)
	id, ok := idParam(w, r)
	if !ok {
		return
	}
	if n, err := h.store.CountAccounts(r.Context(), u.ID); err != nil {
		internalErr(w, err)
		return
	} else if n <= 1 {
		badRequest(w, "oxirgi hisobni o'chirib bo'lmaydi")
		return
	}
	if err := h.store.DeleteAccount(r.Context(), u.ID, id); err != nil {
		storeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ---- categories ----

var colorRe = regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)

func (h *Handler) listCategories(w http.ResponseWriter, r *http.Request) {
	list, err := h.store.ListCategories(r.Context(), userFrom(r).ID)
	if err != nil {
		internalErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func checkCategoryInput(w http.ResponseWriter, in *db.CategoryInput, create bool) bool {
	if in.Name != nil {
		n := strings.TrimSpace(*in.Name)
		in.Name = &n
		if n == "" || len([]rune(n)) > 64 {
			badRequest(w, "nom 1–64 belgidan iborat bo'lishi kerak")
			return false
		}
	}
	if in.Color != nil && !colorRe.MatchString(*in.Color) {
		badRequest(w, "rang #RRGGBB ko'rinishida bo'lishi kerak")
		return false
	}
	if in.Icon != nil && (*in.Icon == "" || len(*in.Icon) > 40) {
		badRequest(w, "ikonka noto'g'ri")
		return false
	}
	if create {
		if in.Kind == nil || (*in.Kind != "expense" && *in.Kind != "income") {
			badRequest(w, "kind: expense yoki income")
			return false
		}
		if in.Name == nil || in.Icon == nil || in.Color == nil {
			badRequest(w, "name, icon va color majburiy")
			return false
		}
	}
	return true
}

func (h *Handler) createCategory(w http.ResponseWriter, r *http.Request) {
	var in db.CategoryInput
	if !decode(w, r, &in) || !checkCategoryInput(w, &in, true) {
		return
	}
	c, err := h.store.CreateCategory(r.Context(), userFrom(r).ID, in)
	if err != nil {
		storeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, c)
}

func (h *Handler) updateCategory(w http.ResponseWriter, r *http.Request) {
	id, ok := idParam(w, r)
	var in db.CategoryInput
	if !ok || !decode(w, r, &in) || !checkCategoryInput(w, &in, false) {
		return
	}
	in.Kind = nil // turini o'zgartirib bo'lmaydi
	c, err := h.store.UpdateCategory(r.Context(), userFrom(r).ID, id, in)
	if err != nil {
		storeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, c)
}

func (h *Handler) deleteCategory(w http.ResponseWriter, r *http.Request) {
	id, ok := idParam(w, r)
	if !ok {
		return
	}
	if err := h.store.DeleteCategory(r.Context(), userFrom(r).ID, id); err != nil {
		storeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ---- transactions ----

func (h *Handler) listTransactions(w http.ResponseWriter, r *http.Request) {
	p, ok := period(w, r)
	if !ok {
		return
	}
	kind := r.URL.Query().Get("kind")
	if kind != "" && kind != "expense" && kind != "income" {
		badRequest(w, "kind: expense yoki income")
		return
	}
	list, err := h.store.ListTransactions(r.Context(), userFrom(r).ID, db.TxFilter{
		Period: p, AccountID: queryInt(r, "account_id"), CategoryID: queryInt(r, "category_id"),
		Kind: kind, Limit: int(queryInt(r, "limit")),
	})
	if err != nil {
		internalErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

// checkTxInput summa, sana va foydalanuvchiga tegishli hisob/kategoriyani tekshiradi.
func (h *Handler) checkTxInput(w http.ResponseWriter, r *http.Request, in *db.TxInput, create bool) bool {
	u := userFrom(r)
	if create && (in.AccountID == nil || in.CategoryID == nil || in.Amount == nil) {
		badRequest(w, "account_id, category_id va amount majburiy")
		return false
	}
	if in.Amount != nil && (*in.Amount <= 0 || *in.Amount > 1e15) {
		badRequest(w, "summa musbat bo'lishi kerak")
		return false
	}
	if in.OccurredOn != nil && !validDate(*in.OccurredOn) {
		badRequest(w, "sana noto'g'ri (YYYY-MM-DD)")
		return false
	}
	if in.Note != nil && len([]rune(*in.Note)) > 500 {
		badRequest(w, "izoh juda uzun")
		return false
	}
	if in.AccountID != nil {
		if _, err := h.store.GetAccount(r.Context(), u.ID, *in.AccountID); err != nil {
			storeErr(w, err)
			return false
		}
	}
	if in.CategoryID != nil {
		if _, err := h.store.GetCategory(r.Context(), u.ID, *in.CategoryID); err != nil {
			storeErr(w, err)
			return false
		}
	}
	return true
}

func (h *Handler) getTransaction(w http.ResponseWriter, r *http.Request) {
	id, ok := idParam(w, r)
	if !ok {
		return
	}
	t, err := h.store.GetTransaction(r.Context(), userFrom(r).ID, id)
	if err != nil {
		storeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, t)
}

func (h *Handler) createTransaction(w http.ResponseWriter, r *http.Request) {
	var in db.TxInput
	if !decode(w, r, &in) || !h.checkTxInput(w, r, &in, true) {
		return
	}
	t, err := h.store.CreateTransaction(r.Context(), userFrom(r).ID, in)
	if err != nil {
		storeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, t)
}

func (h *Handler) updateTransaction(w http.ResponseWriter, r *http.Request) {
	id, ok := idParam(w, r)
	var in db.TxInput
	if !ok || !decode(w, r, &in) || !h.checkTxInput(w, r, &in, false) {
		return
	}
	t, err := h.store.UpdateTransaction(r.Context(), userFrom(r).ID, id, in)
	if err != nil {
		storeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, t)
}

func (h *Handler) deleteTransaction(w http.ResponseWriter, r *http.Request) {
	id, ok := idParam(w, r)
	if !ok {
		return
	}
	if err := h.store.DeleteTransaction(r.Context(), userFrom(r).ID, id); err != nil {
		storeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ---- transfers ----

func (h *Handler) listTransfers(w http.ResponseWriter, r *http.Request) {
	p, ok := period(w, r)
	if !ok {
		return
	}
	list, err := h.store.ListTransfers(r.Context(), userFrom(r).ID, p)
	if err != nil {
		internalErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (h *Handler) createTransfer(w http.ResponseWriter, r *http.Request) {
	u := userFrom(r)
	var in db.TransferInput
	if !decode(w, r, &in) {
		return
	}
	if in.FromAccountID == in.ToAccountID {
		badRequest(w, "hisoblar har xil bo'lishi kerak")
		return
	}
	if in.AmountFrom <= 0 || in.AmountTo < 0 {
		badRequest(w, "summa musbat bo'lishi kerak")
		return
	}
	if in.OccurredOn != nil && !validDate(*in.OccurredOn) {
		badRequest(w, "sana noto'g'ri (YYYY-MM-DD)")
		return
	}
	from, err := h.store.GetAccount(r.Context(), u.ID, in.FromAccountID)
	if err != nil {
		storeErr(w, err)
		return
	}
	to, err := h.store.GetAccount(r.Context(), u.ID, in.ToAccountID)
	if err != nil {
		storeErr(w, err)
		return
	}
	if from.Currency == to.Currency {
		in.AmountTo = in.AmountFrom
	} else if in.AmountTo == 0 {
		badRequest(w, "valyutalar har xil: amount_to majburiy")
		return
	}
	t, err := h.store.CreateTransfer(r.Context(), u.ID, in)
	if err != nil {
		storeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, t)
}

func (h *Handler) deleteTransfer(w http.ResponseWriter, r *http.Request) {
	id, ok := idParam(w, r)
	if !ok {
		return
	}
	if err := h.store.DeleteTransfer(r.Context(), userFrom(r).ID, id); err != nil {
		storeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ---- summary & rates ----

func (h *Handler) summary(w http.ResponseWriter, r *http.Request) {
	p, ok := period(w, r)
	if !ok {
		return
	}
	s, err := h.store.Summary(r.Context(), userFrom(r), p, queryInt(r, "account_id"))
	if err != nil {
		internalErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, s)
}

func (h *Handler) listRates(w http.ResponseWriter, r *http.Request) {
	list, err := h.store.ListRates(r.Context(), userFrom(r).ID)
	if err != nil {
		internalErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

var rateRe = regexp.MustCompile(`^\d{1,12}(\.\d{1,8})?$`)

func (h *Handler) upsertRate(w http.ResponseWriter, r *http.Request) {
	u := userFrom(r)
	cur := chi.URLParam(r, "currency")
	var in struct {
		Rate string `json:"rate"`
	}
	if !decode(w, r, &in) {
		return
	}
	if !currencyRe.MatchString(cur) || cur == u.BaseCurrency {
		badRequest(w, "valyuta kodi noto'g'ri")
		return
	}
	if !rateRe.MatchString(in.Rate) || strings.Trim(in.Rate, "0.") == "" {
		badRequest(w, "kurs musbat son bo'lishi kerak, masalan 12650.50")
		return
	}
	if err := h.store.UpsertRate(r.Context(), u.ID, cur, in.Rate); err != nil {
		internalErr(w, err)
		return
	}
	h.listRates(w, r)
}

func (h *Handler) deleteRate(w http.ResponseWriter, r *http.Request) {
	if err := h.store.DeleteRate(r.Context(), userFrom(r).ID, chi.URLParam(r, "currency")); err != nil {
		internalErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ---- export ----

// export tranzaksiyalarni CSV qilib foydalanuvchiga bot orqali yuboradi
// (Telegram WebView ichida fayl yuklab olish ishonchsiz).
func (h *Handler) export(w http.ResponseWriter, r *http.Request) {
	u := userFrom(r)
	p, ok := period(w, r)
	if !ok {
		return
	}
	rows, err := h.store.ExportTransactions(r.Context(), u.ID, p)
	if err != nil {
		internalErr(w, err)
		return
	}
	var buf bytes.Buffer
	buf.WriteString("\xEF\xBB\xBF") // Excel UTF-8 ni to'g'ri ochishi uchun
	cw := csv.NewWriter(&buf)
	cw.Comma = ';'
	_ = cw.Write([]string{"Sana", "Turi", "Hisob", "Kategoriya", "Summa", "Valyuta", "Izoh"})
	for _, row := range rows {
		kind := "Xarajat"
		if row.Kind == "income" {
			kind = "Daromad"
		}
		amount := fmt.Sprintf("%d.%02d", row.Amount/100, row.Amount%100)
		_ = cw.Write([]string{row.Date, kind, row.Account, row.Category, amount, row.Currency, row.Note})
	}
	cw.Flush()

	name := "monefy-" + time.Now().Format("2006-01-02") + ".csv"
	caption := fmt.Sprintf("Eksport: %d ta yozuv", len(rows))
	if err := h.bot.API().SendDocument(r.Context(), u.TelegramID, name, buf.Bytes(), caption); err != nil {
		writeErr(w, http.StatusBadGateway, "telegram", "Faylni yuborib bo'lmadi. Botga /start yozing va qayta urinib ko'ring")
		return
	}
	writeJSON(w, http.StatusOK, map[string]int{"rows": len(rows)})
}

// ---- premium ----

func (h *Handler) premiumInfo(w http.ResponseWriter, r *http.Request) {
	u := userFrom(r)
	subs, err := h.store.ListSubscriptions(r.Context(), u.ID)
	if err != nil {
		internalErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"is_premium":    u.IsPremium(),
		"premium_until": u.PremiumUntil,
		"plans":         service.Plans,
		"features":      service.PremiumFeatures,
		"subscriptions": subs,
	})
}

// premiumRequest Click ulanguncha: foydalanuvchi so'rovi admin chatiga boradi.
func (h *Handler) premiumRequest(w http.ResponseWriter, r *http.Request) {
	u := userFrom(r)
	var in struct {
		Plan string `json:"plan"`
	}
	if !decode(w, r, &in) {
		return
	}
	plan, ok := service.PlanByCode(in.Plan)
	if !ok {
		badRequest(w, "reja noto'g'ri")
		return
	}
	h.bot.NotifyAdmin(r.Context(), fmt.Sprintf(
		"💳 Premium so'rovi\n%s\nReja: %s — %s so'm\nHozirgi holat: %s\n\nTasdiqlash:\n<code>/grant %d %s</code>",
		bot.UserLabel(u), plan.Title, service.FormatMoney(plan.Price), bot.PremiumStatus(u), u.TelegramID, plan.Code))
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}
