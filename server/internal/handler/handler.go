package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"monefy/internal/auth"
	"monefy/internal/bot"
	"monefy/internal/config"
	"monefy/internal/db"
)

type Handler struct {
	cfg   *config.Config
	store *db.Store
	bot   *bot.Bot
}

func New(cfg *config.Config, store *db.Store, b *bot.Bot) *Handler {
	return &Handler{cfg: cfg, store: store, bot: b}
}

// Routes API marshrutlari. staticDir bo'sh bo'lmasa, React build ham shu serverdan beriladi.
func (h *Handler) Routes(staticDir string) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RealIP, middleware.Recoverer, h.cors)

	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) { w.Write([]byte("ok")) })

	r.Route("/api", func(r chi.Router) {
		r.Use(h.authenticate)

		// Telefon raqamisiz ham ochiq: ilova shu orqali raqam so'raydi
		r.Get("/me", h.me)
		r.Post("/me/phone", h.savePhone)

		r.Group(func(r chi.Router) {
			r.Use(h.requirePhone)

			r.Patch("/me/settings", h.updateSettings)

			r.Get("/accounts", h.listAccounts)
			r.Post("/accounts", h.createAccount)
			r.Patch("/accounts/{id}", h.updateAccount)
			r.Delete("/accounts/{id}", h.deleteAccount)

			r.Get("/categories", h.listCategories)
			r.With(h.requirePremium).Post("/categories", h.createCategory)
			r.With(h.requirePremium).Patch("/categories/{id}", h.updateCategory)
			r.With(h.requirePremium).Delete("/categories/{id}", h.deleteCategory)

			r.Get("/transactions", h.listTransactions)
			r.Post("/transactions", h.createTransaction)
			r.Get("/transactions/{id}", h.getTransaction)
			r.Patch("/transactions/{id}", h.updateTransaction)
			r.Delete("/transactions/{id}", h.deleteTransaction)

			r.Get("/transfers", h.listTransfers)
			r.Post("/transfers", h.createTransfer)
			r.Delete("/transfers/{id}", h.deleteTransfer)

			r.Get("/summary", h.summary)

			r.Get("/rates", h.listRates)
			r.With(h.requirePremium).Put("/rates/{currency}", h.upsertRate)
			r.With(h.requirePremium).Delete("/rates/{currency}", h.deleteRate)

			r.With(h.requirePremium).Post("/export", h.export)

			r.Get("/premium", h.premiumInfo)
			r.Post("/premium/request", h.premiumRequest)

			r.Route("/admin", func(r chi.Router) {
				r.Use(h.requireAdmin)
				r.Get("/stats", h.adminStats)
				r.Get("/users", h.adminUsers)
				r.Get("/users/{id}", h.adminUser)
				r.Post("/users/{id}/premium", h.adminGrant)
				r.Post("/users/{id}/revoke", h.adminRevoke)
			})
		})
	})

	if staticDir != "" {
		files := http.FileServer(http.Dir(staticDir))
		r.Get("/*", func(w http.ResponseWriter, r *http.Request) {
			// index.html keshlanmasin, aks holda yangi deploy Telegram'da ko'rinmaydi
			if r.URL.Path == "/" || r.URL.Path == "/index.html" {
				w.Header().Set("Cache-Control", "no-cache")
			}
			files.ServeHTTP(w, r)
		})
	}
	return r
}

// ---- middleware ----

type ctxKey struct{}
type devKey struct{}

func userFrom(r *http.Request) *db.User { return r.Context().Value(ctxKey{}).(*db.User) }

// isDev so'rov DEV_TELEGRAM_ID orqali (Telegram'siz, lokal brauzerda) kirganmi.
func isDev(r *http.Request) bool { return r.Context().Value(devKey{}) != nil }

