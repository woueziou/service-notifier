package handler

import (
	"encoding/json"
	"net/http"

	"github.com/flyasky/notifier/internal/model"
	"github.com/flyasky/notifier/internal/service"
	"github.com/go-chi/chi/v5"
)

type ConsumerHandler struct {
	svc    *service.ConsumerService
	domain string
}

func NewConsumerHandler(svc *service.ConsumerService, domain string) *ConsumerHandler {
	return &ConsumerHandler{svc: svc, domain: domain}
}

// @Summary      Create a consumer
// @Description  Register a new consumer and generate an API key
// @Tags         consumers
// @Accept       json
// @Produce      json
// @Param        body  body  model.CreateConsumerRequest  true  "Consumer details"
// @Success      201   {object}  model.CreateConsumerResponse
// @Failure      400   {object}  model.ErrorResponse
// @Failure      500   {object}  model.ErrorResponse
// @Router       /v1/consumers [post]
func (h *ConsumerHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req model.CreateConsumerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate request fields
	if msg := ValidateStruct(&req); msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}

	resp, err := h.svc.Create(r.Context(), &req, h.domain)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, resp)
}

// @Summary      List consumers
// @Description  Get all registered consumers
// @Tags         consumers
// @Produce      json
// @Success      200  {array}  model.Consumer
// @Router       /v1/consumers [get]
func (h *ConsumerHandler) List(w http.ResponseWriter, r *http.Request) {
	consumers, err := h.svc.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, consumers)
}

// @Summary      Get consumer by ID
// @Description  Get a single consumer with details
// @Tags         consumers
// @Produce      json
// @Param        id  path  string  true  "Consumer ID"
// @Success      200  {object}  model.Consumer
// @Failure      404  {object}  model.ErrorResponse
// @Router       /v1/consumers/{id} [get]
func (h *ConsumerHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	consumer, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "consumer not found")
		return
	}
	writeJSON(w, http.StatusOK, consumer)
}
