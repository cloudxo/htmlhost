package http

import (
	"fmt"
	"log"
	"net/http"
	"runtime/debug"
	"strconv"
	"text/template"
	"time"

	"github.com/bnkamalesh/errors"
	"github.com/bnkamalesh/htmlhost/internal/api"
	"github.com/bnkamalesh/webgo/v4"
	"github.com/tdewolff/minify/v2"
)

var (
	startedAt              = time.Now()
	startedAtHTTPFormatted = startedAt.Format(http.TimeFormat)
)

// Handler has all the dependencies initialized and stored, and made available to all the handler
// methods
type Handler struct {
	templates        map[string]*template.Template
	generatedBaseURL string

	minifier *minify.M
	api      *api.API
}

func (handler *Handler) recoverer(w http.ResponseWriter) {
	rec := recover()
	if rec == nil {
		return
	}

	log.Println(rec, string(debug.Stack()))
	webgo.R500(w, "sorry, unknown error occurred")
}

func expiryHeaders(w http.ResponseWriter, expiry *time.Time) {
	if expiry == nil {
		return
	}

	w.Header().Set(
		"Cache-Control",
		fmt.Sprintf(
			"public,max-age=%d,immutable",
			int(time.Until(*expiry).Seconds()),
		),
	)
	w.Header().Set("Expires", expiry.Format(http.TimeFormat))
}

func cacheHeaders(w http.ResponseWriter, r *http.Request, etag string, modifiedDate string, expiry *time.Time) (continueExecution bool) {
	w.Header().Set("Date", modifiedDate)
	w.Header().Set("Last-Modified", modifiedDate)
	expiryHeaders(w, expiry)

	if etag != "" {
		w.Header().Set("ETag", etag)
		if r.Header.Get("If-None-Match") == etag {
			w.WriteHeader(http.StatusNotModified)
			return false
		}
	}

	if r.Header.Get("If-Modified-Since") == modifiedDate {
		w.WriteHeader(http.StatusNotModified)
		return false
	}

	return true
}

func (h *Handler) minifiedHTML(w http.ResponseWriter, payload []byte) {
	w.Header().Set("Content-Type", "text/html; charset=UTF-8")

	minified, err := h.minifier.Bytes("text/html", payload)
	if err != nil {
		status, msg, _ := errors.HTTPStatusCodeMessage(err)
		w.Header().Set("Content-Length", strconv.Itoa(len(msg)))
		w.WriteHeader(status)
		w.Write([]byte(msg))
		return
	}

	w.Header().Set("Content-Length", strconv.Itoa(len(minified)))
	w.Write(minified)
}

func newHandler(a *api.API, baseURL string) (*Handler, error) {
	tpl, err := template.ParseFiles(
		"./internal/server/http/web/templates/home.html",
	)
	if err != nil {
		return nil, err
	}

	return &Handler{
		api:              a,
		generatedBaseURL: baseURL,
		templates: map[string]*template.Template{
			"home": tpl,
		},
		minifier: newMinifier(),
	}, nil
}
