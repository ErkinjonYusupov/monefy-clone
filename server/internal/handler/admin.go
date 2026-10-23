package handler

import (
	"context"
	"net/http"
	"strconv"

	"monefy/internal/db"
	"monefy/internal/service"
)

const adminPageSize = 20

func (h *Handler) adminStats(w http.ResponseWriter, r *http.Request) {
	us, err := h.store.UserStats(r.Context())
	if err != nil {
		internalErr(w, err)
		return
	}
	rs, err := h.store.RevenueStats(r.Context())
	if err != nil {
		internalErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"users": us, "revenue": rs})
}

func (h *Handler) adminUsers(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	q := r.URL.Query().Get("q")
	if len([]rune(q)) > 64 {
		badRequest(w, "qidiruv juda uzun")
		return
	}
	users, total, err := h.store.SearchUsers(r.Context(), q, adminPageSize, (page-1)*adminPageSize)
	if err != nil {
		internalErr(w, err)
		return
	}
	pages := (total + adminPageSize - 1) / adminPageSize
	writeJSON(w, http.StatusOK, map[string]any{
		"items": adminUserList(users), "total": total, "page": page, "pages": max(pages, 1),
	})
}

type adminUserItem struct {
	db.User
	IsPremium bool `json:"is_premium"`
}

func adminUserList(users []db.User) []adminUserItem {
	out := make([]adminUserItem, len(users))
	for i := range users {
		out[i] = adminUserItem{User: users[i], IsPremium: users[i].IsPremium()}
	}
	return out
}

func (h *Handler) adminUserDetail(w http.ResponseWriter, r *http.Request, u *db.User) {
	subs, err := h.store.ListSubscriptions(r.Context(), u.ID)
	if err != nil {
		internalErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"user":          adminUserItem{User: *u, IsPremium: u.IsPremium()},
		"subscriptions": subs,
		"options":       service.GrantOptions,
	})
}

func (h *Handler) adminUser(w http.ResponseWriter, r *http.Request) {
	id, ok := idParam(w, r)
	if !ok {
		return
	}
	u, err := h.store.UserByID(r.Context(), id)
	if err != nil {
		storeErr(w, err)
		return
	}
	h.adminUserDetail(w, r, u)
}

func (h *Handler) adminGrant(w http.ResponseWriter, r *http.Request) {
	admin := userFrom(r)
	id, ok := idParam(w, r)
	var in struct {
		Plan   string `json:"plan"`
		Amount *int64 `json:"amount"` // so'mda; berilmasa reja narxi
	}
	if !ok || !decode(w, r, &in) {
		return
	}
	plan, found := service.GrantByCode(in.Plan)
	if !found {
		badRequest(w, "muddat noto'g'ri")
		return
	}
	amount := plan.Price
	if in.Amount != nil {
		if *in.Amount < 0 || *in.Amount > 1e12 {
			badRequest(w, "summa noto'g'ri")
			return
		}
		amount = *in.Amount * 100
	}
	u, err := h.store.UserByID(r.Context(), id)
	if err != nil {
		storeErr(w, err)
		return
	}
	sub, err := h.store.GrantSubscription(r.Context(), db.GrantInput{
		UserID: u.ID, Plan: plan.Code, Days: plan.Days, Source: "admin",
		Amount: amount, Currency: "UZS", GrantedBy: &admin.TelegramID,
	})
	if err != nil {
		storeErr(w, err)
		return
	}
	go h.bot.NotifyPremium(context.WithoutCancel(r.Context()), u, sub)
	u, err = h.store.UserByID(r.Context(), id)
	if err != nil {
		storeErr(w, err)
		return
	}
	h.adminUserDetail(w, r, u)
}

func (h *Handler) adminRevoke(w http.ResponseWriter, r *http.Request) {
	id, ok := idParam(w, r)
	if !ok {
		return
	}
	if err := h.store.RevokePremium(r.Context(), id); err != nil {
		storeErr(w, err)
		return
	}
	u, err := h.store.UserByID(r.Context(), id)
	if err != nil {
		storeErr(w, err)
		return
	}
	h.adminUserDetail(w, r, u)
}
