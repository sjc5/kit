package response

import (
	"encoding/json"
	"net/http"
	"strings"
)

type Response struct {
	Writer      http.ResponseWriter
	isCommitted bool
}

func New(w http.ResponseWriter) Response {
	return Response{Writer: w}
}

func (res *Response) IsCommitted() bool {
	return res.isCommitted
}

/////////////////////////////////////////////////////////////////////
// General helpers
/////////////////////////////////////////////////////////////////////

func (res *Response) SetHeader(key, value string) {
	res.Writer.Header().Set(key, value)
	// should not commit here
}

func (res *Response) AddHeader(key, value string) {
	res.Writer.Header().Add(key, value)
	// should not commit here
}

func (res *Response) SetStatus(status int) {
	res.Writer.WriteHeader(status)
	res.flagAsCommitted()
}

func (res *Response) Error(status int, reasons ...string) {
	reason := strings.Join(reasons, " ")
	if reason == "" {
		reason = http.StatusText(status)
	}
	http.Error(res.Writer, reason, status)
	res.flagAsCommitted()
}

/////////////////////////////////////////////////////////////////////
// Contentful responses
/////////////////////////////////////////////////////////////////////

func (res *Response) JSON(obj any) {
	res.SetHeader("Content-Type", "application/json")
	json.NewEncoder(res.Writer).Encode(obj)
	res.flagAsCommitted()
}

func (res *Response) OK() {
	res.SetHeader("Content-Type", "application/json")
	res.Writer.Write([]byte(`{"ok":true}`))
	res.flagAsCommitted()
}

func (res *Response) Text(text string) {
	res.SetHeader("Content-Type", "text/plain")
	res.Writer.Write([]byte(text))
	res.flagAsCommitted()
}

func (res *Response) OKText() {
	res.Text("OK")
}

func (res *Response) HTML(html string) {
	res.SetHeader("Content-Type", "text/html")
	res.Writer.Write([]byte(html))
	res.flagAsCommitted()
}

/////////////////////////////////////////////////////////////////////
// HTTP status responses
/////////////////////////////////////////////////////////////////////

func (res *Response) NotModified() {
	res.SetStatus(http.StatusNotModified)
}

func (res *Response) NotFound() {
	res.SetStatus(http.StatusNotFound)
}

/////////////////////////////////////////////////////////////////////
// Error responses
/////////////////////////////////////////////////////////////////////

func (res *Response) Unauthorized(reasons ...string) {
	res.Error(http.StatusUnauthorized, reasons...)
}

func (res *Response) InternalServerError(reasons ...string) {
	res.Error(http.StatusInternalServerError, reasons...)
}

func (res *Response) BadRequest(reasons ...string) {
	res.Error(http.StatusBadRequest, reasons...)
}

func (res *Response) TooManyRequests(reasons ...string) {
	res.Error(http.StatusTooManyRequests, reasons...)
}

func (res *Response) Forbidden(reasons ...string) {
	res.Error(http.StatusForbidden, reasons...)
}

func (res *Response) MethodNotAllowed(reasons ...string) {
	res.Error(http.StatusMethodNotAllowed, reasons...)
}

/////////////////////////////////////////////////////////////////////
// Internal
/////////////////////////////////////////////////////////////////////

func (res *Response) flagAsCommitted() {
	res.isCommitted = true
}
