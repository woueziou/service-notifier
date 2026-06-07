package handler

import (
	"github.com/go-fuego/fuego"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// --- Module: Health ---------------------------------------------------------

type HealthModule struct {
	db  *gorm.DB
	rdb *redis.Client
}

func NewHealthModule(db *gorm.DB, rdb *redis.Client) *HealthModule {
	return &HealthModule{db: db, rdb: rdb}
}

func (m *HealthModule) Register(s *fuego.Server) {
	fuego.Get(s, "/health", m.check)
}

func (m *HealthModule) check(c fuego.ContextNoBody) (any, error) {
	dbOK := true
	if sqlDB, err := m.db.DB(); err != nil {
		dbOK = false
	} else if err := sqlDB.Ping(); err != nil {
		dbOK = false
	}

	redisOK := true
	if err := m.rdb.Ping(c.Context()).Err(); err != nil {
		redisOK = false
	}

	return map[string]interface{}{
		"status": "OK",
		"db":     dbOK,
		"redis":  redisOK,
	}, nil
}
