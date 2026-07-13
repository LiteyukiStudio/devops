package redisconfig

import (
	"testing"
)

func TestParseRedisURI(t *testing.T) {
	options, err := Parse("redis://luna:p%40ss@redis.example.com:6380/4")
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if options.Network != "tcp" || options.Addr != "redis.example.com:6380" || options.Username != "luna" || options.Password != "p@ss" || options.DB != 4 {
		t.Fatalf("Parse() = %#v", options)
	}
	if options.TLSConfig != nil {
		t.Fatalf("TLSConfig = %#v, want nil", options.TLSConfig)
	}
}

func TestParseRedisTLSURI(t *testing.T) {
	options, err := Parse("rediss://default:secret@redis.example.com:6380/0")
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if options.TLSConfig == nil || options.TLSConfig.ServerName != "redis.example.com" {
		t.Fatalf("TLSConfig = %#v", options.TLSConfig)
	}
	if options.GoRedis().TLSConfig == nil || options.Asynq().TLSConfig == nil {
		t.Fatal("TLS config was not propagated to both clients")
	}
}

func TestParsePasswordOnlyRedisURI(t *testing.T) {
	options, err := Parse("redis://:secret@redis.example.com:6379/0")
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if options.Username != "" || options.Password != "secret" {
		t.Fatalf("Parse() = %#v", options)
	}
}

func TestParseRejectsLegacyAndAmbiguousAddresses(t *testing.T) {
	for _, value := range []string{
		"redis.example.com:6379",
		"http://redis.example.com:6379/0",
		"redis:///0",
		"redis://redis.example.com:6379/0?read_timeout=1s",
	} {
		t.Run(value, func(t *testing.T) {
			if _, err := Parse(value); err == nil {
				t.Fatalf("Parse(%q) succeeded", value)
			}
		})
	}
}

func TestOptionsBuildsConsistentClients(t *testing.T) {
	options := Options{
		Addr:     " redis.example.com:6379 ",
		Username: " app ",
		Password: "secret",
		DB:       3,
	}

	goRedis := options.GoRedis()
	asynq := options.Asynq()
	if goRedis.Addr != asynq.Addr || goRedis.Username != asynq.Username || goRedis.Password != asynq.Password || goRedis.DB != asynq.DB {
		t.Fatalf("go-redis and Asynq options differ: %#v %#v", goRedis, asynq)
	}
	if goRedis.Addr != "redis.example.com:6379" || goRedis.Username != "app" {
		t.Fatalf("options were not normalized: %#v", goRedis)
	}
}

func TestOptionsNormalizesNegativeDatabase(t *testing.T) {
	if got := (Options{DB: -1}).Normalized().DB; got != 0 {
		t.Fatalf("DB = %d, want 0", got)
	}
}
