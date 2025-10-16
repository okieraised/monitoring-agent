package h264_encoder

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/okieraised/monitoring-agent/internal/infrastructure/log"
)

// EncoderOptions to tune the encoder
type EncoderOptions struct {
	// Expected input frame rate. Used to help ffmpeg pacing; frames are still
	// delivered as fast as you push them.
	FPS int
	// Output bitrate (e.g. "3M", "800k"). If empty, CRF mode is used.
	Bitrate string
	// CRF (Constant Rate Factor) quality (0–51). Lower is better; 23 is a good default. Ignored if Bitrate is set.
	CRF int
	// GOP/keyframe interval, in frames. If 0, defaults to 2×FPS (low latency friendly).
	GOP int
	// x264 preset (ultrafast, superfast, veryfast, faster, fast, medium, slow...).
	// Default: "ultrafast" for real-time safety.
	Preset string
	// x264 tune (zerolatency, film, animation...). Default: "zerolatency".
	Tune string
	// x264 profile (baseline, main, high). Default: "baseline" for max compatibility.
	Profile string
	// Extra x264 params
	X264Params string
	// If true and the input buffer is full, new frames are dropped instead of blocking.
	DropIfBusy bool
	// Size of the internal frame channel buffer. Default: 128.
	InputBuffer int
	// If true, include timing PTS generation on input to stabilize pacing.
	GenPTS    bool
	InsertAUD bool
}

type Option func(*EncoderOptions)

func WithFPS(v int) Option {
	return func(o *EncoderOptions) {
		o.FPS = v
	}
}

func WithBitrate(v string) Option {
	return func(o *EncoderOptions) {
		o.Bitrate = v
	}
}

func WithCRF(v int) Option {
	return func(o *EncoderOptions) {
		o.CRF = v
	}
}

func WithGOP(v int) Option {
	return func(o *EncoderOptions) {
		o.GOP = v
	}
}

func WithPreset(v string) Option {
	return func(o *EncoderOptions) {
		o.Preset = v
	}
}

func WithTune(v string) Option {
	return func(o *EncoderOptions) {
		o.Tune = v
	}
}

func WithProfile(v string) Option {
	return func(o *EncoderOptions) {
		o.Profile = v
	}
}

func WithInputBuffer(n int) Option {
	return func(o *EncoderOptions) {
		o.InputBuffer = n
	}
}

func WithDropIfBusy(v bool) Option {
	return func(o *EncoderOptions) {
		o.DropIfBusy = v
	}
}

func DropIfBusy() Option {
	return func(o *EncoderOptions) {
		o.DropIfBusy = true
	}
}

func WithGenPTS(v bool) Option {
	return func(o *EncoderOptions) {
		o.GenPTS = v
	}
}

func GenPTS() Option {
	return func(o *EncoderOptions) {
		o.GenPTS = true
	}
}

func WithInsertAUD(v bool) Option {
	return func(o *EncoderOptions) {
		o.InsertAUD = v
	}
}

func InsertAUD() Option {
	return func(o *EncoderOptions) {
		o.InsertAUD = true
	}
}

func WithX264Params(raw string) Option {
	return func(o *EncoderOptions) {
		raw = strings.Trim(raw, " :,")
		if raw == "" {
			return
		}
		if o.X264Params == "" {
			o.X264Params = raw
		} else {
			o.X264Params += ":" + raw
		}
	}
}

func WithX264Param(k, v string) Option {
	return func(o *EncoderOptions) {
		kv := strings.TrimSpace(k)
		if kv == "" {
			return
		}
		if v != "" {
			kv += "=" + v
		}
		if o.X264Params == "" {
			o.X264Params = kv
		} else {
			o.X264Params += ":" + kv
		}
	}
}
func WithX264ParamsSlice(params ...string) Option {
	return func(o *EncoderOptions) {
		pp := make([]string, 0, len(params))
		for _, p := range params {
			p = strings.Trim(p, " :,")
			if p != "" {
				pp = append(pp, p)
			}
		}
		if len(pp) == 0 {
			return
		}
		sort.Strings(pp)
		add := strings.Join(pp, ":")
		if o.X264Params == "" {
			o.X264Params = add
		} else {
			o.X264Params += ":" + add
		}
	}
}