func (h *Handler) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var tg *auth.TelegramUser
		dev := false
		initData, ok := strings.CutPrefix(r.Header.Get("Authorization"), "tma ")
		switch {
		case ok && initData != "":
			u, err := auth.Validate(initData, h.cfg.BotToken, 24*time.Hour)
			if err != nil {
				writeErr(w, http.StatusUnauthorized, "unauthorized", err.Error())
				return
			}
			tg = u
		case h.cfg.DevTelegramID != 0:
			tg = &auth.TelegramUser{ID: h.cfg.DevTelegramID, FirstName: "Dev"}
			dev = true
		default:
			writeErr(w, http.StatusUnauthorized, "unauthorized", "missing Telegram initData")
			return
		}

		u, created, err := h.store.AuthUser(r.Context(), db.UpsertUser{
			TelegramID: tg.ID, FirstName: tg.FirstName, LastName: tg.LastName,
			Username: tg.Username, LanguageCode: tg.LanguageCode,
		})
		if err != nil {
			internalErr(w, err)
			return
		}
		if created {
			go h.bot.NotifyAdmin(context.Background(), "🆕 Yangi foydalanuvchi: "+bot.UserLabel(u))
		}
		ctx := context.WithValue(r.Context(), ctxKey{}, u)
		if dev {
			ctx = context.WithValue(ctx, devKey{}, true)
		}
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// requirePhone telefon raqamini ulashmagan foydalanuvchiga ilovadan foydalanishga ruxsat bermaydi.
// Lokal dev rejimda (Telegram yo'q, raqam so'rab bo'lmaydi) tekshirilmaydi.
func (h *Handler) requirePhone(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if userFrom(r).Phone == "" && !isDev(r) {
			writeErr(w, http.StatusForbidden, "phone_required", "Telefon raqamingizni ulashing")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (h *Handler) isAdmin(u *db.User) bool {
	return h.cfg.AdminChatID != 0 && u.TelegramID == h.cfg.AdminChatID
}

// requireAdmin admin = telegram_id ADMIN_CHAT_ID ga teng foydalanuvchi (shaxsiy chat ID = user ID).
func (h *Handler) requireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !h.isAdmin(userFrom(r)) {
			writeErr(w, http.StatusForbidden, "forbidden", "Ruxsat yo'q")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (h *Handler) requirePremium(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !userFrom(r).IsPremium() {
			writeErr(w, http.StatusPaymentRequired, "premium_required", "Bu imkoniyat faqat Premium uchun")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (h *Handler) cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		for _, o := range h.cfg.CORSOrigins {
			if o == "*" || o == origin {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Vary", "Origin")
				w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
				break
			}
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// ---- helpers ----

type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, code, msg string) {
	writeJSON(w, status, map[string]apiError{"error": {Code: code, Message: msg}})
}

func internalErr(w http.ResponseWriter, err error) {
	slog.Error("internal error", "err", err)
	writeErr(w, http.StatusInternalServerError, "internal", "Server xatosi")
}

// storeErr db xatolarini HTTP javobiga aylantiradi.
func storeErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, db.ErrNotFound):
		writeErr(w, http.StatusNotFound, "not_found", "Topilmadi")
	case errors.Is(err, db.ErrCategoryInUse):
		writeErr(w, http.StatusConflict, "category_in_use", "Kategoriyada yozuvlar bor, uni arxivlang")
	default:
		internalErr(w, err)
	}
}

func badRequest(w http.ResponseWriter, msg string) {
	writeErr(w, http.StatusBadRequest, "bad_request", msg)
}

func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		badRequest(w, "JSON noto'g'ri: "+err.Error())
		return false
	}
	return true
}

func idParam(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		badRequest(w, "id noto'g'ri")
		return 0, false
	}
	return id, true
}

var (
	currencyRe = regexp.MustCompile(`^[A-Z]{3}$`)
	dateRe     = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
)

func validDate(s string) bool {
	if !dateRe.MatchString(s) {
		return false
	}
	_, err := time.Parse("2006-01-02", s)
	return err == nil
}

// period ?from=&to= parametrlarini o'qiydi.
func period(w http.ResponseWriter, r *http.Request) (db.Period, bool) {
	p := db.Period{From: r.URL.Query().Get("from"), To: r.URL.Query().Get("to")}
	for _, d := range []string{p.From, p.To} {
		if d != "" && !validDate(d) {
			badRequest(w, fmt.Sprintf("sana noto'g'ri: %q (YYYY-MM-DD)", d))
			return p, false
		}
	}
	return p, true
}

func queryInt(r *http.Request, key string) int64 {
	n, _ := strconv.ParseInt(r.URL.Query().Get(key), 10, 64)
	return n
}
