package api

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

func TestBindRestoreParamsForm(t *testing.T) {
	t.Helper()

	e := echo.New()
	form := url.Values{}
	form.Set("environment", "rc")
	req := httptest.NewRequest(http.MethodPost, "/restores/myapp/restore", strings.NewReader(form.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	params, err := bindRestoreParams(c)
	if err != nil {
		t.Fatalf("bindRestoreParams: %v", err)
	}
	if params.Environment != "rc" {
		t.Fatalf("environment: got %q", params.Environment)
	}
}

func TestBindRestoreParamsRequiresFormContentType(t *testing.T) {
	t.Helper()

	e := echo.New()
	req := httptest.NewRequest(
		http.MethodPost,
		"/restores/myapp/restore",
		strings.NewReader(`{"environment":"rc"}`),
	)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	_, err := bindRestoreParams(c)
	if err == nil {
		t.Fatal("expected content-type error")
	}
}

func TestBindRestoreParamsWithExecutionID(t *testing.T) {
	t.Helper()

	e := echo.New()
	executionID := uuid.New()
	form := url.Values{}
	form.Set("environment", "rc")
	form.Set("execution_id", executionID.String())
	req := httptest.NewRequest(http.MethodPost, "/restores/myapp/restore", strings.NewReader(form.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	params, err := bindRestoreParams(c)
	if err != nil {
		t.Fatalf("bindRestoreParams: %v", err)
	}
	if params.ExecutionID == nil || *params.ExecutionID != executionID {
		t.Fatalf("unexpected execution_id: %+v", params.ExecutionID)
	}
}
