package scanner

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/schollz/progressbar/v3"
	"vmap/internal/logx"
	"vmap/internal/rangeparse"
	"vmap/internal/vsock"
)

const (
	modeAuto  = "auto"
	modeHost  = "host"
	modeGuest = "guest"

	cidAllMin  uint32 = 0
	cidAllMax  uint32 = 65535
	portAllMin uint32 = 1
	portAllMax uint32 = 65535
)

var detectPayloads = [][]byte{
	{},
	[]byte("{}"),
	[]byte("\n"),
	[]byte("1"),
	[]byte("a"),
}

type Options struct {
	Mode      string
	CIDInput  string
	PortInput string
	Payloads  []string
	Detect    bool
	Timeout   time.Duration
	Interval  time.Duration
	LogLevel  string
}

type Scanner struct {
	opts   Options
	cids   []uint32
	ports  []uint32
	logger *logx.Logger
}

type scanTarget struct {
	cid  uint32
	port uint32
}

func New(opts Options) (*Scanner, error) {
	if strings.TrimSpace(opts.LogLevel) == "" {
		opts.LogLevel = "info"
	}
	logger, err := logx.NewFromString(opts.LogLevel, os.Stdout)
	if err != nil {
		return nil, err
	}

	requestedMode := opts.Mode
	resolvedMode, localCID, modeErr := resolveMode(opts.Mode)
	if modeErr != nil {
		return nil, modeErr
	}

	if requestedMode == modeAuto && (resolvedMode == modeHost || resolvedMode == modeGuest) {
		opts.Mode = resolvedMode
		logger.Infof("auto mode detected runtime=%s local-cid=%d; overriding mode", resolvedMode, localCID)
	}

	cids, err := resolveCIDTargets(opts.CIDInput, resolvedMode, localCID)
	if err != nil {
		return nil, err
	}

	ports, err := resolvePortTargets(opts.PortInput)
	if err != nil {
		return nil, err
	}

	if len(cids) == 0 {
		return nil, errors.New("no CID targets resolved")
	}
	if len(ports) == 0 {
		return nil, errors.New("no port targets resolved")
	}

	if opts.Interval < 0 {
		return nil, errors.New("--interval must be >= 0")
	}
	if opts.Timeout < 0 {
		return nil, errors.New("--timeout must be >= 0")
	}

	if requestedMode == modeAuto && opts.CIDInput == "" && resolvedMode == modeAuto {
		logger.Warnf("local CID unavailable in auto mode; CID target falls back to all")
	}

	if requestedMode == modeGuest && opts.CIDInput == "" && localCID == 0 {
		logger.Warnf("local CID unavailable in guest mode; CID target falls back to all")
	}

	return &Scanner{opts: opts, cids: cids, ports: ports, logger: logger}, nil
}

func (s *Scanner) Run(ctx context.Context) error {
	s.logger.Infof("mode=%s cids=%d ports=%d detect=%v timeout=%s interval=%s log-level=%s",
		s.opts.Mode,
		len(s.cids),
		len(s.ports),
		s.opts.Detect,
		durationLabel(s.opts.Timeout),
		durationLabel(s.opts.Interval),
		s.opts.LogLevel,
	)

	var payloads [][]byte
	if s.opts.Detect {
		payloads, err = buildDetectPayloads(s.opts.Payloads)
		if err != nil {
			return err
		}
		s.logger.Infof("detect payloads configured=%d custom=%v", len(payloads), len(s.opts.Payloads) > 0)
	}

	openTargets, err := s.scanConnectivity(ctx)
	if err != nil {
		return err
	}
	s.printConnectivitySummary(openTargets)

	if !s.opts.Detect {
		return nil
	}

	if len(openTargets) == 0 {
		s.logger.Infof("detect phase skipped because no open targets")
		return nil
	}

	s.logger.Infof("detect phase starting, open-targets=%d", len(openTargets))
	if err := s.runDetectPhase(ctx, openTargets, payloads); err != nil {
		return err
	}

	return nil
}

