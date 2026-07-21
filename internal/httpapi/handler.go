package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/CarambaG/subscription-service/internal/domain"
	"github.com/CarambaG/subscription-service/internal/service"
	"github.com/google/uuid"
)

const maxRequestBody = 1 << 20

type SubscriptionService interface {
	Create(context.Context, domain.Subscription) (domain.Subscription, error)
	Get(context.Context, uuid.UUID) (domain.Subscription, error)
	List(context.Context, domain.ListFilter) ([]domain.Subscription, int, error)
	Update(context.Context, uuid.UUID, domain.Subscription) (domain.Subscription, error)
	Delete(context.Context, uuid.UUID) error
	CalculateTotal(context.Context, domain.TotalFilter) (int64, error)
}

type Pinger interface {
	Ping(context.Context) error
}

type Handler struct {
	service SubscriptionService
	pinger  Pinger
}

func NewHandler(service SubscriptionService, pinger Pinger) *Handler {
	return &Handler{service: service, pinger: pinger}
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var request subscriptionRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	subscription, err := request.toDomain()
	if err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}
	created, err := h.service.Create(r.Context(), subscription)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	w.Header().Set("Location", "/api/v1/subscriptions/"+created.ID.String())
	writeJSON(w, http.StatusCreated, toResponse(created))
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePathID(w, r)
	if !ok {
		return
	}
	subscription, err := h.service.Get(r.Context(), id)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toResponse(subscription))
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	filter, err := parseListFilter(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}
	items, total, err := h.service.List(r.Context(), filter)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	responses := make([]subscriptionResponse, 0, len(items))
	for _, item := range items {
		responses = append(responses, toResponse(item))
	}
	writeJSON(w, http.StatusOK, listResponse{
		Items: responses,
		Pagination: paginationResponse{
			Limit:  filter.Limit,
			Offset: filter.Offset,
			Total:  total,
		},
	})
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePathID(w, r)
	if !ok {
		return
	}
	var request subscriptionRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	subscription, err := request.toDomain()
	if err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}
	updated, err := h.service.Update(r.Context(), id, subscription)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toResponse(updated))
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePathID(w, r)
	if !ok {
		return
	}
	if err := h.service.Delete(r.Context(), id); err != nil {
		writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) CalculateTotal(w http.ResponseWriter, r *http.Request) {
	filter, err := parseTotalFilter(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}
	total, err := h.service.CalculateTotal(r.Context(), filter)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, totalResponse{
		TotalCost:  total,
		Currency:   "RUB",
		PeriodStart: domain.FormatMonth(filter.PeriodStart),
		PeriodEnd:   domain.FormatMonth(filter.PeriodEnd),
	})
}

