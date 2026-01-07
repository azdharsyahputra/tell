package http

import (
	"net/http"

	"tell/internal/auth"
	"tell/internal/config"
	"tell/internal/http/handler"
	mw "tell/internal/http/middleware"
	"tell/internal/memo"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"gorm.io/gorm"
)

func NewRouter(cfg config.Config, db *gorm.DB, jwtSvc *auth.JWT) http.Handler {
	r := chi.NewRouter()

	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(chimw.Recoverer)

	if len(cfg.CORSAllowedOrigins) > 0 {
		r.Use(mw.CORS(cfg.CORSAllowedOrigins, cfg.CORSAllowCredentials))
	}

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	ah := &handler.AuthHandler{DB: db, JWT: jwtSvc}
	r.Post("/auth/register", ah.Register)
	r.Post("/auth/login", ah.Login)

	me := &handler.MeHandler{}
	r.With(auth.RequireAuth(jwtSvc)).Get("/me", me.Me)

	memoSvc := &memo.Service{DB: db}
	memoH := &handler.MemoHandler{Svc: memoSvc, DB: db}
	memoRead := &handler.MemoReadHandler{DB: db}

	r.Route("/memos", func(r chi.Router) {
		r.Use(auth.RequireAuth(jwtSvc))

		r.Post("/", memoH.Create)
		r.Get("/", memoRead.List)

		r.Get("/tags", memoRead.Tags)

		r.Post("/{id}/events", memoH.AppendEvent)
		r.Get("/{id}/timeline", memoRead.Timeline)
	})

	return r
}
