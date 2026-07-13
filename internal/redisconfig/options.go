package redisconfig

import (
	"crypto/tls"
	"fmt"
	"net/url"
	"strings"

	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"
)

// Options is the shared Redis connection configuration used by go-redis and Asynq.
type Options struct {
	Network   string
	Addr      string
	Username  string
	Password  string
	DB        int
	TLSConfig *tls.Config
}

// Parse converts the canonical REDIS_ADDR URI into the shared client options.
func Parse(raw string) (Options, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return Options{}, nil
	}

	parsedURL, err := url.Parse(raw)
	if err != nil {
		return Options{}, fmt.Errorf("parse Redis URI: %w", err)
	}
	if parsedURL.Scheme != "redis" && parsedURL.Scheme != "rediss" {
		return Options{}, fmt.Errorf("Redis URI scheme must be redis or rediss")
	}
	if parsedURL.Host == "" {
		return Options{}, fmt.Errorf("Redis URI must include a host")
	}
	if parsedURL.RawQuery != "" {
		return Options{}, fmt.Errorf("Redis URI query parameters are not supported")
	}

	parsed, err := redis.ParseURL(raw)
	if err != nil {
		return Options{}, fmt.Errorf("parse Redis URI: %w", err)
	}
	return Options{
		Network:   parsed.Network,
		Addr:      parsed.Addr,
		Username:  parsed.Username,
		Password:  parsed.Password,
		DB:        parsed.DB,
		TLSConfig: parsed.TLSConfig,
	}.Normalized(), nil
}

func MustParse(raw string) Options {
	options, err := Parse(raw)
	if err != nil {
		panic(fmt.Sprintf("invalid REDIS_ADDR: %v", err))
	}
	return options
}

func (o Options) Normalized() Options {
	o.Network = strings.TrimSpace(o.Network)
	o.Addr = strings.TrimSpace(o.Addr)
	o.Username = strings.TrimSpace(o.Username)
	if o.DB < 0 {
		o.DB = 0
	}
	return o
}

func (o Options) GoRedis() *redis.Options {
	o = o.Normalized()
	return &redis.Options{
		Network:   o.Network,
		Addr:      o.Addr,
		Username:  o.Username,
		Password:  o.Password,
		DB:        o.DB,
		TLSConfig: o.TLSConfig,
	}
}

func (o Options) Asynq() asynq.RedisClientOpt {
	o = o.Normalized()
	return asynq.RedisClientOpt{
		Network:   o.Network,
		Addr:      o.Addr,
		Username:  o.Username,
		Password:  o.Password,
		DB:        o.DB,
		TLSConfig: o.TLSConfig,
	}
}
