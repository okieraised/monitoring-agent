package constants

import "time"

const (
	AgentDefaultHTTPPort       = 8080
	AgentDefaultGRPCPort       = 7070
	AgentDefaultMonitoringPort = 6060
)

const (
	GraceWaitPeriod = 10 * time.Second
)

const (
	MqttDefaultWriteTimeout         = 10 * time.Second
	MqttDefaultKeepAlive            = 30 * time.Second
	MqttDefaultPingTimeout          = 5 * time.Second
	MqttDefaultMaxReconnectInterval = 30 * time.Second
	MqttDefaultConnectTimeout       = 10 * time.Second
	MqttDefaultConnectRetryInterval = 10 * time.Second
)
