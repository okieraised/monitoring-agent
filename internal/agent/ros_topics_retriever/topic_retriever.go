package ros_topics_retriever

import (
	"context"
	"fmt"
	"time"

	"github.com/okieraised/monitoring-agent/internal/constants"
	"github.com/okieraised/monitoring-agent/internal/infrastructure/log"
	"github.com/okieraised/rclgo/pkg/rclgo"
)

type Config struct {
	Interval int64
}

type Option func(*Config)

func defaultConfig() Config {
	return Config{
		Interval: 5,
	}
}

func WithInterval(i int64) Option {
	return func(o *Config) {
		o.Interval = i
	}
}

func NewROS2TopicRetriever(ctx context.Context, optFns ...Option) error {
	log.Default().Info("Starting ROS2 topic retriever node")

	conf := defaultConfig()
	for _, fn := range optFns {
		if fn != nil {
			fn(&conf)
		}
	}

	node, err := rclgo.NewNode(constants.NodeNameTopicRetriever, "")
	if err != nil {
		wErr := fmt.Errorf("failed to create ROS2 topic retriever node: %v", err)
		log.Default().Error(wErr.Error())
		return wErr
	}
	defer func(node *rclgo.Node) {
		cErr := node.Close()
		if cErr != nil && err == nil {
			err = cErr
		}
	}(node)

	ticker := time.NewTicker(time.Duration(conf.Interval) * time.Second)
	defer ticker.Stop()

	typesByTopic, err := node.GetTopicNamesAndTypes(true)
	if err != nil {
		log.Default().Error(fmt.Sprintf("Failed to retrieve topic names and types: %v", err))
	}

	for {
		select {
		case <-ctx.Done():
			log.Default().Info("Shutting down topic retriever node")
			return nil
		case <-ticker.C:
			typesByTopic, err = node.GetTopicNamesAndTypes(true)
			if err != nil {
				log.Default().Error(fmt.Sprintf("Failed to retrieve topic names and types: %v", err))
				continue
			}
			log.Default().Debug(fmt.Sprintf("%v", typesByTopic))
		}
	}
}
