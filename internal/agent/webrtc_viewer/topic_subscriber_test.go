package webrtc_viewer

import (
	"context"
	"fmt"
	"os/signal"
	"syscall"
	"testing"
	"time"

	"github.com/okieraised/monitoring-agent/internal/constants"
	"github.com/okieraised/monitoring-agent/internal/infrastructure/log"
	"github.com/okieraised/rclgo/humble"
	"github.com/stretchr/testify/assert"
)

func TestNewCameraStreamer(t *testing.T) {
	err := log.InitDefault()
	assert.NoError(t, err)

	err = humble.Init(nil)
	assert.NoError(t, err)
	defer func() {
		cErr := humble.Deinit()
		if cErr != nil && err == nil {
			err = cErr
		}
	}()

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	frameCh := make(chan []byte, 64)
	defer close(frameCh)

	enableCh := make(chan bool)
	defer close(enableCh)

	go func() {
		for c := range frameCh {
			fmt.Println("got", len(c))
		}
	}()

	go func() {
		for {
			time.Sleep(3 * time.Second)
			enableCh <- true
			time.Sleep(4 * time.Second)
			enableCh <- false
		}
	}()

	node, err := humble.NewNode(constants.NodeNameWebRTCStreaming, "")
	assert.NoError(t, err)
	defer node.Close()

	time.Sleep(1 * time.Second)
	err = newCameraSubscriberV2(ctx, node, "/camera/camera/color/image_raw", frameCh, enableCh)
	assert.NoError(t, err)
}
