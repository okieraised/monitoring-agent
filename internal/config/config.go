package config

const (
	AgentID               = "agent.id"
	AgentEnableMonitoring = "agent.enable_monitoring"
	AgentMonitoringPort   = "agent.monitoring_port"
	AgentLogLevel         = "agent.log_level"
	AgentHTTPSecure       = "agent.http_secure"
	AgentHTTPPort         = "agent.http_port"
	AgentGRPCSecure       = "agent.grpc_secure"
	AgentGRPCPort         = "agent.grpc_port"
	AgentTLSCertFile      = "agent.tls_cert_file"
	AgentTLSKeyFile       = "agent.tls_key_file"
	AgentTLSCACertFile    = "agent.tls_ca_cert_file"
	AgentTLSClientCAFile  = "agent.tls_client_ca_file"
)

const (
	MqttEndpoint              = "mqtt.endpoint"
	MqttCleanSession          = "mqtt.clean_session"
	MqttClientId              = "mqtt.client_id"
	MqttAutoReconnect         = "mqtt.auto_reconnect"
	MqttConnectRetry          = "mqtt.connect_retry"
	MqttMaxConnectInterval    = "mqtt.max_connect_interval"
	MqttWriteTimeout          = "mqtt.write_timeout"
	MqttPingTimeout           = "mqtt.ping_timeout"
	MqttKeepAliveDuration     = "mqtt.keep_alive_duration"
	MqttResumeSubs            = "mqtt.resume_subs"
	MqttConnectTimeout        = "mqtt.connect_timeout"
	MqttConnectRetryInterval  = "mqtt.connect_retry_interval"
	MqttTLSInsecureSkipVerify = "mqtt.tls_insecure_skip_verify"
	MqttWebRTCOfferTopic      = "mqtt.webrtc_offer_topic"
	MqttWebRTCAnswerTopic     = "mqtt.webrtc_answer_topic"
)

const (
	S3Region                = "s3.region"
	S3Endpoint              = "s3.endpoint"
	S3AccessKey             = "s3.access_key"
	S3SecretKey             = "s3.secret_key"
	S3UsePathStyle          = "s3.use_path_style"
	S3TLSInsecureSkipVerify = "s3.tls_insecure_skip_verify"
)

const (
	WebRTCICEServer = "webrtc.ice_server"
)
