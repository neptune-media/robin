package ffmpeg

import (
	"bufio"
	"errors"
	"fmt"
	"go.uber.org/zap"
	"io"
	"net"
	"strings"
)

type ProgressListener struct {
	listener *net.TCPListener
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
		switch kv[0] {
		case "out_time":
			logger.Infow("ffmpeg progress", "out-time", kv[1])
		case "progress":
			if strings.Trim(kv[1], "\n") == "end" {
				return
			}
		default:
		}
	}
}