func (h *Handler) Live(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) Ready(w http.ResponseWriter, r *http.Request) {
	if err := h.pinger.Ping(r.Context()); err != nil {
		writeError(w, http.StatusServiceUnavailable, "service_unavailable", "database is unavailable")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

type subscriptionRequest struct {
	ServiceName string  `json:"service_name"`
	Price       int64   `json:"price"`
	UserID      string  `json:"user_id"`
	StartDate   string  `json:"start_date"`
	EndDate     *string `json:"end_date"`
}

func (r subscriptionRequest) toDomain() (domain.Subscription, error) {
	userID, err := uuid.Parse(r.UserID)
	if err != nil {
		return domain.Subscription{}, errors.New("user_id must be a valid UUID")
	}
	startDate, err := domain.ParseMonth(r.StartDate)
	if err != nil {
		return domain.Subscription{}, fmt.Errorf("invalid start_date: %w", err)
	}
	var endDate *time.Time
	if r.EndDate != nil {
		parsed, parseErr := domain.ParseMonth(*r.EndDate)
		if parseErr != nil {
			return domain.Subscription{}, fmt.Errorf("invalid end_date: %w", parseErr)
		}
		endDate = &parsed
	}

	subscription := domain.Subscription{
		ServiceName: r.ServiceName,
		Price:       r.Price,
		UserID:      userID,
		StartDate:   startDate,
	}
	subscription.EndDate = endDate
	return subscription, nil
}

type subscriptionResponse struct {
	ID          string  `json:"id"`
	ServiceName string  `json:"service_name"`
	Price       int64   `json:"price"`
	UserID      string  `json:"user_id"`
	StartDate   string  `json:"start_date"`
	EndDate     *string `json:"end_date,omitempty"`
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   string  `json:"updated_at"`
}

type listResponse struct {
	Items      []subscriptionResponse `json:"items"`
	Pagination paginationResponse     `json:"pagination"`
}

type paginationResponse struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
	Total  int `json:"total"`
}

type totalResponse struct {
	TotalCost   int64  `json:"total_cost"`
	Currency    string `json:"currency"`
	PeriodStart string `json:"period_start"`
	PeriodEnd   string `json:"period_end"`
}

type errorEnvelope struct {
	Error apiError `json:"error"`
}

type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func toResponse(subscription domain.Subscription) subscriptionResponse {
	response := subscriptionResponse{
		ID:          subscription.ID.String(),
		ServiceName: subscription.ServiceName,
		Price:       subscription.Price,
		UserID:      subscription.UserID.String(),
		StartDate:   domain.FormatMonth(subscription.StartDate),
		CreatedAt:   subscription.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:   subscription.UpdatedAt.UTC().Format(time.RFC3339),
	}
	if subscription.EndDate != nil {
		endDate := domain.FormatMonth(*subscription.EndDate)
		response.EndDate = &endDate
	}
	return response
}

func parseListFilter(r *http.Request) (domain.ListFilter, error) {
	query := r.URL.Query()
	limit, err := parseInteger(query.Get("limit"), service.DefaultPageSize)
	if err != nil {
		return domain.ListFilter{}, errors.New("limit must be an integer")
	}
	offset, err := parseInteger(query.Get("offset"), 0)
	if err != nil {
		return domain.ListFilter{}, errors.New("offset must be an integer")
	}
	userID, err := parseOptionalUUID(query.Get("user_id"))
	if err != nil {
		return domain.ListFilter{}, err
	}
	return domain.ListFilter{
		UserID:      userID,
		ServiceName: optionalString(query.Get("service_name")),
		Limit:       limit,
		Offset:      offset,
	}, nil
}

func parseTotalFilter(r *http.Request) (domain.TotalFilter, error) {
	query := r.URL.Query()
	periodStart, err := domain.ParseMonth(query.Get("period_start"))
	if err != nil {
		return domain.TotalFilter{}, fmt.Errorf("invalid period_start: %w", err)
	}
	periodEnd, err := domain.ParseMonth(query.Get("period_end"))
	if err != nil {
		return domain.TotalFilter{}, fmt.Errorf("invalid period_end: %w", err)
	}
	userID, err := parseOptionalUUID(query.Get("user_id"))
	if err != nil {
		return domain.TotalFilter{}, err
	}
	return domain.TotalFilter{
		PeriodStart: periodStart,
		PeriodEnd:   periodEnd,
		UserID:      userID,
		ServiceName: optionalString(query.Get("service_name")),
	}, nil
}

func parsePathID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_id", "id must be a valid UUID")
		return uuid.Nil, false
	}
	return id, true
}

func parseOptionalUUID(value string) (*uuid.UUID, error) {
	if value == "" {
		return nil, nil
	}
	parsed, err := uuid.Parse(value)
	if err != nil {
		return nil, errors.New("user_id must be a valid UUID")
	}
	return &parsed, nil
}

func optionalString(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return &value
}

func parseInteger(value string, fallback int) (int, error) {
	if value == "" {
		return fallback, nil
	}
	return strconv.Atoi(value)
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBody)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("decode JSON: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("request body must contain exactly one JSON object")
	}
	return nil
}

func writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrInvalidInput):
		writeError(w, http.StatusBadRequest, "validation_error", strings.TrimPrefix(err.Error(), domain.ErrInvalidInput.Error()+": "))
	case errors.Is(err, domain.ErrNotFound):
		writeError(w, http.StatusNotFound, "subscription_not_found", "subscription not found")
	default:
		writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
	}
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, errorEnvelope{Error: apiError{Code: code, Message: message}})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
