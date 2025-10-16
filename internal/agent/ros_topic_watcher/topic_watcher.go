package ros_topic_watcher

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/okieraised/monitoring-agent/internal/constants"
	"github.com/okieraised/monitoring-agent/internal/infrastructure/log"
	sensor_msgs_msg "github.com/okieraised/monitoring-agent/internal/ros_msgs/sensor_msgs/msg"
	"github.com/okieraised/rclgo/humble"
)

var topicsToWatch = []string{
	"/camera/front_robot/color/image_raw/compressed",
	"/camera/front_robot/depth/image_rect_raw/compressedDepth",
}

func NewROS2TopicWatcher(ctx context.Context) error {
	log.Default().Info("Starting ROS2 topic watcher node")

	node, err := humble.NewNode(constants.NodeNameTopicWatcher, "")
	if err != nil {
		log.Default().Error(fmt.Sprintf("Failed to create ROS2 topic watcher node: %v", err))
		return err
	}
	defer func(node *humble.Node) {
		cErr := node.Close()
		if cErr != nil && err == nil {
			err = cErr
		}
	}(node)

	log.Default().Info("Warming up topic watcher node")
	time.Sleep(1 * time.Second)
	typesByTopic, err := node.GetTopicNamesAndTypes(true)
	if err != nil {
		wErr := fmt.Errorf("failed to get topic names and types: %v", err)
		log.Default().Error(wErr.Error())
		return wErr
	}

	wg := new(sync.WaitGroup)
	for _, topic := range topicsToWatch {
		msgTypes := typesByTopic[topic]
		if len(msgTypes) == 0 {
			log.Default().Info(fmt.Sprintf("Skipping topic watcher for topic [%s]: no type found", topic))
			continue
		}

		msgType := msgTypes[0]
		parts := strings.Split(msgType, "/")
		pkg, msg := parts[0], parts[len(parts)-1]
		log.Default().Info(fmt.Sprintf("topic package: %s, message type: %s", pkg, msg))

		typeSupport, cErr := humble.LoadDynamicMessageTypeSupport(pkg, msg)
		if cErr != nil {
			wErr := fmt.Errorf("failed to load dynamic message type support for topic [%s]: %v", topic, cErr)
			log.Default().Error(wErr.Error())
			continue
		}

		subscriptionHandler := matchTopicMessageType(msgType)

		wg.Add(1)
		go topicWatcherHandler(ctx, wg, node, typeSupport, topic, subscriptionHandler)
	}
	wg.Wait()

	return nil
}

func topicWatcherHandler(ctx context.Context, wg *sync.WaitGroup, node *humble.Node, typeSupport humble.MessageTypeSupport, topic string, subHandler func(s *humble.Subscription)) {
	defer wg.Done()
	log.Default().Info(fmt.Sprintf("Started subscribing to topic [%s]", topic))

	sub, err := node.NewSubscription(topic, typeSupport, nil, subHandler)
	if err != nil {
		log.Default().Error(fmt.Errorf("failed to create subscription: %v", err).Error())
		return
	}
	defer func(sub *humble.Subscription) {
		cErr := sub.Close()
		if cErr != nil && err == nil {
			err = cErr
		}
	}(sub)

	ws, err := humble.NewWaitSet()
	if err != nil {
		wErr := fmt.Errorf("failed to create new waitset: %v", err)
		log.Default().Error(wErr.Error())
		return
	}
	defer func(ws *humble.WaitSet) {
		cErr := ws.Close()
		if cErr != nil && err == nil {
			err = cErr
		}
	}(ws)

	ws.AddSubscriptions(sub)
	err = ws.Run(ctx)
	if err != nil && !errors.Is(err, context.Canceled) {
		log.Default().Error(fmt.Sprintf("Failed to run waitset: %v", err))
	}
}

func matchTopicMessageType(msgType string) func(s *humble.Subscription) {
	switch msgType {
	case "sensor_msgs/msg/CompressedImage":
		return compressedImageHandler
	case "":

	}

	return nil
}

var compressedImageHandler = func(s *humble.Subscription) {
	raw, _, err := s.TakeSerializedMessage()
	if err != nil {
		log.Default().Error(fmt.Sprintf("Failed to take serialized message: %v", err))
		return
	}

	msg, err := humble.Deserialize(raw, sensor_msgs_msg.CompressedImageTypeSupport)
	if err != nil {
		log.Default().Error(fmt.Sprintf("Failed to deserialize message: %v", err))
	}
	deserialized := msg.(*sensor_msgs_msg.CompressedImage)

	fmt.Println(deserialized.Format)
}