func (s *Scanner) scanConnectivity(ctx context.Context) ([]scanTarget, error) {
	totalTargets := int64(len(s.cids)) * int64(len(s.ports))
	bar := newProgressBar(totalTargets, "connectivity")
	defer finishProgressBar(bar)

	openTargets := make([]scanTarget, 0)
	first := true
	for _, cid := range s.cids {
		for _, port := range s.ports {
			if err := ctx.Err(); err != nil {
				return nil, err
			}

			if !first && s.opts.Interval > 0 {
				time.Sleep(s.opts.Interval)
			}
			first = false

			open, err := s.probeOpen(cid, port)
			if err != nil {
				s.logger.Debugf("connectivity probe failed target=%d:%d err=%v", cid, port, err)
			} else if open {
				openTargets = append(openTargets, scanTarget{cid: cid, port: port})
			}

			if bar != nil {
				_ = bar.Add(1)
			}
		}
	}

	return openTargets, nil
}

func (s *Scanner) probeOpen(cid uint32, port uint32) (bool, error) {
	s.logger.Debugf("dial connectivity target=%d:%d", cid, port)
	conn, err := vsock.Dial(cid, port, s.opts.Timeout)
	if err != nil {
		return false, err
	}
	_ = conn.Close()

	s.logger.Infof("[open] %d:%d", cid, port)
	return true, nil
}

func (s *Scanner) printConnectivitySummary(targets []scanTarget) {
	s.logger.Infof("[summary] connectivity scan completed, open-targets=%d", len(targets))
	if len(targets) == 0 {
		return
	}
	s.logger.Infof("[summary] open targets (replay):")
	for _, target := range targets {
		s.logger.Infof("[open] %d:%d", target.cid, target.port)
	}
}

func (s *Scanner) runDetectPhase(ctx context.Context, targets []scanTarget, payloads [][]byte) error {
	totalDetectProbes := int64(len(targets)) * int64(len(payloads))
	bar := newProgressBar(totalDetectProbes, "detect")
	defer finishProgressBar(bar)

	for _, target := range targets {
		for _, payload := range payloads {
			if err := ctx.Err(); err != nil {
				return err
			}

			response, err := s.detectOnce(target.cid, target.port, payload)
			if err != nil {
				s.logger.Debugf("detect probe failed target=%d:%d payload=%q err=%v", target.cid, target.port, payload, err)
			} else if len(response) > 0 {
				s.logger.Infof("[detect] %d:%d payload=%q response=%q", target.cid, target.port, payload, response)
			} else {
				s.logger.Debugf("detect probe empty response target=%d:%d payload=%q", target.cid, target.port, payload)
			}

			if bar != nil {
				_ = bar.Add(1)
			}
		}
	}

	return nil
}

func (s *Scanner) detectOnce(cid uint32, port uint32, payload []byte) ([]byte, error) {
	s.logger.Debugf("dial detect target=%d:%d", cid, port)
	conn, err := vsock.Dial(cid, port, s.opts.Timeout)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	s.logger.Debugf("send payload target=%d:%d payload=%q", cid, port, payload)
	if len(payload) > 0 {
		written, err := conn.Write(payload)
		if err != nil {
			return nil, err
		}
		s.logger.Debugf("send payload done target=%d:%d bytes=%d", cid, port, written)
	} else {
		s.logger.Debugf("send payload skipped write for empty payload target=%d:%d", cid, port)
	}

	readTimeout := s.opts.Timeout
	if readTimeout <= 0 {
		readTimeout = 250 * time.Millisecond
	}
	if err := conn.SetReadDeadline(time.Now().Add(readTimeout)); err != nil {
		return nil, err
	}

	buf := make([]byte, 4096)
	n, err := conn.Read(buf)
	if err != nil {
		if isTimeout(err) {
			s.logger.Debugf("receive timeout target=%d:%d", cid, port)
			return nil, nil
		}
		return nil, err
	}
	if n <= 0 {
		s.logger.Debugf("receive empty payload target=%d:%d", cid, port)
		return nil, nil
	}

	response := append([]byte{}, buf[:n]...)
	s.logger.Debugf("receive response target=%d:%d bytes=%d response=%q", cid, port, n, response)
	return response, nil
}

