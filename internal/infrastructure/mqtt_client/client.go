package mqtt_client

import (
	"crypto/tls"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/okieraised/monitoring-agent/internal/config"
	"github.com/okieraised/monitoring-agent/internal/constants"
	"github.com/okieraised/monitoring-agent/internal/utilities"
	"github.com/spf13/viper"
)

func getBool(key string, def bool) bool {
	if !viper.IsSet(key) {
		return def
	}
	return viper.GetBool(key)
}

// readDuration accepts "10s"/"500ms", an int (seconds), or a native duration.
func readDuration(key string, def time.Duration) time.Duration {
	if !viper.IsSet(key) {
		return def
	}
	if s := viper.GetString(key); s != "" {
		if d, err := time.ParseDuration(s); err == nil && d > 0 {
			return d
		}
	}
	if n := viper.GetInt(key); n > 0 {
		return time.Duration(n) * time.Second
	}
	if d := viper.GetDuration(key); d > 0 {
		return d
	}
	return def
}

func isSecureScheme(u string) bool {
	s := strings.ToLower(u)
	return strings.HasPrefix(s, "mqtts://") || strings.HasPrefix(s, "ssl://") ||
		strings.HasPrefix(s, "tls://") || strings.HasPrefix(s, "wss://")
}

var defaultPublishHandler mqtt.MessageHandler = func(_ mqtt.Client, _ mqtt.Message) {}

var defaultConnLostHandler mqtt.ConnectionLostHandler = func(_ mqtt.Client, _ error) {}

var defaultConnAttemptHandler mqtt.ConnectionAttemptHandler = func(_ *url.URL, _ *tls.Config) *tls.Config { return nil }

var defaultReconnectHandler mqtt.ReconnectHandler = func(_ mqtt.Client, _ *mqtt.ClientOptions) {}

type Options struct {
	PublishHandler        mqtt.MessageHandler
	ConnectionLostHandler mqtt.ConnectionLostHandler
	ConnectionAttempt     mqtt.ConnectionAttemptHandler
	ReconnectHandler      mqtt.ReconnectHandler
	CleanSession          *bool
	AutoReconnect         *bool
	ConnectRetry          *bool
	ResumeSubs            *bool
	TLSInsecureSkip       *bool
	WriteTimeout          *time.Duration
	KeepAlive             *time.Duration
	PingTimeout           *time.Duration
	MaxReconnectInterval  *time.Duration
	ConnectTimeout        *time.Duration
	ConnectRetryInterval  *time.Duration

	TLSConfig *tls.Config
}

type Option func(*Options)

func WithPublishHandler(h mqtt.MessageHandler) Option {
	return func(o *Options) { o.PublishHandler = h }
}
func WithConnectionLostHandler(h mqtt.ConnectionLostHandler) Option {
	return func(o *Options) { o.ConnectionLostHandler = h }
}
func WithConnectionAttemptHandler(h mqtt.ConnectionAttemptHandler) Option {
	return func(o *Options) { o.ConnectionAttempt = h }
}
func WithReconnectHandler(h mqtt.ReconnectHandler) Option {
	return func(o *Options) { o.ReconnectHandler = h }
}

func WithCleanSession(v bool) Option {
	return func(o *Options) {
		o.CleanSession = &v
	}
}

func WithAutoReconnect(v bool) Option {
	return func(o *Options) {
		o.AutoReconnect = &v
	}
}

func WithConnectRetry(v bool) Option {
	return func(o *Options) {
		o.ConnectRetry = &v
	}
}

func WithResumeSubs(v bool) Option {
	return func(o *Options) {
		o.ResumeSubs = &v
	}
}

func WithTLSInsecureSkipVerify(v bool) Option {
	return func(o *Options) {
		o.TLSInsecureSkip = &v
	}
}

func WithWriteTimeout(d time.Duration) Option {
	return func(o *Options) {
		o.WriteTimeout = &d
	}
}

func WithKeepAlive(d time.Duration) Option {
	return func(o *Options) {
		o.KeepAlive = &d
	}
}

func WithPingTimeout(d time.Duration) Option {
	return func(o *Options) {
		o.PingTimeout = &d
	}
}

func WithMaxReconnectInterval(d time.Duration) Option {
	return func(o *Options) {
		o.MaxReconnectInterval = &d
	}
}

