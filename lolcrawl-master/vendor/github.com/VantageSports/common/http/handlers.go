package http

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/NYTimes/gziphandler"
	"golang.org/x/net/context"
)

// Handler is a context-aware http.Handler that is allowed to spit out errors
// and status codes that should be returned to the client. The only contract to
// abide by is that anything handler an error should NOT have written to the
// ResponseWriter.
type Handler func(ctx context.Context, w http.ResponseWriter, r *http.Request) (interface{}, error)

// HandlerWrapper is a way to nest Handlers inside of one another.
type HandlerWrapper interface {
	Next(Handler) Handler
}

// WrapAll builds a wrapper chain starting with wrappers[0] and ending with
// the specified end handler. It is an error to call Wrap with no wrappers.
// The result of calling WrapAll(h, w1, w2, w3) will be a handler that passes
// through w1 -> w2 -> w3 -> h.
func WrapAll(last Handler, wrappers ...HandlerWrapper) Handler {
	// Start by making the final one wrap the 'end' handler.
	res := wrappers[len(wrappers)-1].Next(last)

	for i := len(wrappers) - 2; i >= 0; i-- {
		cur := wrappers[i]
		res = cur.Next(res)
	}
	return res
}

//
// Logging Wrappers
//

// ResponseLogWrapper is a named http response logger.
type ResponseLogWrapper string

func (rlw ResponseLogWrapper) Next(n Handler) Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) (interface{}, error) {
		start := time.Now()
		res, err := n(ctx, w, r)
		log.Printf("%s %s %s err:%v in: %s", rlw, r.Method, r.RequestURI, err, time.Since(start))
		return res, err
	}
}

//
// Response writers
//

// JsonWriter is a HandlerWrapper that writes the result of the underlying
// handlers to the responsewriter as a json object.
type JSONWriter struct{}

func (jw JSONWriter) Next(next Handler) Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) (interface{}, error) {
		res, err := next(ctx, w, r)
		if err != nil {
			hErr, ok := err.(*Error)
			if !ok {
				hErr = NewError(http.StatusBadRequest, err)
			}
			writeJSON(w, hErr.Code, true, hErr)
			return nil, hErr
		}
		return nil, writeJSON(w, http.StatusOK, true, res)
	}
}

//
// Header manipulation
//

// SetAllowedOrigin is a string representing the value to set the allow-origin
// header to on http responses. If there is an origin specified in the request,
// that origin will be used instead.
type SetAllowedOrigin string

func (so SetAllowedOrigin) Next(n Handler) Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) (interface{}, error) {
		origin := r.Header.Get("Origin")
		if origin == "" {
			origin = string(so)
		}
		w.Header().Add("Access-Control-Allow-Origin", origin)
		return n(ctx, w, r)
	}
}

//
// Authentication
//

const ContextKeyToken = "auth.token"

// CredentialParser is a context cache key that the bearer token in the
// Authorization header will be stored as. NOTE: does not verify/decode the
// token, nor is it a problem if the token doesn't exist (it just won't be
// present in the context object)
type CredentialsParser string

func (cp CredentialsParser) Next(next Handler) Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) (interface{}, error) {
		header := r.Header.Get("Authorization")
		if len(header) > 7 { // remove the 'bearer ' prefix
			ctx = context.WithValue(ctx, string(cp), header[7:])
		}
		return next(ctx, w, r)
	}
}

//
// Base handler wrapper
//

// BaseHandler is a plain-old http handler that will call the provided handler
// and optionally gzip responses.
func BaseHandler(ctx context.Context, gzip bool, h Handler) http.Handler {
	base := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = h(ctx, w, r)
	})

	if gzip {
		return gziphandler.GzipHandler(base)
	}
	return base
}

//
// Generic handlers
//

func ServeObj(v interface{}) Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) (interface{}, error) {
		return v, nil
	}
}

func ServeOK() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"success": true}`))
	})
}

func ServeFile(path string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, path)
	})
}

func ServeOptions() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		w.Header().Add("Access-Control-Allow-Origin", "*")
		w.Header().Add("Access-Control-Allow-Headers", "Authorization,Content-Type,Origin")
		w.Header().Add("Access-Control-Allow-Methods", "GET,DELETE,OPTIONS,PATH,POST,PUT")
		w.Write([]byte(`{"success": true}`))
	})
}

//
// Utility
//

// writeJSON attempts to write the value to the ResponseWriter. Returns any
// errors encountered, but clients should be aware that there may not be any
// 'recovery' option from write errors, since the header (and some data) was
// likely already written.
func writeJSON(w http.ResponseWriter, code int, indent bool, v interface{}) error {
	data, err := marshal(indent, v)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		// This is almost certainly a developer error.
		log.Println("json marshal error: ", err)
		return err
	}
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(code)
	if _, err = w.Write(data); err != nil {
		log.Printf("error writing response (maybe not returned to user): %v", err)
	}
	return err
}

func marshal(indent bool, v interface{}) ([]byte, error) {
	if indent {
		return json.MarshalIndent(v, "", "  ")
	}
	return json.Marshal(v)
}
