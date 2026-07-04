package controllers

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/bicosteve/booking-system/entities"
	"github.com/bicosteve/booking-system/pkg/health"
	"github.com/bicosteve/booking-system/pkg/utils"
)

// waitForDependencies blocks until all enabled dependencies are reachable or
// the startup timeout expires. On timeout it logs the failure and exits.
func (b *Base) waitForDependencies(cfg entities.Config) {
	checkers := buildStartupProbes(cfg, b)
	if len(checkers) == 0 {
		return
	}
	timeout := startupTimeout()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	if err := health.Await(ctx, checkers, 2*time.Second, timeout); err != nil {
		utils.LogError(err.Error(), entities.ErrorLog)
		os.Exit(1)
	}
}

// startupTimeout reads STARTUP_DEPENDENCY_TIMEOUT; defaults to 60s.
func startupTimeout() time.Duration {
	if v := os.Getenv("STARTUP_DEPENDENCY_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			return d
		}
	}
	return 60 * time.Second
}

// buildStartupProbes returns config-only checkers for enabled dependencies.
// It also captures b.rabbitURL so the health endpoint can reuse it.
func buildStartupProbes(cfg entities.Config, b *Base) []health.Checker {
	var cs []health.Checker
	for _, m := range cfg.Mysql {
		dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=latin1&parseTime=True&loc=Local",
			m.Username, m.Password, m.Host, m.Port, m.Schema)
		cs = append(cs, health.MySQLProbe(dsn))
	}
	for _, r := range cfg.Redis {
		cs = append(cs, health.RedisProbe(r.Address+":"+r.Port))
	}
	for _, k := range cfg.Kafka {
		if k.On == 1 {
			cs = append(cs, health.KafkaProbe(utils.KafkaConfigMap(k)))
		}
	}
	for _, rb := range cfg.Rabbit {
		if rb.On == 1 {
			url := rabbitURL(rb)
			b.rabbitURL = url
			cs = append(cs, health.RabbitProbe(url, utils.RabbitTLSConfig(rb)))
		}
	}
	return cs
}

// envBool reads a boolean env var; returns def when unset/unrecognized.
func envBool(name string, def bool) bool {
	switch os.Getenv(name) {
	case "true", "1":
		return true
	case "false", "0":
		return false
	default:
		return def
	}
}

// rabbitURL builds the amqp(s) URL. In prod the vhost is included; elsewhere it is omitted.
func rabbitURL(rb entities.RabbitMQConfig) string {
	scheme := "amqp"
	if rb.TLS {
		scheme = "amqps"
	}
	if os.Getenv("ENV") == "prod" {
		return fmt.Sprintf("%s://%s:%s@%s:%s/%s", scheme, rb.User, rb.Password, rb.Host, rb.Port, rb.Vhost)
	}
	return fmt.Sprintf("%s://%s:%s@%s:%s", scheme, rb.User, rb.Password, rb.Host, rb.Port)
}
