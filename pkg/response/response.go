package response

import (
	"encoding/json"
	"net/http"
	"strings"
)

type Response struct {
	w http.ResponseWriter
}

func New(w http.ResponseWriter) Response {
	return Response{w}
}

/////////////////////////////////////////////////////////////////////
// General helpers
/////////////////////////////////////////////////////////////////////

func (res Response) SetHeader(key, value string) {
	res.w.Header().Set(key, value)
}

func (res Response) SetStatus(status int) {
	res.w.WriteHeader(status)
}

func (res Response) Error(status int, reasons ...string) {
	reason := strings.Join(reasons, " ")
	if reason == "" {
		reason = http.StatusText(status)
	}
	http.Error(res.w, reason, status)
}

/////////////////////////////////////////////////////////////////////
// Contentful responses
/////////////////////////////////////////////////////////////////////

func (res Response) JSON(obj any) {
	res.SetHeader("Content-Type", "application/json")
	json.NewEncoder(res.w).Encode(obj)
}

func (res Response) OK() {
	res.JSON(map[string]bool{"ok": true})
}

func (res Response) Text(text string) {
	res.SetHeader("Content-Type", "text/plain")
	res.w.Write([]byte(text))
}

func (res Response) OKText() {
	res.Text("OK")
}

func (res Response) HTML(html string) {
	res.SetHeader("Content-Type", "text/html")
	res.w.Write([]byte(html))
}

/////////////////////////////////////////////////////////////////////
// HTTP status responses
/////////////////////////////////////////////////////////////////////

func (res Response) NotModified() {
	res.SetStatus(http.StatusNotModified)
}

func (res Response) NotFound() {
	res.SetStatus(http.StatusNotFound)
}

/////////////////////////////////////////////////////////////////////
// Error responses
/////////////////////////////////////////////////////////////////////

func (res Response) Unauthorized(reasons ...string) {
	res.Error(http.StatusUnauthorized, reasons...)
}

func (res Response) InternalServerError(reasons ...string) {
	res.Error(http.StatusInternalServerError, reasons...)
}

func (res Response) BadRequest(reasons ...string) {
	res.Error(http.StatusBadRequest, reasons...)
}

func (res Response) TooManyRequests(reasons ...string) {
	res.Error(http.StatusTooManyRequests, reasons...)
}

func (res Response) Forbidden(reasons ...string) {
	res.Error(http.StatusForbidden, reasons...)
}

func (res Response) MethodNotAllowed(reasons ...string) {
	res.Error(http.StatusMethodNotAllowed, reasons...)
}
