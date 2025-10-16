package webrtc_viewer

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/google/uuid"
	"github.com/okieraised/monitoring-agent/internal/config"
	"github.com/okieraised/monitoring-agent/internal/constants"
	"github.com/okieraised/monitoring-agent/internal/infrastructure/h264_encoder"
	"github.com/okieraised/monitoring-agent/internal/infrastructure/log"
	"github.com/okieraised/monitoring-agent/internal/infrastructure/mqtt_client"
	"github.com/okieraised/rclgo/humble"
	"github.com/pion/webrtc/v4"
	"github.com/pion/webrtc/v4/pkg/media"
	"github.com/pion/webrtc/v4/pkg/media/h264reader"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"golang.org/x/sync/errgroup"
)

const (
	qosExactlyOnce byte = 2
)

const (
	webRTCMQTTMsgTypeSignaling   = "Signaling"
	webRTCMQTTMsgTypeNegotiating = "Negotiating"
	webRTCMQTTMsgTypeNegotiated  = "Negotiated"
)

var webRTCValidCommandType = map[string]struct{}{
	webRTCMQTTMsgTypeSignaling:   {},
	webRTCMQTTMsgTypeNegotiating: {},
}

type WebRTCSDPAnswerMsg struct {
	Timestamp time.Time `json:"timestamp"`
	AgentID   string    `json:"agentId"`
	RobotID   string    `json:"robotId"`
	SDP       string    `json:"sdp"`
}

type WebrtcSDPOfferMsg struct {
	HeaderID     int       `json:"headerId"`
	Timestamp    time.Time `json:"timestamp"`
	Version      string    `json:"version"`
	Manufacturer string    `json:"manufacturer"`
	SerialNumber string    `json:"serialNumber"`
	AgentID      string    `json:"agentId"`
	Command      struct {
		Type     string    `json:"type"`
		ID       uuid.UUID `json:"id"`
		Argument struct {
			RosTopic string `json:"rosTopic"`
			SDP      string `json:"sdp"`
		} `json:"argument"`
	} `json:"command"`
}

func (req *WebrtcSDPOfferMsg) validate() error {
	if _, ok := webRTCValidCommandType[req.Command.Type]; !ok {
		return errors.New("invalid webrtc command type")
	}

	return nil

}

type WebRTCViewer struct {
	ctx       context.Context
	cancel    context.CancelFunc
	client    mqtt.Client
	node      *humble.Node
	sem       map[string]struct{}
	wg        sync.WaitGroup
	frameCh   map[string]chan []byte
	enableCh  map[string]chan bool
	closeCh   map[string]chan struct{}
	errCh     map[string]chan error
	mu        sync.RWMutex
	closeOnce sync.Once
}

func NewWebRTCViewer(ctx context.Context) (*WebRTCViewer, error) {
	log.Default().Info("Starting WebRTC streaming daemon")

	node, err := humble.NewNode(constants.NodeNameWebRTCStreaming, "")
	if err != nil {
		log.Default().Error(fmt.Sprintf("Failed to create ROS2 camera streaming node: %v", err))
		return nil, err
	}

	cCtx, cancel := context.WithCancel(ctx)
	webRTCViewer := &WebRTCViewer{
		ctx:      cCtx,
		cancel:   cancel,
		client:   mqtt_client.Client(),
		node:     node,
		sem:      make(map[string]struct{}),
		frameCh:  make(map[string]chan []byte),
		enableCh: make(map[string]chan bool),
		closeCh:  make(map[string]chan struct{}),
		errCh:    make(map[string]chan error),
	}

	return webRTCViewer, nil
}

func (w *WebRTCViewer) Start() error {
	token := w.client.Subscribe(viper.GetString(config.MqttWebRTCOfferTopic), qosExactlyOnce, w.handler)
	if token.WaitTimeout(5*time.Second) && token.Error() != nil {
		return token.Error()
	}
	return nil
}

