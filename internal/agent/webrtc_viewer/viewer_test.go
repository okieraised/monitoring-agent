package webrtc_viewer

import (
	"context"
	"fmt"
	"os/signal"
	"syscall"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/okieraised/monitoring-agent/internal/config"
	"github.com/okieraised/monitoring-agent/internal/infrastructure/mqtt_client"
	"github.com/okieraised/rclgo/humble"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestNewWebRTCViewer(t *testing.T) {
	viper.Set(config.MqttWebRTCOfferTopic, "tripg")
	viper.Set(config.MqttWebRTCAnswerTopic, "tripg2")
	//rosTopic := "/camera/front_robot/color/image_raw/compressed"
	err := mqtt_client.NewMQTTClient(
		"mqtt://127.0.0.1:1883",
		uuid.New().String(),
		mqtt_client.WithAutoReconnect(true),
		mqtt_client.WithConnectTimeout(5*time.Second),
		mqtt_client.WithTLSInsecureSkipVerify(true),
	)
	assert.NoError(t, err)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	err = humble.Init(nil)
	assert.NoError(t, err)

	defer humble.Deinit()

	c, err := NewWebRTCViewer(ctx)
	if err != nil {
		fmt.Println(err)
	}
	defer c.Close()

	err = c.Start()
	assert.NoError(t, err)

	//go func() {
	//	time.Sleep(2 * time.Second)
	//
	//	msg := WebrtcSDPOfferMsg{
	//		HeaderID: 1,
	//		Command: struct {
	//			Type     string    `json:"type"`
	//			ID       uuid.UUID `json:"id"`
	//			Argument struct {
	//				RosTopic string `json:"rosTopic"`
	//				SDP      string `json:"sdp"`
	//			} `json:"argument"`
	//		}{
	//			Type: "Signaling",
	//			ID:   uuid.New(),
	//			Argument: struct {
	//				RosTopic string `json:"rosTopic"`
	//				SDP      string `json:"sdp"`
	//			}{
	//				RosTopic: rosTopic,
	//				SDP:      "",
	//			},
	//		},
	//	}
	//
	//	for _ = range 2 {
	//		payload, err := json.Marshal(msg)
	//		assert.NoError(t, err)
	//
	//		mqtt_client.Client().Publish("tripg", 2, false, payload)
	//		time.Sleep(500 * time.Millisecond)
	//
	//	}
	//}()
	//
	//go func() {
	//	for {
	//		time.Sleep(10 * time.Second)
	//		c.closeCh[rosTopic] <- struct{}{}
	//	}
	//}()

	//time.Sleep(10 * time.Second)
	ctx2, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()
	for {
		select {
		case <-ctx2.Done():
			return
		default:
		}
	}
}