func NewEncoderOptions(opts ...Option) *EncoderOptions {
	o := &EncoderOptions{
		FPS:         30,
		CRF:         23,
		GOP:         60,
		Preset:      "ultrafast",
		Tune:        "zerolatency",
		Profile:     "baseline",
		InputBuffer: 128,
		InsertAUD:   true,
	}

	for _, opt := range opts {
		opt(o)
	}

	if o.GOP == 0 {
		if o.FPS > 0 {
			o.GOP = 2 * o.FPS
		} else {
			o.GOP = 60
		}
	}
	o.X264Params = strings.Trim(o.X264Params, " :,")
	if o.Bitrate != "" {
	} else if o.CRF == 0 {
		o.CRF = 23
	}

	o.Validate()
	return o
}

func (o *EncoderOptions) Validate() {
	if o.FPS <= 0 {
		o.FPS = 30
	}
	if o.CRF <= 0 || o.CRF > 51 {
		o.CRF = 30
	}
	if o.GOP <= 0 {
		o.GOP = 2 * o.FPS
	}
	if o.Preset == "" {
		o.Preset = "ultrafast"
	}
	if o.Tune == "" {
		o.Tune = "zerolatency"
	}
	if o.Profile == "" {
		o.Profile = "baseline"
	}
	if o.InputBuffer <= 0 {
		o.InputBuffer = 128
	}
}

// H264Encoder manages a single ffmpeg process.
// Use WriteFrame to push JPEG frames.
// Read the encoded H264 from Output.
type H264Encoder struct {
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	cmd       *exec.Cmd
	stdin     io.WriteCloser
	stdout    io.ReadCloser
	stderrBuf bytes.Buffer
	outReader io.Reader
	inCh      chan []byte
	errCh     chan error
	closed    chan struct{}
	opts      *EncoderOptions
}