func buildDetectPayloads(inputs []string) ([][]byte, error) {
	if len(inputs) == 0 {
		payloads := append([][]byte{}, detectPayloads...)
		payloads = append(payloads, []byte(newUUID()))
		return payloads, nil
	}

	payloads := make([][]byte, 0, len(inputs))
	for _, input := range inputs {
		payload, err := decodePayload(input)
		if err != nil {
			return nil, err
		}
		payloads = append(payloads, payload)
	}
	return payloads, nil
}

func decodePayload(input string) ([]byte, error) {
	normalized := strings.ToLower(strings.TrimSpace(input))
	if normalized == "empty" || normalized == "<empty>" {
		return []byte{}, nil
	}

	escaped := strings.ReplaceAll(input, "\"", "\\\"")
	decoded, err := strconv.Unquote("\"" + escaped + "\"")
	if err != nil {
		return nil, fmt.Errorf("invalid payload %q: %w", input, err)
	}
	return []byte(decoded), nil
}

func resolveMode(input string) (string, uint32, error) {
	mode := strings.ToLower(strings.TrimSpace(input))
	switch mode {
	case modeHost:
		return modeHost, 0, nil
	case modeGuest:
		cid, err := vsock.LocalCID()
		if err != nil {
			return modeGuest, 0, nil
		}
		return modeGuest, cid, nil
	case modeAuto:
		cid, err := vsock.LocalCID()
		if err != nil {
			return modeAuto, 0, nil
		}
		if cid == vsock.HostCID {
			return modeHost, cid, nil
		}
		return modeGuest, cid, nil
	default:
		return "", 0, fmt.Errorf("invalid mode %q", input)
	}
}

func resolveCIDTargets(input string, mode string, localCID uint32) ([]uint32, error) {
	parsed, all, err := rangeparse.ParseUint32List(input, cidAllMin, cidAllMax)
	if err != nil {
		return nil, fmt.Errorf("parse --cid: %w", err)
	}
	if input != "" {
		if all {
			return expandRange(cidAllMin, cidAllMax), nil
		}
		return parsed, nil
	}

	switch mode {
	case modeHost:
		return expandRange(cidAllMin, cidAllMax), nil
	case modeGuest:
		if localCID == 0 {
			return expandRange(cidAllMin, cidAllMax), nil
		}
		if localCID == vsock.HostCID {
			return []uint32{vsock.HostCID}, nil
		}
		return []uint32{vsock.HostCID, localCID}, nil
	case modeAuto:
		return expandRange(cidAllMin, cidAllMax), nil
	default:
		return nil, fmt.Errorf("cannot resolve CID targets for mode=%s", mode)
	}
}

func resolvePortTargets(input string) ([]uint32, error) {
	if strings.TrimSpace(input) == "" {
		input = "all"
	}
	parsed, all, err := rangeparse.ParseUint32List(input, portAllMin, portAllMax)
	if err != nil {
		return nil, fmt.Errorf("parse --port: %w", err)
	}
	if all {
		return expandRange(portAllMin, portAllMax), nil
	}
	return parsed, nil
}

func expandRange(min uint32, max uint32) []uint32 {
	values := make([]uint32, 0, int(max-min+1))
	for i := min; i <= max; i++ {
		values = append(values, i)
		if i == max {
			break
		}
	}
	return values
}

func durationLabel(d time.Duration) string {
	if d <= 0 {
		return "disabled"
	}
	return d.String()
}

func isTimeout(err error) bool {
	type timeout interface {
		Timeout() bool
	}
	if te, ok := err.(timeout); ok {
		return te.Timeout()
	}
	return false
}

func newProgressBar(total int64, description string) *progressbar.ProgressBar {
	if total <= 0 {
		return nil
	}

	return progressbar.NewOptions64(
		total,
		progressbar.OptionSetDescription(description),
		progressbar.OptionSetWriter(os.Stderr),
		progressbar.OptionSetWidth(20),
		progressbar.OptionShowCount(),
		progressbar.OptionShowIts(),
		progressbar.OptionSetRenderBlankState(true),
		progressbar.OptionThrottle(100*time.Millisecond),
	)
}

func finishProgressBar(bar *progressbar.ProgressBar) {
	if bar == nil {
		return
	}
	_ = bar.Finish()
	fmt.Fprintln(os.Stderr)
}