func (w *WebRTCViewer) handler(client mqtt.Client, msg mqtt.Message) {

	var payload WebrtcSDPOfferMsg
	err := json.Unmarshal(msg.Payload(), &payload)
	if err != nil {
		wErr := errors.Wrapf(err, "failed to unmarshal payload")
		log.Default().Error(wErr.Error())
		return
	}
	log.Default().Debug(fmt.Sprintf("WebRTC signaling message received: %v", payload))

	err = payload.validate()
	if err != nil {
		wErr := errors.Wrapf(err, "failed to validate payload")
		log.Default().Error(wErr.Error())
		return
	}

	switch payload.Command.Type {
	case webRTCMQTTMsgTypeSignaling:
		w.mu.RLock()
		if _, ok := w.sem[payload.Command.Argument.RosTopic]; ok {
			w.mu.RUnlock()
			log.Default().Error(fmt.Sprintf("There is an existing subscriber on topic [%s]", payload.Command.Argument.RosTopic))
			return
		}
		w.mu.RUnlock()
		w.mu.Lock()
		w.sem[payload.Command.Argument.RosTopic] = struct{}{}
		w.frameCh[payload.Command.Argument.RosTopic] = make(chan []byte, 64)
		w.enableCh[payload.Command.Argument.RosTopic] = make(chan bool, 1)
		w.closeCh[payload.Command.Argument.RosTopic] = make(chan struct{})
		w.errCh[payload.Command.Argument.RosTopic] = make(chan error, 1)
		w.mu.Unlock()
		w.signalingHandler(payload)
	case webRTCMQTTMsgTypeNegotiating:
		w.negotiatingHandler(payload)
	case webRTCMQTTMsgTypeNegotiated:
	default:
		wErr := fmt.Errorf("invalid payload type: %s", payload.Command.Type)
		log.Default().Error(wErr.Error())
		return
	}
}

