package ffmpeg

import (
	"bufio"
	"errors"
	"fmt"
	"go.uber.org/zap"
	"io"
	"net"
	"strconv"
	"strings"
)

type ProgressListener struct {
	listener *net.TCPListener
}

type ProgressReport struct {
	Bitrate   string
	FPS       float64
	Frame     int
	OutTime   string
	Speed     string
	TotalSize int
}

func (p *ProgressListener) Begin() (string, error) {
	if p.listener == nil {
		addr, err := net.ResolveTCPAddr("tcp", "127.0.0.1:0")
		if err != nil {
			return "", err
		}

		listener, err := net.ListenTCP(addr.Network(), addr)
		if err != nil {
			return "", err
		}

		p.listener = listener
	}
	return fmt.Sprintf("tcp://%s", p.listener.Addr().String()), nil
}

func (p *ProgressListener) Close() error {
	if p.listener != nil {
		err := p.listener.Close()
		p.listener = nil
		return err
	}
	return nil
}

func (p *ProgressListener) Run(logger *zap.SugaredLogger) {
	conn, err := p.listener.Accept()
	if err != nil {
		logger.Errorw("error while accepting ffmpeg connection", "err", err)
		return
	}
	defer conn.Close()

	reader := bufio.NewReader(conn)
	report := &ProgressReport{}
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if errors.Is(err, io.EOF) {
				return
			}
			logger.Errorw("error while reading ffmpeg progress", "err", err)
			return
		}

		kv := strings.Split(line, "=")
		value := strings.TrimSpace(kv[1])
		switch kv[0] {
		case "bitrate":
			report.Bitrate = value
		case "frame":
			report.Frame, _ = strconv.Atoi(value)
		case "fps":
			report.FPS, _ = strconv.ParseFloat(value, 32)
		case "out_time":
			report.OutTime = value
		case "speed":
			report.Speed = value
		case "total_size":
			report.TotalSize, _ = strconv.Atoi(value)
		case "progress":
			logger.Infow("ffmpeg progress",
				"bitrate", report.Bitrate,
				"frame", report.Frame,
				"fps", fmt.Sprintf("%.02f", report.FPS),
				"out_time", report.OutTime,
				"speed", report.Speed,
				"total_size", report.TotalSize,
			)
			if value == "end" {
				return
			}
		default:
		}
	}
}
