package constants

import "time"

const (
	MqttDefaultWriteTimeout         = 10 * time.Second
	MqttDefaultKeepAlive            = 30 * time.Second
	MqttDefaultPingTimeout          = 5 * time.Second
	MqttDefaultMaxReconnectInterval = 30 * time.Second
	MqttDefaultConnectTimeout       = 10 * time.Second
	MqttDefaultConnectRetryInterval = 10 * time.Second
)