func WithConnectTimeout(d time.Duration) Option {
	return func(o *Options) {
		o.ConnectTimeout = &d
	}
}

func WithConnectRetryInterval(d time.Duration) Option {
	return func(o *Options) {
		o.ConnectRetryInterval = &d
	}
}

func WithTLSConfig(cfg *tls.Config) Option {
	return func(o *Options) {
		o.TLSConfig = cfg
	}
}

func defaultOptionsFromViper() Options {
	return Options{
		PublishHandler:        defaultPublishHandler,
		ConnectionLostHandler: defaultConnLostHandler,
		ConnectionAttempt:     defaultConnAttemptHandler,
		ReconnectHandler:      defaultReconnectHandler,
		CleanSession:          utilities.Ptr(getBool(config.MqttCleanSession, true)),
		AutoReconnect:         utilities.Ptr(getBool(config.MqttAutoReconnect, true)),
		ConnectRetry:          utilities.Ptr(getBool(config.MqttConnectRetry, true)),
		ResumeSubs:            utilities.Ptr(getBool(config.MqttResumeSubs, true)),
		TLSInsecureSkip:       utilities.Ptr(getBool(config.MqttTLSInsecureSkipVerify, false)),
		WriteTimeout:          utilities.Ptr(readDuration(config.MqttWriteTimeout, constants.MqttDefaultWriteTimeout)),
		KeepAlive:             utilities.Ptr(readDuration(config.MqttKeepAliveDuration, constants.MqttDefaultKeepAlive)),
		PingTimeout:           utilities.Ptr(readDuration(config.MqttPingTimeout, constants.MqttDefaultPingTimeout)),
		MaxReconnectInterval:  utilities.Ptr(readDuration(config.MqttMaxConnectInterval, constants.MqttDefaultMaxReconnectInterval)),
		ConnectTimeout:        utilities.Ptr(readDuration(config.MqttConnectTimeout, constants.MqttDefaultConnectTimeout)),
		ConnectRetryInterval:  utilities.Ptr(readDuration(config.MqttConnectRetryInterval, constants.MqttDefaultConnectRetryInterval)),
	}
}

var (
	once   sync.Once
	client mqtt.Client
)

// NewMQTTClient creates the mqtt client.
func NewMQTTClient(endpoint, clientID string, optFns ...Option) error {
	once.Do(func() {
		conf := defaultOptionsFromViper()
		for _, fn := range optFns {
			if fn != nil {
				fn(&conf)
			}
		}

		opts := mqtt.NewClientOptions().
			AddBroker(endpoint).
			SetClientID(clientID).
			SetDefaultPublishHandler(conf.PublishHandler).
			SetConnectionLostHandler(conf.ConnectionLostHandler).
			SetReconnectingHandler(conf.ReconnectHandler).
			SetConnectionAttemptHandler(conf.ConnectionAttempt).
			SetCleanSession(*conf.CleanSession).
			SetAutoReconnect(*conf.AutoReconnect).
			SetConnectRetry(*conf.ConnectRetry).
			SetConnectRetryInterval(*conf.ConnectRetryInterval).
			SetMaxReconnectInterval(*conf.MaxReconnectInterval).
			SetWriteTimeout(*conf.WriteTimeout).
			SetKeepAlive(*conf.KeepAlive).
			SetPingTimeout(*conf.PingTimeout).
			SetResumeSubs(*conf.ResumeSubs).
			SetConnectTimeout(*conf.ConnectTimeout)
		if conf.TLSConfig != nil {
			opts.SetTLSConfig(conf.TLSConfig)
		} else if isSecureScheme(endpoint) {
			if conf.TLSInsecureSkip != nil && *conf.TLSInsecureSkip {
				opts.SetTLSConfig(&tls.Config{InsecureSkipVerify: true}) // #nosec G402
			} else {
				opts.SetTLSConfig(&tls.Config{})
			}
		}

		c := mqtt.NewClient(opts)
		tok := c.Connect()
		if !tok.WaitTimeout(*conf.ConnectTimeout) {
			panic(fmt.Sprintf("mqtt connect timeout after %s", conf.ConnectTimeout.String()))
			return
		}
		if err := tok.Error(); err != nil {
			panic(fmt.Sprintf("mqtt connect error: %v", err))
			return
		}
		client = c
	})
	return nil
}

func Client() mqtt.Client {
	if client == nil {
		panic("mqtt client not initialized")
	}
	return client
}
