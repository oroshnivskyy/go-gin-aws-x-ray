package xray

import (
	"context"
	"encoding/json"
	"github.com/aws/aws-xray-sdk-go/xray"
	"net"
	"time"
)

func NewXRayTestDaemon() (*testDaemon, error) {
	d := &testDaemon{
		Channel: make(chan *result, 200),
	}
	listenerAddr := &net.UDPAddr{
		IP:   net.IPv4(127, 0, 0, 1),
		Port: 2010,
	}
	err := xray.Configure(xray.Config{DaemonAddr: "localhost:2010"})
	if err != nil {
		return nil, err
	}
	if d.Connection == nil {
		conn, err := net.ListenUDP("udp", listenerAddr)
		if err != nil {
			return nil, err
		}

		d.Connection = conn
	}
	return d, nil
}

type testDaemon struct {
	Connection *net.UDPConn
	Channel    chan *result
	Done       bool
}
type result struct {
	Segment *xray.Segment
	Error   error
}

func (td *testDaemon) Run() {
	buffer := make([]byte, 64000)
	// Don't do that in production code
	for !td.Done {
		n, _, err := td.Connection.ReadFromUDP(buffer)
		if err != nil {
			td.Channel <- &result{nil, err}
			continue
		}

		buffered := buffer[len(xray.Header):n]

		seg := &xray.Segment{}
		err = json.Unmarshal(buffered, &seg)
		if err != nil {
			td.Channel <- &result{nil, err}
			continue
		}

		seg.Sampled = true
		td.Channel <- &result{seg, err}
	}
}

func (td *testDaemon) Recv() (*xray.Segment, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	select {
	case r := <-td.Channel:
		return r.Segment, r.Error
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (td *testDaemon) Stop() {
	td.Done = true
}
