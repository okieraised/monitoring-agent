package s3_client

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/retry"
	awscfg "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

var (
	client *s3.Client
	once   sync.Once
)

type Options struct {
	Region           string
	AccessKeyID      string
	SecretAccessKey  string
	SessionToken     string
	Endpoint         string // e.g., "https://s3.amazonaws.com" or "https://s3.minio.local:9000"
	UsePathStyle     bool   // true for MinIO/on-prem
	HTTPClient       *http.Client
	RetryMaxAttempts int           // default AWS SDK policy if 0
	RetryMaxBackoff  time.Duration // cap; 0 = default
}

// Client returns the singleton S3 client (after NewS3Client).
func Client() *s3.Client {
	if client == nil {
		panic("s3 client not initialized; call NewS3Client first")
	}
	return client
}

// Option mutates Options.
type Option func(*Options)

func WithRegion(r string) Option { return func(o *Options) { o.Region = r } }

func WithStaticCredentials(id, secret, token string) Option {
	return func(o *Options) { o.AccessKeyID, o.SecretAccessKey, o.SessionToken = id, secret, token }
}

func WithEndpoint(endpoint string, pathStyle bool) Option {
	return func(o *Options) { o.Endpoint, o.UsePathStyle = endpoint, pathStyle }
}
func WithHTTPClient(h *http.Client) Option { return func(o *Options) { o.HTTPClient = h } }

func WithRetry(maxAttempts int, maxBackoff time.Duration) Option {
	return func(o *Options) { o.RetryMaxAttempts, o.RetryMaxBackoff = maxAttempts, maxBackoff }
}

func NewS3Client(ctx context.Context, opts ...Option) error {
	conf := Options{}
	for _, fn := range opts {
		fn(&conf)
	}

	awsCfg, err := loadAWSConfig(ctx, conf)
	if err != nil {
		return err
	}

	once.Do(func() {
		if conf.RetryMaxAttempts > 0 || conf.RetryMaxBackoff > 0 {
			awsCfg.RetryMaxAttempts = conf.RetryMaxAttempts
			if conf.RetryMaxBackoff > 0 {
				awsCfg.Retryer = func() aws.Retryer {
					return retry.AddWithMaxAttempts(retry.NewStandard(), awsCfg.RetryMaxAttempts)
				}
			}
		}

		var s3Opts []func(*s3.Options)
		if conf.UsePathStyle {
			s3Opts = append(s3Opts, func(o *s3.Options) { o.UsePathStyle = true })
		}

		if conf.Endpoint != "" {
			s3Opts = append(s3Opts, func(o *s3.Options) {
				o.BaseEndpoint = aws.String(conf.Endpoint)
			})
		}

		client = s3.NewFromConfig(awsCfg, s3Opts...)
	})

	return nil
}

func loadAWSConfig(ctx context.Context, o Options) (aws.Config, error) {
	var lo []func(*awscfg.LoadOptions) error

	if o.Region != "" {
		lo = append(lo, awscfg.WithRegion(o.Region))
	}

	if o.HTTPClient != nil {
		lo = append(lo, awscfg.WithHTTPClient(o.HTTPClient))
	}

	creds := aws.NewCredentialsCache(credentials.NewStaticCredentialsProvider(o.AccessKeyID, o.SecretAccessKey, o.SessionToken))
	lo = append(lo, awscfg.WithCredentialsProvider(creds))

	cfg, err := awscfg.LoadDefaultConfig(ctx, lo...)
	if err != nil {
		return aws.Config{}, err
	}

	return cfg, nil
}