func (w *WebRTCViewer) negotiatingHandler(payload WebrtcSDPOfferMsg) {
	go func() {
		ctx, cancel := context.WithCancel(w.ctx)
		defer func() {
			if rec := recover(); rec != nil {
				log.Default().Panic(fmt.Sprintf("recovered from panic: %v: %s", rec, debug.Stack()))
			}
			cancel()
		}()

		g, runCtx := errgroup.WithContext(ctx)

		iceServers := []webrtc.ICEServer{
			{
				URLs: []string{
					"stun:stun.l.google.com:19302",
				},
			},
		}
		pc, err := webrtc.NewPeerConnection(webrtc.Configuration{ICEServers: iceServers})
		if err != nil {
			wErr := errors.Wrapf(err, "failed to create peer connection")
			log.Default().Error(wErr.Error())
			w.errCh[payload.Command.Argument.RosTopic] <- wErr
			return
		}
		defer func() {
			cErr := pc.Close()
			if cErr != nil && err == nil {
				log.Default().Error(fmt.Sprintf("peer connection close: %v", cErr))
				w.errCh[payload.Command.Argument.RosTopic] <- cErr
			}
		}()

		iceConnectedCtx, iceConnectedCtxCancel := context.WithCancel(runCtx)
		defer iceConnectedCtxCancel()

		pc.OnICEConnectionStateChange(func(state webrtc.ICEConnectionState) {
			log.Default().Info(fmt.Sprintf("ICE Connection State changed: %s", state.String()))
			if state == webrtc.ICEConnectionStateConnected || state == webrtc.ICEConnectionStateCompleted {
				iceConnectedCtxCancel()
			}
		})

		// Signal when the peer closes/disconnects, so we can exit cleanly.
		peerClosed := make(chan struct{})
		var closePeerClosed sync.Once

		pc.OnConnectionStateChange(func(s webrtc.PeerConnectionState) {
			log.Default().Info(fmt.Sprintf("Peer Connection State changed: %s", s.String()))
			if s == webrtc.PeerConnectionStateFailed ||
				s == webrtc.PeerConnectionStateDisconnected ||
				s == webrtc.PeerConnectionStateClosed {
				closePeerClosed.Do(func() { close(peerClosed) })
			}
		})

		// Create a video track and add to peer
		trackID := fmt.Sprintf("video_%s", strings.ReplaceAll(payload.Command.Argument.RosTopic, "/", "_"))
		videoTrack, err := webrtc.NewTrackLocalStaticSample(
			webrtc.RTPCodecCapability{
				MimeType: webrtc.MimeTypeH264,
			},
			trackID,
			viper.GetString(config.AgentID),
		)
		if err != nil {
			wErr := errors.Wrapf(err, "failed to create track")
			log.Default().Error(wErr.Error())
			w.errCh[payload.Command.Argument.RosTopic] <- wErr
			return
		}

		rtpSender, err := pc.AddTrack(videoTrack)
		if err != nil {
			wErr := errors.Wrapf(err, "failed to add track")
			log.Default().Error(wErr.Error())
			w.errCh[payload.Command.Argument.RosTopic] <- wErr
			return
		}

		// Read incoming RTCP packets
		// Before these packets are returned, they are processed by interceptors. For things
		// like NACK this needs to be called.
		g.Go(func() error {
			rtcpBuf := make([]byte, 1500)
			for {
				if _, _, rtcpErr := rtpSender.Read(rtcpBuf); rtcpErr != nil {
					return nil
				}
			}
		})

		g.Go(func() error {
			enc, out, encErr := h264_encoder.NewH264Encoder(runCtx, nil)
			if encErr != nil {
				return fmt.Errorf("failed to create new h264 encoder: %w", encErr)
			}
			defer func() {
				cErr := enc.Close()
				if cErr != nil {
					log.Default().Error(fmt.Sprintf("failed to close encoder: %v", cErr))
					if encErr == nil {
						encErr = cErr
					}
				}
			}()

			h264, hErr := h264reader.NewReader(out)
			if hErr != nil {
				return fmt.Errorf("failed to create h264 reader: %w", hErr)
			}

			subCtx, subCancel := context.WithCancel(runCtx)
			defer subCancel()
			subGCtx, subCtx := errgroup.WithContext(subCtx)

			// Feed JPEG frames from ROS into the encoder stdin.
			subGCtx.Go(func() error {
				for {
					select {
					case <-subCtx.Done():
						return subCtx.Err()
					case frame, ok := <-w.frameCh[payload.Command.Argument.RosTopic]:
						if !ok {
							return nil
						}
						wErr := enc.WriteFrame(frame)
						if wErr != nil && !errors.Is(wErr, io.EOF) {
							return fmt.Errorf("enc.WriteFrame: %w", wErr)
						}
					}
				}
			})

			select {
			case <-iceConnectedCtx.Done():
				// ok
			case <-runCtx.Done():
				_ = subGCtx.Wait()
				return runCtx.Err()
			}

			for {
				select {
				case <-runCtx.Done():
					_ = subGCtx.Wait()
					return runCtx.Err()
				default:
					nal, nErr := h264.NextNAL()
					if nErr != nil {
						if errors.Is(nErr, io.EOF) {
							log.Default().Info("H264 stream ended")
							_ = subGCtx.Wait()
							return nil
						}
						_ = subGCtx.Wait()
						return errors.Wrapf(nErr, "H264 stream ended")
					}

					if wErr := videoTrack.WriteSample(media.Sample{
						Data:     nal.Data,
						Duration: 33 * time.Millisecond,
					}); wErr != nil {
						_ = subGCtx.Wait()
						return errors.Wrap(wErr, "failed to write sample")
					}
				}
			}
		})

		offer, err := w.decodeSessionDescription(payload.Command.Argument.SDP)
		if err != nil {
			wErr := errors.Wrapf(err, "failed to decode session description")
			log.Default().Error(wErr.Error())
			w.errCh[payload.Command.Argument.RosTopic] <- wErr
			return
		}

		if err = pc.SetRemoteDescription(offer); err != nil {
			wErr := errors.Wrapf(err, "failed to set remote description")
			log.Default().Error(wErr.Error())
			w.errCh[payload.Command.Argument.RosTopic] <- wErr
			return
		}

		answer, err := pc.CreateAnswer(nil)
		if err != nil {
			wErr := errors.Wrapf(err, "failed to create answer")
			log.Default().Error(wErr.Error())
			w.errCh[payload.Command.Argument.RosTopic] <- wErr
			return
		}

		gatherComplete := webrtc.GatheringCompletePromise(pc)

		if err = pc.SetLocalDescription(answer); err != nil {
			wErr := errors.Wrapf(err, "failed to set local description")
			log.Default().Error(wErr.Error())
			w.errCh[payload.Command.Argument.RosTopic] <- wErr
			return
		}

		<-gatherComplete

		w.enableCh[payload.Command.Argument.RosTopic] <- true

		// Encode the session description then publish it back to mqtt
		sdp, err := w.encodeSessionDescription(pc.LocalDescription())
		if err != nil {
			wErr := errors.Wrapf(err, "failed to encode session description")
			log.Default().Error(wErr.Error())
			w.errCh[payload.Command.Argument.RosTopic] <- wErr
			return
		}

		sdpPayload := WebRTCSDPAnswerMsg{
			SDP: sdp,
		}

		bSDPPayload, err := json.Marshal(sdpPayload)
		if err != nil {
			wErr := errors.Wrapf(err, "failed to encode session description")
			log.Default().Error(wErr.Error())
			w.errCh[payload.Command.Argument.RosTopic] <- wErr
			return
		}

		token := w.client.Publish(viper.GetString(config.MqttWebRTCAnswerTopic), qosExactlyOnce, false, bSDPPayload)
		if token.WaitTimeout(5*time.Second) && token.Error() != nil {
			w.errCh[payload.Command.Argument.RosTopic] <- token.Error()
			return
		}

		done := make(chan error, 1)
		go func() {
			done <- g.Wait()
		}()

		select {
		case err = <-done:
			w.errCh[payload.Command.Argument.RosTopic] <- err
			return

		case <-peerClosed:
			// Remote hung up: cascade cancel, close PC to unblock RTCP reader, then wait.
			cancel()
			_ = pc.Close()
			w.errCh[payload.Command.Argument.RosTopic] <- <-done
			return

		case <-ctx.Done():
			// Ctrl-C or parent canceled: cascade cancel, close PC, then wait.
			cancel()
			_ = pc.Close()

			// Grace period
			shutdownTimer := time.NewTimer(1 * time.Second)
			defer shutdownTimer.Stop()

			select {
			case dErr := <-done:
				w.errCh[payload.Command.Argument.RosTopic] <- dErr
				return
			case <-shutdownTimer.C:
				w.errCh[payload.Command.Argument.RosTopic] <- fmt.Errorf("shutdown timed out")
				return
			}
		}
	}()
}

