package handler

import (
	"net/http"

	"woueziou/notifier/internal/model"
	"woueziou/notifier/internal/service"
	"github.com/go-fuego/fuego"
)

// --- Module: Consumer CRUD --------------------------------------------------

type ConsumerModule struct {
	svc    *service.ConsumerService
	domain string
}

func NewConsumerModule(svc *service.ConsumerService, domain string) *ConsumerModule {
	return &ConsumerModule{svc: svc, domain: domain}
}

func (m *ConsumerModule) Register(s *fuego.Server, middlewares ...func(http.Handler) http.Handler) {
	opts := make([]fuego.RouteOption, len(middlewares))
	for i, mw := range middlewares {
		opts[i] = fuego.OptionMiddleware(mw)
	}

	fuego.Get(s, "/admin/consumers", m.list, opts...)
	fuego.Get(s, "/admin/consumers/{id}", m.getByID, opts...)
	fuego.Post(s, "/admin/consumers", m.create, opts...)
}

func (m *ConsumerModule) list(c fuego.ContextNoBody) (any, error) {
	consumers, err := m.svc.List(c.Context())
	if err != nil {
		return nil, err
	}
	return consumers, nil
}

func (m *ConsumerModule) getByID(c fuego.ContextNoBody) (any, error) {
	id := c.Request().PathValue("id")
	consumer, err := m.svc.GetByID(c.Context(), id)
	if err != nil {
		return nil, fuego.NotFoundError{Title: "Consumer not found", Detail: err.Error()}
	}
	return consumer, nil
}

func (m *ConsumerModule) create(c fuego.ContextWithBody[model.CreateConsumerRequest]) (any, error) {
	req, err := c.Body()
	if err != nil {
		return nil, err
	}

	resp, err := m.svc.Create(c.Context(), &req, m.domain)
	if err != nil {
		return nil, err
	}
	return resp, nil
}
