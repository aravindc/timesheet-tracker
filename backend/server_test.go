package main

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/lib/pq"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func newTestServer() (*Server, *observer.ObservedLogs) {
	gin.SetMode(gin.TestMode)
	core, logs := observer.New(zap.InfoLevel)
	return &Server{logger: zap.New(core)}, logs
}

func TestRespondDBError(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantMsg    string
	}{
		{"unique violation", &pq.Error{Code: "23505", Constraint: "projects_name_key"}, http.StatusConflict, "A record with these values already exists"},
		{"foreign key violation", &pq.Error{Code: "23503"}, http.StatusBadRequest, "Referenced record does not exist"},
		{"invalid text representation", &pq.Error{Code: "22P02"}, http.StatusBadRequest, "Invalid input"},
		{"other error", errors.New(`relation "projects" does not exist`), http.StatusInternalServerError, "Internal server error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, logs := newTestServer()
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodGet, "/api/projects", nil)

			s.respondDBError(c, tt.err)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}
			body := w.Body.String()
			if !strings.Contains(body, tt.wantMsg) {
				t.Errorf("body = %s, want message %q", body, tt.wantMsg)
			}
			if strings.Contains(body, "projects") {
				t.Errorf("body leaks database details: %s", body)
			}
			if logs.Len() != 1 {
				t.Errorf("expected the error to be logged once, got %d entries", logs.Len())
			}
		})
	}
}

func TestOAuthCallbackLogOmitsQuery(t *testing.T) {
	s, logs := newTestServer()
	r := gin.New()
	r.Use(zapMiddlewares(s)...)
	r.GET("/api/auth/google/callback", s.requestLoggerWithoutQuery(), func(c *gin.Context) { c.Status(http.StatusTemporaryRedirect) })
	r.GET("/api/projects", func(c *gin.Context) { c.Status(http.StatusOK) })

	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/auth/google/callback?code=secretcode&state=secretstate", nil))

	if logs.Len() != 1 {
		t.Fatalf("expected exactly one log entry for the callback, got %d", logs.Len())
	}
	for _, entry := range logs.All() {
		for k, v := range entry.ContextMap() {
			if s, ok := v.(string); ok && strings.Contains(s, "secret") {
				t.Errorf("log field %q contains query data: %q", k, s)
			}
		}
	}

	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/projects?x=1", nil))
	if logs.Len() != 2 {
		t.Errorf("expected other routes to still be logged by ginzap, got %d entries", logs.Len())
	}
}
