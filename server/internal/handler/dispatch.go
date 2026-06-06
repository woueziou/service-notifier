package handler

import (
	"encoding/json"
	"net/http"

	"github.com/flyasky/notifier/internal/model"
	"github.com/flyasky/notifier/internal/service"
	"github.com/go-chi/chi/v5"
)

type DispatchHandler struct {
	svc *service.DispatchService
}

func NewDispatchHandler(svc *service.DispatchService) *DispatchHandler {
	return &DispatchHandler{svc: svc}
}

// @Summary      Send an email
// @Description  Dispatch an email on behalf of a consumer
// @Tags         dispatch
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body  model.SendRequest  true  "Email payload"
// @Success      202   {object}  model.SendResponse
// @Failure      400   {object}  model.ErrorResponse
// @Failure      429   {object}  model.ErrorResponse
// @Router       /v1/send [post]
func (h *DispatchHandler) Send(w http.ResponseWriter, r *http.Request) {
	consumer := getConsumer(r)

	var req model.SendRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := h.svc.Enqueue(r.Context(), consumer, &req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusAccepted, resp)
}

// @Summary      Get job status
// @Description  Get the status of an email job
// @Tags         dispatch
// @Produce      json
// @Security     BearerAuth
// @Param        id  path  string  true  "Job ID"
// @Success      200  {object}  model.Job
// @Failure      404  {object}  model.ErrorResponse
// @Router       /v1/jobs/{id} [get]
func (h *DispatchHandler) GetJob(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	job, err := h.svc.GetJob(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "job not found")
		return
	}
	writeJSON(w, http.StatusOK, job)
}
