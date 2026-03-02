package handler

import (
	"context"
	"net/http"
	"strconv"
	"subscription-service/internal/logger"
	"time"

	"subscription-service/internal/model"
	"subscription-service/internal/service"

	"github.com/gin-gonic/gin"
)

// --- DTO (Data Transfer Objects) для Swagger ---
// Эти структуры нужно вынести наружу, чтобы swag мог их прочитать

type SubscriptionRequest struct {
	ServiceName string  `json:"service_name" example:"Netflix"`
	Price       int     `json:"price" example:"100"`
	UserID      string  `json:"user_id" example:"user-uuid-123"`
	StartDate   string  `json:"start_date" example:"01-2025"`
	EndDate     *string `json:"end_date" example:"12-2025"`
}

type SumRequest struct {
	UserID      string  `json:"user_id" example:"user-uuid-123"`
	ServiceName string  `json:"service_name" example:"Netflix"`
	From        string  `json:"from" example:"01-2025"`
	To          *string `json:"to" example:"12-2025"`
}

type SumResponse struct {
	Total int `json:"total"`
}

// ------------------------------------------------

type SubscriptionHandler struct {
	service *service.SubscriptionService
}

func NewSubscriptionHandler(s *service.SubscriptionService) *SubscriptionHandler {
	return &SubscriptionHandler{service: s}
}

func (h *SubscriptionHandler) Register(r *gin.Engine) {
	r.POST("/subscriptions", h.Create)
	r.GET("/subscriptions/:id", h.Get)
	r.PUT("/subscriptions/:id", h.Update)
	r.DELETE("/subscriptions/:id", h.Delete)
	r.GET("/subscriptions/sum", h.Sum)
}

func parseMonthYear(s string) (time.Time, error) {
	return time.Parse("01-2006", s)
}

// Create godoc
// @Summary      Создание подписки
// @Description  Создает новую запись о подписке. Даты принимаются в формате "MM-YYYY".
// @Tags         subscriptions
// @Accept       json
// @Produce      json
// @Param        input body SubscriptionRequest true "Данные подписки"
// @Success      201  {object}  model.Subscription
// @Failure      400  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Router       /subscriptions [post]
func (h *SubscriptionHandler) Create(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	var input SubscriptionRequest

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, err)
		return
	}

	start, _ := parseMonthYear(input.StartDate)

	var end *time.Time
	if input.EndDate != nil {
		t, _ := parseMonthYear(*input.EndDate)
		end = &t
	}

	sub := model.Subscription{
		ServiceName: input.ServiceName,
		Price:       input.Price,
		UserID:      input.UserID,
		StartDate:   start,
		EndDate:     end,
	}

	err := h.service.Create(ctx, &sub)
	if err != nil {
		logger.Error("Create error: %v", err)
		c.JSON(http.StatusInternalServerError, err)
		return
	}
	logger.Info("Create success")
	c.JSON(http.StatusCreated, sub)
}

// Get godoc
// @Summary      Получение подписки
// @Description  Возвращает подписку по её ID
// @Tags         subscriptions
// @Produce      json
// @Param        id   path      int  true  "Subscription ID"
// @Success      200  {object}  model.Subscription
// @Failure      400  {object}  map[string]interface{}
// @Router       /subscriptions/{id} [get]
func (h *SubscriptionHandler) Get(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	sub, err := h.service.Get(ctx, id)
	if err != nil {
		logger.Error("Get error: id = %v, err = %v", id, err)
		c.JSON(http.StatusBadRequest, err)
		return
	}
	logger.Info("Get success: id = %v, err = %v", id, err)
	c.JSON(http.StatusOK, sub)
}

// Update godoc
// @Summary      Обновление подписки
// @Description  Обновляет данные существующей подписки
// @Tags         subscriptions
// @Accept       json
// @Produce      json
// @Param        id     path    int                        true  "Subscription ID"
// @Param        input  body    SubscriptionRequest  true  "Новые данные"
// @Success      200    {object}  model.Subscription
// @Failure      400    {object}  map[string]interface{}
// @Router       /subscriptions/{id} [put]
func (h *SubscriptionHandler) Update(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	var input SubscriptionRequest

	err := c.ShouldBindJSON(&input)
	if err != nil {
		logger.Error("Error json: %v", err)
		c.JSON(http.StatusBadRequest, err)
		return
	}

	start, _ := parseMonthYear(input.StartDate)

	var end *time.Time
	if input.EndDate != nil {
		t, _ := parseMonthYear(*input.EndDate)
		end = &t
	}

	sub := model.Subscription{
		ID:          id,
		ServiceName: input.ServiceName,
		Price:       input.Price,
		UserID:      input.UserID,
		StartDate:   start,
		EndDate:     end,
	}

	err = h.service.Update(ctx, &sub)
	if err != nil {
		logger.Error("Update error: id = %v; %v", id, err)
		c.JSON(http.StatusBadRequest, err)
		return
	}
	logger.Info("Update success: %v", sub)
	c.JSON(http.StatusOK, sub)
}

// Delete godoc
// @Summary      Удаление подписки
// @Description  Удаляет подписку по ID
// @Tags         subscriptions
// @Param        id   path      int  true  "Subscription ID"
// @Success      204  {string}  string    "No Content"
// @Failure      400  {object}  map[string]interface{}
// @Router       /subscriptions/{id} [delete]
func (h *SubscriptionHandler) Delete(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	err := h.service.Delete(ctx, id)
	if err != nil {
		logger.Error("Delete error: id = %v; %v", id, err)
		c.JSON(http.StatusBadRequest, err)
		return
	}
	logger.Info("Delete success: %v", id)
	c.Status(http.StatusNoContent)
}

// Sum godoc
// @Summary      Расчет суммы
// @Description  Считает сумму подписок пользователя за период.
// @Description  **Внимание**: Использует JSON body в GET запросе (может не работать в Swagger UI "Try it out", используйте curl).
// @Tags         subscriptions
// @Accept       json
// @Produce      json
// @Param        input body SumRequest true "Фильтры для расчета"
// @Success      200 {object} SumResponse
// @Failure      400 {object} map[string]interface{}
// @Router       /subscriptions/sum [get]
func (h *SubscriptionHandler) Sum(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	var req struct {
		UserID      string  `json:"user_id"`
		ServiceName string  `json:"service_name"`
		From        string  `json:"from"`
		To          *string `json:"to"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, err.Error())
		return
	}

	from, err := parseMonthYear(req.From)
	if err != nil {
		c.JSON(http.StatusBadRequest, "invalid from format")
		return
	}

	var to time.Time
	if req.To != nil {
		to, _ = parseMonthYear(*req.To)
	}

	total, err := h.service.Sum(ctx, req.UserID, req.ServiceName, from, to)
	if err != nil {
		logger.Error("Sum error: user_id=%v; %v", req.UserID, err)
		c.JSON(http.StatusBadRequest, err)
		return
	}
	logger.Info("Sum success: %v", total)
	c.JSON(http.StatusOK, gin.H{"total": total})
}
