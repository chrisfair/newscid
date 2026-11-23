package support

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func newTestSupporterServer(t *testing.T, store map[string]RemoteSupporterRecord) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/supporter" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet:
			handle := strings.TrimSpace(r.URL.Query().Get("handle"))
			if rec, ok := store[strings.ToLower(handle)]; ok {
				_ = json.NewEncoder(w).Encode(rec)
				return
			}
			_ = json.NewEncoder(w).Encode(RemoteSupporterRecord{
				Supported:   false,
				SupporterID: handle,
				PlanID:      "free",
			})
		case http.MethodPost:
			var body struct {
				Handle    string `json:"handle"`
				App       string `json:"app"`
				PlanID    string `json:"planId"`
				TierLabel string `json:"tierLabel"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			rec := RemoteSupporterRecord{
				Supported:     true,
				SupporterID:   body.Handle,
				PlanID:        body.PlanID,
				TierLabel:     body.TierLabel,
				SupportedAtMs: time.Now().UnixMilli(),
				AppSource:     body.App,
			}
			store[strings.ToLower(body.Handle)] = rec
			_ = json.NewEncoder(w).Encode(rec)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))
}
