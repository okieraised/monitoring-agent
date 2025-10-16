package webrtc_viewer

import (
	"context"
	"fmt"
	"sync/atomic"

	"github.com/okieraised/monitoring-agent/internal/infrastructure/log"
	sensor_msgs_msg "github.com/okieraised/monitoring-agent/internal/ros_msgs/sensor_msgs/msg"
	"github.com/okieraised/monitoring-agent/internal/utilities"
	"github.com/okieraised/rclgo/humble"
)

const (
	ImageMessageType           = "sensor_msgs/msg/Image"
	CompressedImageMessageType = "sensor_msgs/msg/CompressedImage"
)

var supportedTopicMsgTypes = map[string]struct{}{
	ImageMessageType:           {},
	CompressedImageMessageType: {},
}

func newCameraSubscriberV2(ctx context.Context, node *humble.Node, topicName string, frameCh chan<- []byte, enableCh <-chan bool) error {
	log.Default().Info(fmt.Sprintf("Started initializing new [%s] subscriber", topicName))
	var enabled atomic.Bool

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case v, ok := <-enableCh:
				if !ok {
					return
				}
				enabled.Store(v)
			}
		}
	}()

	topicMsgsMap, err := node.GetTopicNamesAndTypes(true)
	if err != nil {
		wErr := fmt.Errorf("failed to get topic names and types: %w", err)
		log.Default().Error(wErr.Error())
		return wErr
	}

	msgTypes, ok := topicMsgsMap[topicName]
	if !ok {
		wErr := fmt.Errorf("failed to get topic %s' message type: %v", topicName, err)
		log.Default().Error(wErr.Error())
		return wErr
	}

	if len(msgTypes) == 0 {
		wErr := fmt.Errorf("no topic message types found for topic %s", topicName)
		log.Default().Error(wErr.Error())
		return wErr
	}

	if _, ok = supportedTopicMsgTypes[msgTypes[0]]; !ok {
		wErr := fmt.Errorf("topic %s has unsupported type for webrtc streaming: %v", topicName, msgTypes[0])
		log.Default().Error(wErr.Error())
		return wErr
	}

	ws, err := humble.NewWaitSet()
	if err != nil {
		return fmt.Errorf("failed to create waitset: %w", err)
	}
	defer func() {
		cErr := ws.Close()
		if cErr != nil && err == nil {
			wErr := fmt.Errorf("failed to close waitset: %v", cErr)
			log.Default().Error(wErr.Error())
			err = wErr
		}
	}()

	switch msgTypes[0] {
	case CompressedImageMessageType:
		sub, err := sensor_msgs_msg.NewCompressedImageSubscription(
			node,
			topicName,
			nil,
			func(msg *sensor_msgs_msg.CompressedImage, info *humble.MessageInfo, cbErr error) {
				select {
				case <-ctx.Done():
					return
				default:
				}

				// If not enabled, ignore the message.
				if !enabled.Load() {
					return
				}

				select {
				case <-ctx.Done():
					return
				case frameCh <- msg.Data:
				default:
				}
			},
		)
		if err != nil {
			return fmt.Errorf("failed to subscribe topic: %w", err)
		}
		defer func() {
			cErr := sub.Close()
			if cErr != nil && err == nil {
				wErr := fmt.Errorf("failed to close subscription: %v", cErr)
				log.Default().Error(wErr.Error())
				err = wErr
			}
		}()
		ws.AddSubscriptions(sub.Subscription)
	case ImageMessageType:
		sub, err := sensor_msgs_msg.NewImageSubscription(
			node,
			topicName,
			nil,
			func(msg *sensor_msgs_msg.Image, info *humble.MessageInfo, cbErr error) {
				select {
				case <-ctx.Done():
					return
				default:
				}

				// If not enabled, ignore the message.
				if !enabled.Load() {
					return
				}

				// Encode only when enabled (work is non-trivial).
				content, encErr := utilities.EncodeROSImageToBytes(
					int(msg.Width), int(msg.Height),
					msg.Encoding, int(msg.Step),
					msg.IsBigendian != 0, msg.Data,
					"png", 0,
				)
				if encErr != nil {
					log.Default().Error(encErr.Error())
					return
				}

				select {
				case <-ctx.Done():
					return
				case frameCh <- content:
				default:
				}
			},
		)
		if err != nil {
			return fmt.Errorf("failed to subscribe topic: %w", err)
		}
		defer func() {
			cErr := sub.Close()
			if cErr != nil && err == nil {
				wErr := fmt.Errorf("failed to close subscription: %v", cErr)
				log.Default().Error(wErr.Error())
				err = wErr
			}
		}()
		ws.AddSubscriptions(sub.Subscription)
	}

	return ws.Run(ctx)
}
