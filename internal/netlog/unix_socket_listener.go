package netlog

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/2beens/serjtubincom/internal/telemetry/metrics"
	"github.com/2beens/serjtubincom/pkg"

	log "github.com/sirupsen/logrus"
)

// VisitsBackupUnixSocketListenerSetup - this is a deliberately overengineered method of communicating of netlog backup with the main service
// just so I have a piece of code that uses UNIX socket interprocess communication, and also to avoid
// adding the Prometheus push gateway to push metrics to it
func VisitsBackupUnixSocketListenerSetup(
	ctx context.Context,
	socketAddrDir, socketFileName string,
	instr *metrics.Manager,
) (net.Addr, error) {
	socket := filepath.Join(socketAddrDir, socketFileName)
	listener, err := net.Listen("unix", socket)
	if err != nil {
		return nil, fmt.Errorf("binding to unix socket %s: %w", socket, err)
	}

	if err := os.Chmod(socket, os.ModeSocket|0666); err != nil {
		return nil, err
	}

	go func() {
		go func() {
			<-ctx.Done()
			log.Debugln("netlog backup unix socket listener context done, closing listener")
			_ = listener.Close()
		}()

		for {
			select {
			case <-ctx.Done():
				return
			default:
				// Otherwise, continue accepting new connections.
			}

			conn, err := listener.Accept()
			if err != nil {
				log.Errorf("netlog backup unix socket listener conn accept: %s", err)
				return
			}
			log.Debugf("netlog backup unix socket got new conn: %s", conn.RemoteAddr().String())

			// if it takes over 5 minutes to transfer all netlog data, then something is probably not right
			if err := conn.SetDeadline(time.Now().Add(5 * time.Minute)); err != nil {
				log.Errorf("failed to set conn timeout: %s", err)
				return
			}

			go func() {
				defer func() { _ = conn.Close() }()

				buf := make([]byte, 1024)
				n, err := conn.Read(buf)
				if err != nil {
					return
				}

				messageReceived := pkg.BytesToString(buf[:n])
				log.Infof("netlog backup unix socket received: %s", messageReceived)

				msgParts := strings.Split(messageReceived, "||")
				if len(msgParts) != 2 {
					log.Errorf("netlog backup conn, invalid message received: %s", messageReceived)
					return
				}

				durationInfo := msgParts[1]
				sendNetlogBackupDurationInfo(durationInfo, instr)

				visitsCountInfo := msgParts[0]
				sendNetlogBackupVisitsCount(visitsCountInfo, instr)

				_, err = conn.Write([]byte("ok"))
				if err != nil {
					log.Errorf("netlog backup conn, send response: %s", err)
				}
			}()
		}
	}()

	return listener.Addr(), nil
}

func sendNetlogBackupDurationInfo(durationInfoMsg string, metrics *metrics.Manager) {
	durationInfoParts := strings.Split(durationInfoMsg, "::")
	if len(durationInfoParts) != 2 {
		log.Errorf("netlog backup conn, invalid duration info received: %s", durationInfoMsg)
		return
	}

	durationInSec, err := strconv.ParseFloat(durationInfoParts[1], 64)
	if err != nil {
		log.Errorf("netlog backup conn, invalid duration info received: %s", err)
		return
	}

	metrics.HistNetlogBackupDuration.Observe(durationInSec)
}

func sendNetlogBackupVisitsCount(visitsCountInfoMsg string, metrics *metrics.Manager) {
	visitsCountInfoParts := strings.Split(visitsCountInfoMsg, "::")
	if len(visitsCountInfoParts) != 2 {
		log.Errorf("netlog backup conn, invalid visits info received: %s", visitsCountInfoMsg)
		return
	}

	visitsCount, err := strconv.Atoi(visitsCountInfoParts[1])
	if err != nil {
		log.Errorf("netlog backup conn, invalid visits counter: %s", err)
		return
	}

	metrics.CounterVisitsBackups.Add(float64(visitsCount))
}
