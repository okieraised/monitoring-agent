package s3_client

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/okieraised/monitoring-agent/internal/utilities"
	"github.com/stretchr/testify/assert"
)

func TestInit(t *testing.T) {
	err := NewS3Client(
		context.Background(),
		WithRegion("us-east-1"),
		WithEndpoint("https://s3.dataloop.io", true),
		WithStaticCredentials("admin", "123456aA@", ""),
		WithRetry(5, 30*time.Second),
		WithHTTPClient(&http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			},
		},
		),
	)
	assert.NoError(t, err)

	buckets, err := Client().ListBuckets(context.Background(), nil)
	assert.NoError(t, err)
	for _, bucket := range buckets.Buckets {
		fmt.Println(utilities.Deref(bucket.Name))
	}
}
