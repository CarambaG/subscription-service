package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/CarambaG/subscription-service/internal/domain"
	"github.com/google/uuid"
)

type serviceStub struct {
	created   domain.Subscription
	updateErr error
}

func (s *serviceStub) Create(_ context.Context, subscription domain.Subscription) (domain.Subscription, error) {
	subscription.ID = uuid.MustParse("13ed5902-ae6f-4663-8d2e-54c63d591456")
	subscription.CreatedAt = time.Date(2025, time.July, 1, 0, 0, 0, 0, time.UTC)
	subscription.UpdatedAt = subscription.CreatedAt
	s.created = subscription
	return subscription, nil
}

func (s *serviceStub) Get(context.Context, uuid.UUID) (domain.Subscription, error) {
	return domain.Subscription{}, domain.ErrNotFound
}

func (s *serviceStub) List(context.Context, domain.ListFilter) ([]domain.Subscription, int, error) {
	return []domain.Subscription{}, 0, nil
}

func (s *serviceStub) Update(context.Context, uuid.UUID, domain.Subscription) (domain.Subscription, error) {
	return domain.Subscription{}, s.updateErr
}

func (s *serviceStub) Delete(context.Context, uuid.UUID) error {
	return nil
}

func (s *serviceStub) CalculateTotal(context.Context, domain.TotalFilter) (int64, error) {
	return 0, nil
}

type pingerStub struct{}

func (pingerStub) Ping(context.Context) error { return nil }

func TestCreateReturns201AndLocation(t *testing.T) {
	service := &serviceStub{}
	router := NewRouter(NewHandler(service, pingerStub{}))
	body := `{
		"service_name":"Yandex Plus",
		"price":400,
		"user_id":"60601fee-2bf1-4721-ae6f-7636e79a0cba",
		"start_date":"07-2025"
	}`
	request := httptest.NewRequest(http.MethodPost, "/api/v1/subscriptions", strings.NewReader(body))
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d: %s", recorder.Code, http.StatusCreated, recorder.Body.String())
	}
	if got := recorder.Header().Get("Location"); got != "/api/v1/subscriptions/13ed5902-ae6f-4663-8d2e-54c63d591456" {
		t.Fatalf("Location = %q", got)
	}
	if service.created.Price != 400 {
		t.Fatalf("created price = %d", service.created.Price)
	}
}

func TestUpdateMissingSubscriptionReturns404(t *testing.T) {
	service := &serviceStub{updateErr: domain.ErrNotFound}
	router := NewRouter(NewHandler(service, pingerStub{}))
	body := `{
		"service_name":"Yandex Plus",
		"price":400,
		"user_id":"60601fee-2bf1-4721-ae6f-7636e79a0cba",
		"start_date":"07-2025"
	}`
	request := httptest.NewRequest(http.MethodPut,
		"/api/v1/subscriptions/13ed5902-ae6f-4663-8d2e-54c63d591456",
		strings.NewReader(body),
	)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d: %s", recorder.Code, http.StatusNotFound, recorder.Body.String())
	}
	var response errorEnvelope
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Error.Code != "subscription_not_found" {
		t.Fatalf("error code = %q", response.Error.Code)
	}
}

func TestInvalidPathIDReturns400(t *testing.T) {
	router := NewRouter(NewHandler(&serviceStub{}, pingerStub{}))
	request := httptest.NewRequest(http.MethodGet, "/api/v1/subscriptions/not-a-uuid", nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
	var response errorEnvelope
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Error.Code != "invalid_id" {
		t.Fatalf("error code = %q", response.Error.Code)
	}
}

func TestServiceErrorDoesNotLeakDetails(t *testing.T) {
	service := &serviceStub{updateErr: errors.New("password=secret")}
	router := NewRouter(NewHandler(service, pingerStub{}))
	body := `{
		"service_name":"Yandex Plus",
		"price":400,
		"user_id":"60601fee-2bf1-4721-ae6f-7636e79a0cba",
		"start_date":"07-2025"
	}`
	request := httptest.NewRequest(http.MethodPut,
		"/api/v1/subscriptions/13ed5902-ae6f-4663-8d2e-54c63d591456",
		strings.NewReader(body),
	)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusInternalServerError)
	}
	if strings.Contains(recorder.Body.String(), "secret") {
		t.Fatal("internal error details leaked to response")
	}
}
