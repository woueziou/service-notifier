package handler

import (
	"net/http"

	"woueziou/notifier/internal/repository"
	"woueziou/notifier/internal/service"
	"github.com/go-fuego/fuego"
)

// --- Module: Stats ----------------------------------------------------------

type StatsModule struct {
	consumerRepo *repository.ConsumerRepo
	jobRepo      *repository.JobRepo
	rateLimiter  *service.RateLimiter
}

func NewStatsModule(consumerRepo *repository.ConsumerRepo, jobRepo *repository.JobRepo, rateLimiter *service.RateLimiter) *StatsModule {
	return &StatsModule{
		consumerRepo: consumerRepo,
		jobRepo:      jobRepo,
		rateLimiter:  rateLimiter,
	}
}

func (m *StatsModule) Register(s *fuego.Server, middlewares ...func(http.Handler) http.Handler) {
	opts := make([]fuego.RouteOption, len(middlewares))
	for i, mw := range middlewares {
		opts[i] = fuego.OptionMiddleware(mw)
	}

	fuego.Get(s, "/admin/stats", m.stats, opts...)
}

type consumerStats struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Active      bool    `json:"active"`
	Suspended   bool    `json:"suspended"`
	SenderEmail string  `json:"sender_email"`
	TotalJobs   int64   `json:"total_jobs"`
	BounceRate  float64 `json:"bounce_rate"`
	RLCurrent   int64   `json:"rate_limit_current"`
	RLMax       int     `json:"rate_limit_max"`
}

type statsResponse struct {
	Summary   statsSummary    `json:"summary"`
	Consumers []consumerStats `json:"consumers"`
}

type statsSummary struct {
	TotalConsumers     int   `json:"total_consumers"`
	ActiveConsumers    int   `json:"active_consumers"`
	SuspendedConsumers int   `json:"suspended_consumers"`
	TotalJobs          int64 `json:"total_jobs"`
}

const defaultRLMax = 60

func (m *StatsModule) stats(c fuego.ContextNoBody) (any, error) {
	ctx := c.Context()

	consumers, err := m.consumerRepo.List(ctx)
	if err != nil {
		return nil, err
	}

	items := make([]consumerStats, 0, len(consumers))
	summary := statsSummary{}

	for _, c := range consumers {
		bounceRate, totalJobs, _ := m.jobRepo.GetBounceRate(ctx, c.ID)
		rlCount, _ := m.rateLimiter.GetCurrentCount(ctx, c.ID)

		cs := consumerStats{
			ID:          c.ID,
			Name:        c.Name,
			Active:      c.Active,
			Suspended:   c.Suspended,
			SenderEmail: c.SenderEmail,
			TotalJobs:   totalJobs,
			BounceRate:  bounceRate,
			RLMax:       defaultRLMax,
			RLCurrent:   rlCount,
		}
		items = append(items, cs)

		summary.TotalJobs += totalJobs
		if c.Suspended {
			summary.SuspendedConsumers++
		} else if c.Active {
			summary.ActiveConsumers++
		}
	}

	summary.TotalConsumers = len(consumers)

	return statsResponse{
		Summary:   summary,
		Consumers: items,
	}, nil
}