// NewH264Encoder starts ffmpeg and returns the encoder plus an io.Reader for the H.264 output.
// Call Close() when done. ffmpeg must be available on PATH.
func NewH264Encoder(parent context.Context, opts *EncoderOptions) (*H264Encoder, io.Reader, error) {
	ctx, cancel := context.WithCancel(parent)
	if opts == nil {
		opts = NewEncoderOptions()
	}

	// Build ffmpeg args
	args := []string{
		"-hide_banner",
		"-loglevel",
		"error",
	}
	if opts.GenPTS {
		args = append(args, "-fflags", "+genpts")
	}

	// Input: MJPEG from stdin
	args = append(args,
		"-f", "image2pipe",
		"-r", fmt.Sprintf("%d", opts.FPS),
		"-i", "pipe:0",
	)

	// Video encoder settings
	args = append(
		args,
		"-an",
		"-c:v", "libx264",
		"-preset", opts.Preset,
		"-tune", opts.Tune,
		"-pix_fmt", "yuv420p",
		"-profile:v", opts.Profile,
		"-g", fmt.Sprintf("%d", opts.GOP),
		"-keyint_min", fmt.Sprintf("%d", opts.GOP),
		"-sc_threshold", "0",
		"-vf", "scale=640:-1",
	)

	if opts.InsertAUD {
		args = append(args, "-bsf:v", "h264_mp4toannexb")
	}

	if opts.X264Params != "" {
		args = append(args, "-x264-params", "sliced-threads=0", opts.X264Params)
	} else {
		args = append(args, "-x264-params", "sliced-threads=0")
	}

	if opts.Bitrate != "" {
		args = append(args, "-b:v", opts.Bitrate, "-maxrate", opts.Bitrate, "-bufsize", "2M")
	} else {
		args = append(args, "-crf", fmt.Sprintf("%d", opts.CRF))
	}

	// Output format to h264
	args = append(args, "-f", "h264", "pipe:1")

	cmd := exec.CommandContext(ctx, "ffmpeg", args...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		cancel()
		return nil, nil, fmt.Errorf("stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		return nil, nil, fmt.Errorf("stdout pipe: %w", err)
	}
	cmd.Stderr = &bytes.Buffer{}

	if err := cmd.Start(); err != nil {
		cancel()
		return nil, nil, fmt.Errorf("start ffmpeg: %w", err)
	}

	enc := &H264Encoder{
		cmd:       cmd,
		stdin:     stdin,
		stdout:    stdout,
		stderrBuf: bytes.Buffer{},
		cancel:    cancel,
		inCh:      make(chan []byte, opts.InputBuffer),
		errCh:     make(chan error, 1),
		closed:    make(chan struct{}),
		opts:      opts,
	}

	bufferSize := 1 << 20

	// Wrap output in a buffered reader with 1MB buffer.
	enc.outReader = bufio.NewReaderSize(enc.stdout, bufferSize)

	// Writer goroutine: serialize frames into ffmpeg stdin.
	enc.wg.Add(1)
	go func() {
		defer enc.wg.Done()
		bufWriter := bufio.NewWriterSize(enc.stdin, bufferSize)
		for frame := range enc.inCh {
			// Write each frame as is
			_, wErr := bufWriter.Write(frame)
			if wErr != nil {
				enc.errCh <- fmt.Errorf("failed to write frame: %w", wErr)
				break
			}
			fErr := bufWriter.Flush()
			if fErr != nil {
				enc.errCh <- fmt.Errorf("failed to flush bufWriter: %w", wErr)
				break
			}
		}
		// Closing on end of stream.
		_ = enc.stdin.Close()
	}()

	// Wait for process exit and capture stderr for diagnostics.
	enc.wg.Add(1)
	go func() {
		defer enc.wg.Done()
		waitErr := cmd.Wait()
		if cmd.Stderr != nil {
			if sb, ok := cmd.Stderr.(*bytes.Buffer); ok {
				enc.stderrBuf = *sb
				if enc.stderrBuf.String() != "" {
					enc.errCh <- fmt.Errorf(enc.stderrBuf.String())
				}
			}
		}
		if waitErr != nil && !errors.Is(waitErr, context.Canceled) && !strings.EqualFold(waitErr.Error(), "signal: killed") {
			enc.errCh <- fmt.Errorf("failed to exit ffmpeg: %w: %s", waitErr, enc.stderrBuf.String())
		}
		close(enc.closed)
		log.Default().Info("closed encoder frame channel")
	}()

	return enc, enc.outReader, nil
}

// WriteFrame push a single JPEG frame to the channel and can be reused.
func (e *H264Encoder) WriteFrame(frame []byte) error {
	select {
	case <-e.closed:
		return io.EOF
	default:
	}

	if e.opts.DropIfBusy {
		select {
		case e.inCh <- frame:
			return nil
		default:
			return nil
		}
	}
	select {
	case e.inCh <- frame:
		return nil
	case <-e.closed:
		return io.EOF
	}
}

// Close stops the encoder. It drains and closes stdin, waits for ffmpeg to exit.
func (e *H264Encoder) Close() error {
	// Stop accepting frames.
	select {
	case <-e.closed:
	default:
		close(e.inCh)
	}

	// Wait ffmpeg to flush
	done := make(chan struct{})
	go func() {
		defer close(done)
		e.wg.Wait()
	}()

	select {
	case <-done:
	case <-time.After(1 * time.Second):
		e.cancel()
		<-done
	}

	e.cancel()

	select {
	case err := <-e.errCh:
		return err
	default:
		log.Default().Info("successfully closed h264 encoder")
		return nil
	}
}

// Output returns an io.Reader for the encoded stream.
func (e *H264Encoder) Output() io.Reader {
	return e.outReader
}
