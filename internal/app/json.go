package app

import (
	"encoding/json"
	"io"
)

type ErrorBody struct {
	Kind    string `json:"kind"`
	Message string `json:"message"`
	Path    string `json:"path,omitempty"`
	Hint    string `json:"hint,omitempty"`
}

type Envelope map[string]any

func Success(extra map[string]any) Envelope {
	out := Envelope{"ok": true}
	for k, v := range extra {
		out[k] = v
	}
	return out
}

func Failure(err *Error) Envelope {
	body := ErrorBody{
		Kind:    err.Kind,
		Message: err.Message,
		Path:    err.Path,
		Hint:    err.Hint,
	}
	return Envelope{"ok": false, "error": body}
}

func WriteJSON(w io.Writer, env Envelope) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(env)
}
