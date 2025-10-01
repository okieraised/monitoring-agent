package local_cache

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewLocalCache(t *testing.T) {
	err := NewLocalCache()
	assert.NoError(t, err)

	success := Cache().Set("test-key", "test", 10)
	assert.Equal(t, true, success)

	time.Sleep(50 * time.Millisecond)

	val, success := Cache().Get("test-key")
	assert.Equal(t, "test", val)
	assert.Equal(t, true, success)
}
