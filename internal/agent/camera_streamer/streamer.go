package camera_streamer

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"

	"github.com/okieraised/monitoring-agent/internal/constants"
	"github.com/okieraised/monitoring-agent/internal/infrastructure/log"
	sensor_msgs_msg "github.com/okieraised/monitoring-agent/internal/ros_msgs/sensor_msgs/msg"
	"github.com/okieraised/rclgo/pkg/rclgo"
	"github.com/pion/webrtc/v4"
	"github.com/pion/webrtc/v4/pkg/media"
	ffmpeg "github.com/u2takey/ffmpeg-go"
	"golang.org/x/sync/errgroup"
)

func NewCameraStreamer(ctx context.Context) error {
	log.Default().Info("Starting ROS2 camera streaming node")

	g, ctx := errgroup.WithContext(ctx)

	iceServers := []webrtc.ICEServer{
		{
			URLs: []string{"stun:stun.l.google.com:19302"},
		},
	}

	// Create a new RTCPeerConnection
	config := webrtc.Configuration{
		ICEServers: iceServers,
	}
	peerConnection, err := webrtc.NewPeerConnection(config)
	if err != nil {
		wErr := fmt.Errorf("failed to create peer connection: %v", err)
		log.Default().Error(wErr.Error())
		return wErr
	}

	// Handle ICE connection state changes
	peerConnection.OnICEConnectionStateChange(func(state webrtc.ICEConnectionState) {
		log.Default().Info(fmt.Sprintf("ICE Connection State has changed: %s", state.String()))
	})

	videoTrack, err := webrtc.NewTrackLocalStaticSample(webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeH264}, "video", "pion")
	if err != nil {
		wErr := fmt.Errorf("failed to create new track: %v", err)
		log.Default().Error(wErr.Error())
		return wErr
	}

	rtpSender, err := peerConnection.AddTrack(videoTrack)
	if err != nil {
		wErr := fmt.Errorf("failed to add track to peer: %v", err)
		log.Default().Error(wErr.Error())
		return wErr
	}

	go func() {
		rtcpBuf := make([]byte, 1500)
		for {
			if _, _, rtcpErr := rtpSender.Read(rtcpBuf); rtcpErr != nil {
				return
			}
		}
	}()

	// Wait for the offer to be pasted
	offer := webrtc.SessionDescription{}
	decode(&offer)

	// Set the remote SessionDescription
	err = peerConnection.SetRemoteDescription(offer)
	if err != nil {
		wErr := fmt.Errorf("failed to set remote description: %v", err)
		log.Default().Error(wErr.Error())
		return wErr
	}

	// Create an answer
	answer, err := peerConnection.CreateAnswer(nil)
	if err != nil {
		wErr := fmt.Errorf("failed to create answer: %v", err)
		log.Default().Error(wErr.Error())
	}

	// Create channel that is blocked until ICE Gathering is complete
	gatherComplete := webrtc.GatheringCompletePromise(peerConnection)

	// Sets the LocalDescription, and starts our UDP listeners
	err = peerConnection.SetLocalDescription(answer)
	if err != nil {
		wErr := fmt.Errorf("failed to set local description: %v", err)
		log.Default().Error(wErr.Error())
		return wErr
	}

	<-gatherComplete

	// Output the answer in base64 so we can paste it in browser
	fmt.Println(encode(peerConnection.LocalDescription()))

	// Start pushing buffers on these tracks
	//writeH264ToTrack(videoTrack)
	// start subscription
	g.Go(func() error {
		return newCameraSubscriber(ctx, videoTrack)
	})

	return g.Wait()
}

func newCameraSubscriber(ctx context.Context, track *webrtc.TrackLocalStaticSample) error {
	node, err := rclgo.NewNode(constants.NodeNameCameraStreaming, "")
	if err != nil {
		log.Default().Error(fmt.Sprintf("Failed to create ROS2 camera streaming node: %v", err))
		return err
	}
	defer func(node *rclgo.Node) {
		cErr := node.Close()
		if cErr != nil && err == nil {
			err = cErr
		}
	}(node)

	sub, err := sensor_msgs_msg.NewCompressedImageSubscription(
		node,
		"/camera/front_robot/color/image_raw/compressed",
		nil,
		func(msg *sensor_msgs_msg.CompressedImage, info *rclgo.MessageInfo, err error) {

		})
	defer func(sub *sensor_msgs_msg.CompressedImageSubscription) {
		cErr := sub.Close()
		if cErr != nil && err == nil {
			err = cErr
		}
	}(sub)

	ws, err := rclgo.NewWaitSet()
	if err != nil {
		return fmt.Errorf("failed to create waitset: %v", err)
	}
	defer func(ws *rclgo.WaitSet) {
		cErr := ws.Close()
		if cErr != nil && err == nil {
			err = cErr
		}
	}(ws)
	ws.AddSubscriptions(sub.Subscription)

	return ws.Run(ctx)
}

func encode(obj *webrtc.SessionDescription) string {
	b, err := json.Marshal(obj)
	if err != nil {
		panic(err)
	}

	return base64.StdEncoding.EncodeToString(b)
}

// Decode a base64 and unmarshal JSON into a SessionDescription.
func decode(obj *webrtc.SessionDescription) {
	in, err := os.ReadFile("/home/pham/workspace/repo/monitoring-agent/session.txt")
	if err != nil {
		panic(err)
	}

	b, err := base64.StdEncoding.DecodeString(string(in))
	if err != nil {
		panic(err)
	}

	if err = json.Unmarshal(b, obj); err != nil {
		panic(err)
	}
}

func writeH264ToTrack(track *webrtc.TrackLocalStaticSample) {
	cmd := exec.Command("ffmpeg",
		"-framerate", "30",
		"-video_size", "640x480",
		"-i", "/dev/video0",
		"-pix_fmt", "yuv420p",
		"-f", "rawvideo",
		"pipe:1",
	)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Println(err)
		return
	}

	if err := cmd.Start(); err != nil {
		fmt.Println(err)
		return
	}

	reader := bufio.NewReader(stdout)
	width := 640
	height := 480
	frameSize := width * height * 3 / 2

	for {
		frame := make([]byte, frameSize)

		n, err := reader.Read(frame)
		if err != nil {
			if err == io.EOF {
				break
			} else {
				fmt.Println(err)
				return
			}
		}

		if err := track.WriteSample(media.Sample{Data: frame[:n], Duration: 33 * time.Millisecond}); err != nil {
			panic(err)
		}
	}

	cmd.Process.Kill()

}

func JPEGToH264Frame(jpeg []byte, width, height, fps int) ([]byte, error) {
	in := bytes.NewReader(jpeg)
	var out bytes.Buffer
	err := ffmpeg.
		Input("pipe:0",
			ffmpeg.KwArgs{"f": "mjpeg"},
			ffmpeg.KwArgs{"r": fps},
		).
		// single frame, baseline H.264, Annex-B
		Output("pipe:1",
			ffmpeg.KwArgs{
				"f":           "h264",
				"vcodec":      "libx264", // swap to h264_nvenc / h264_vaapi if you have HW
				"pix_fmt":     "yuv420p",
				"preset":      "veryfast",
				"tune":        "zerolatency",
				"profile:v":   "baseline",
				"level":       "3.1",
				"g":           1,
				"vframes":     1,
				"x264-params": "keyint=1:min-keyint=1:scenecut=0:repeat-headers=1",
			},
		).
		WithInput(in).
		WithOutput(&out).
		Run()
	if err != nil {
		return nil, fmt.Errorf("ffmpeg transcode failed: %w", err)
	}
	return out.Bytes(), nil
}
