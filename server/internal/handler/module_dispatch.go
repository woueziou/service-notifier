package handler

import (
	"net/http"

	"woueziou/notifier/internal/model"
	"woueziou/notifier/internal/service"
	"github.com/go-fuego/fuego"
)

// --- Module: Dispatch -------------------------------------------------------

type DispatchModule struct {
	svc *service.DispatchService
}

func NewDispatchModule(svc *service.DispatchService) *DispatchModule {
	return &DispatchModule{svc: svc}
}

func (m *DispatchModule) Register(s *fuego.Server, middlewares ...func(http.Handler) http.Handler) {
	opts := make([]fuego.RouteOption, len(middlewares))
	for i, mw := range middlewares {
		opts[i] = fuego.OptionMiddleware(mw)
	}

	fuego.Post(s, "/v1/send", m.send, opts...)
	fuego.Get(s, "/v1/jobs/{id}", m.getJob, opts...)
}

func (m *DispatchModule) send(c fuego.ContextWithBody[model.SendRequest]) (*model.SendResponse, error) {
	consumer := getConsumer(c.Request())

	req, err := c.Body()
	if err != nil {
		return nil, err
	}

	resp, err := m.svc.Enqueue(c.Context(), consumer, &req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (m *DispatchModule) getJob(c fuego.ContextNoBody) (*model.Job, error) {
	id := c.Request().PathValue("id")
	job, err := m.svc.GetJob(c.Context(), id)
	if err != nil {
		return nil, err
	}
	return job, nil
}
