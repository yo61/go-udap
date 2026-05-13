package udap

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

// MultiTransport composes a set of child Transports. Send fans out to
// all of them; Recv merges replies through per-child goroutines into a
// shared channel. The Client doesn't care whether its transport is a
// single UDPTransport or a MultiTransport — both satisfy the Transport
// port.
//
// Used by --all-interfaces: one UDPTransport per usable interface,
// composed into a MultiTransport.
type MultiTransport struct {
	children []Transport
	logger   Logger

	startOnce sync.Once
	merged    chan multiTransportRecv
	stop      chan struct{}
	wg        sync.WaitGroup

	closeOnce sync.Once
	closeErr  error
}

type multiTransportRecv struct {
	pkt []byte
	src string
	err error
}

// NewMultiTransport constructs a MultiTransport composing the given
// children. The slice must contain at least one transport (callers
// should reject empty fan-out upstream so the error message can mention
// "no usable interfaces").
func NewMultiTransport(children []Transport, logger Logger) *MultiTransport {
	return &MultiTransport{
		children: children,
		logger:   logger,
		merged:   make(chan multiTransportRecv, 32),
		stop:     make(chan struct{}),
	}
}

// Send broadcasts the packet on every child. Returns success if at
// least one child succeeded; aggregated error only if every child
// failed. Per-child errors are logged at Warn.
func (m *MultiTransport) Send(packet []byte) error {
	var failures []string
	successes := 0
	for i, c := range m.children {
		if err := c.Send(packet); err != nil {
			m.logger.Warn("MultiTransport child send failed", "child", i, "error", err)
			failures = append(failures, err.Error())
			continue
		}
		successes++
	}
	if successes == 0 {
		return fmt.Errorf("all children failed: %v", failures)
	}
	return nil
}

// Recv returns the next packet from any child, or the context error if
// ctx is cancelled. Lazily starts one goroutine per child on first
// call.
func (m *MultiTransport) Recv(ctx context.Context) ([]byte, string, error) {
	m.startOnce.Do(m.spawnPumps)
	select {
	case <-ctx.Done():
		return nil, "", ctx.Err()
	case ro, ok := <-m.merged:
		if !ok {
			return nil, "", errors.New("multi transport closed")
		}
		if ro.err != nil {
			return nil, "", ro.err
		}
		return ro.pkt, ro.src, nil
	}
}

// spawnPumps starts one goroutine per child that forwards packets to
// m.merged until the stop channel closes.
func (m *MultiTransport) spawnPumps() {
	for i, c := range m.children {
		m.wg.Add(1)
		go m.pumpChild(i, c)
	}
}

func (m *MultiTransport) pumpChild(idx int, c Transport) {
	defer m.wg.Done()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	// One watcher goroutine per pump that translates stop → ctx cancel.
	go func() {
		<-m.stop
		cancel()
	}()
	for {
		pkt, src, err := c.Recv(ctx)
		if err != nil {
			// On error (including ctx cancel from Close), pump exits.
			// Don't forward errors except as a debug log.
			select {
			case <-m.stop:
				// normal shutdown
			default:
				m.logger.Warn("MultiTransport child recv error",
					"child", idx, "error", err)
			}
			return
		}
		select {
		case m.merged <- multiTransportRecv{pkt: pkt, src: src}:
		case <-m.stop:
			return
		}
	}
}

// Close closes all children and signals the pump goroutines to exit.
// Returns the first non-nil child Close error.
func (m *MultiTransport) Close() error {
	m.closeOnce.Do(func() {
		close(m.stop)
		for _, c := range m.children {
			if err := c.Close(); err != nil && m.closeErr == nil {
				m.closeErr = err
			}
		}
		m.wg.Wait()
	})
	return m.closeErr
}
