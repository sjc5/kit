package response

import (
	"encoding/json"
	"net/http"
)

type Response struct {
	w http.ResponseWriter
}

func New(w http.ResponseWriter) Response {
	return Response{w}
}

func (r Response) JSON(obj any) {
	w := r.w
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(obj)
}

func (r Response) Redirect(req *http.Request, url string) {
	http.Redirect(r.w, req, url, http.StatusFound)
}

func (r Response) Unauthorized(reason string) {
	if reason == "" {
		reason = "Unauthorized"
	}
	http.Error(r.w, reason, http.StatusUnauthorized)
}

func (r Response) InternalServerError(reason string) {
	if reason == "" {
		reason = "Internal server error"
	}
	http.Error(r.w, reason, http.StatusInternalServerError)
}

func (r Response) BadRequest(reason string) {
	if reason == "" {
		reason = "Bad request"
	}
	http.Error(r.w, reason, http.StatusBadRequest)
}

func (r Response) TooManyRequests(reason string) {
	if reason == "" {
		reason = "Too many requests"
	}
	http.Error(r.w, reason, http.StatusTooManyRequests)
}

func (r Response) Forbidden(reason string) {
	if reason == "" {
		reason = "Forbidden"
	}
	http.Error(r.w, reason, http.StatusForbidden)
}
