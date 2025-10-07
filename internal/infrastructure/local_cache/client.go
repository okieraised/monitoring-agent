package local_cache

import (
	"sync"

	"github.com/dgraph-io/ristretto"
)

type Options struct {
	NumCounters            int64 // number of counters (10x your max items is a good start)
	MaxCost                int64 // total cost capacity (sum of item costs)
	BufferItems            int64 // number of keys per Get buffer
	TtlTickerDurationInSec int64
	IgnoreInternalCost     bool
	Metrics                bool
	OnEvict                func(item *ristretto.Item)
	OnReject               func(item *ristretto.Item)
	OnExit                 func(val interface{})
	KeyToHash              func(key interface{}) (uint64, uint64)
	Cost                   func(value interface{}) int64
}

type Option func(*Options)

func WithNumCounters(n int64) Option {
	return func(o *Options) {
		o.NumCounters = n
	}
}

func WithMaxCost(c int64) Option {
	return func(o *Options) {
		o.MaxCost = c
	}
}

func WithBufferItems(n int64) Option {
	return func(o *Options) {
		o.BufferItems = n
	}
}

func WithMetrics() Option {
	return func(o *Options) {
		o.Metrics = true
	}
}

func WithOnEvict(f func(item *ristretto.Item)) Option {
	return func(o *Options) {
		o.OnEvict = f
	}
}

func WithKeyToHash(f func(key interface{}) (uint64, uint64)) Option {
	return func(o *Options) {
		o.KeyToHash = f
	}
}

func WithCost(f func(value interface{}) int64) Option {
	return func(o *Options) {
		o.Cost = f
	}
}

func WithOnReject(f func(item *ristretto.Item)) Option {
	return func(o *Options) {
		o.OnReject = f
	}
}

func WithOnExit(f func(val interface{})) Option {
	return func(o *Options) {
		o.OnExit = f
	}
}

func WithTtlTickerDurationInSec(d int64) Option {
	return func(o *Options) {
		o.TtlTickerDurationInSec = d
	}
}

func WithIgnoreInternalCost(ignore bool) Option {
	return func(o *Options) {
		o.IgnoreInternalCost = ignore
	}
}

// defaultOptions set default values
func defaultOptions() Options {
	return Options{
		NumCounters: 1_000_000,
		MaxCost:     100_000,
		BufferItems: 64,
		Metrics:     false,
	}
}

var (
	once  sync.Once
	cache *ristretto.Cache
)

// NewLocalCache builds (or returns) the singleton. The first successful call fixes config.
func NewLocalCache(opts ...Option) error {
	once.Do(func() {
		conf := defaultOptions()
		for _, fn := range opts {
			fn(&conf)
		}

		cfg := &ristretto.Config{
			NumCounters:            conf.NumCounters,
			MaxCost:                conf.MaxCost,
			BufferItems:            conf.BufferItems,
			Metrics:                conf.Metrics,
			OnEvict:                conf.OnEvict,
			OnReject:               conf.OnReject,
			OnExit:                 conf.OnExit,
			KeyToHash:              conf.KeyToHash,
			Cost:                   conf.Cost,
			IgnoreInternalCost:     conf.IgnoreInternalCost,
			TtlTickerDurationInSec: conf.TtlTickerDurationInSec,
		}
		c, err := ristretto.NewCache(cfg)
		if err != nil {
			panic(err)
		}
		cache = c
	})
	return nil
}

func Cache() *ristretto.Cache {
	if cache == nil {
		panic("local cache not initialized; call NewLocalCache first")
	}
	return cache
}