func (w *WebRTCViewer) decodeSessionDescription(sdp string) (webrtc.SessionDescription, error) {
	offer := webrtc.SessionDescription{}

	b, err := base64.StdEncoding.DecodeString(sdp)
	if err != nil {
		return offer, err
	}
	if err = json.Unmarshal(b, &offer); err != nil {
		return offer, err
	}
	return offer, nil
}

func (w *WebRTCViewer) encodeSessionDescription(sdp *webrtc.SessionDescription) (string, error) {

	b, err := json.Marshal(sdp)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(b), nil
}

func (w *WebRTCViewer) signalingHandler(payload WebrtcSDPOfferMsg) {
	w.wg.Add(1)
	go func() {
		ctx, cancel := context.WithCancel(w.ctx)

		defer func() {
			if rec := recover(); rec != nil {
				log.Default().Panic(fmt.Sprintf("recovered from panic: %v: %s", rec, debug.Stack()))
			}
			cancel()
			w.wg.Done()
			w.mu.Lock()
			delete(w.sem, payload.Command.Argument.RosTopic)
			delete(w.frameCh, payload.Command.Argument.RosTopic)
			delete(w.enableCh, payload.Command.Argument.RosTopic)
			delete(w.closeCh, payload.Command.Argument.RosTopic)
			w.mu.Unlock()
		}()

		go func() {
			err := newCameraSubscriberV2(
				ctx,
				w.node,
				payload.Command.Argument.RosTopic,
				w.frameCh[payload.Command.Argument.RosTopic],
				w.enableCh[payload.Command.Argument.RosTopic],
			)
			if err != nil && !errors.Is(err, context.Canceled) {
				wErr := errors.Wrap(err, "error creating new camera subscriber")
				log.Default().Error(wErr.Error())
				w.errCh[payload.Command.Argument.RosTopic] <- wErr
			}
		}()

		for {
			select {
			case <-ctx.Done():
				log.Default().Info(fmt.Sprintf("Shutting down subscriber node for topic [%s]", payload.Command.Argument.RosTopic))
				return
			case <-w.closeCh[payload.Command.Argument.RosTopic]:
				log.Default().Info(fmt.Sprintf("Closing subscriber node for topic [%s]", payload.Command.Argument.RosTopic))
				return
			case err := <-w.errCh[payload.Command.Argument.RosTopic]:
				if !errors.Is(err, context.Canceled) {
					log.Default().Error(err.Error())
				}
				return
			}
		}
	}()
}

func (w *WebRTCViewer) Close() {
	w.closeOnce.Do(func() {
		w.cancel()
		w.wg.Wait()
		for _, ch := range w.frameCh {
			close(ch)
		}
		for _, ch := range w.enableCh {
			close(ch)
		}
		for _, ch := range w.closeCh {
			close(ch)
		}
		for _, ch := range w.errCh {
			close(ch)

		}
		err := w.node.Close()
		if err != nil {
			wErr := errors.Wrapf(err, "failed to close node")
			log.Default().Error(wErr.Error())
			return
		}
	})
}
