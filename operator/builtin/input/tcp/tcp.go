package tcp

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"sync"

	"github.com/observiq/stanza/operator"
	"github.com/observiq/stanza/operator/helper"
	"go.uber.org/zap"
)

func init() {
	operator.Register("tcp_input", func() operator.Builder { return NewInputConfig("") })
}

// NewInputConfig creates a new TCP input config with default values
func NewInputConfig(operatorID string) *InputConfig {
	return &InputConfig{
		InputConfig:   helper.NewInputConfig(operatorID, "tcp_input"),
		DecoderConfig: helper.NewDecoderConfig("nop"),
	}
}

// InputConfig is the configuration of a tcp input operator.
type InputConfig struct {
	helper.InputConfig   `yaml:",inline"`
	helper.DecoderConfig `yaml:",inline,omitempty"`

	ListenAddress string `json:"listen_address,omitempty" yaml:"listen_address,omitempty"`
}

// Build will build a tcp input operator.
func (c InputConfig) Build(context operator.BuildContext) ([]operator.Operator, error) {
	inputOperator, err := c.InputConfig.Build(context)
	if err != nil {
		return nil, err
	}

	decoder, err := c.DecoderConfig.Build()
	if err != nil {
		return nil, err
	}

	if c.ListenAddress == "" {
		return nil, fmt.Errorf("missing required parameter 'listen_address'")
	}

	address, err := net.ResolveTCPAddr("tcp", c.ListenAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve listen_address: %s", err)
	}

	tcpInput := &InputOperator{
		InputOperator: inputOperator,
		decoder:       decoder,
		address:       address,
	}
	return []operator.Operator{tcpInput}, nil
}

// InputOperator is an operator that listens for log entries over tcp.
type InputOperator struct {
	helper.InputOperator
	decoder *helper.Decoder
	address *net.TCPAddr

	listener *net.TCPListener
	cancel   context.CancelFunc
	wg       sync.WaitGroup
}

// Start will start listening for log entries over tcp.
func (t *InputOperator) Start() error {
	listener, err := net.ListenTCP("tcp", t.address)
	if err != nil {
		return fmt.Errorf("failed to listen on interface: %w", err)
	}

	t.listener = listener
	ctx, cancel := context.WithCancel(context.Background())
	t.cancel = cancel
	t.goListen(ctx)
	return nil
}

// goListenn will listen for tcp connections.
func (t *InputOperator) goListen(ctx context.Context) {
	t.wg.Add(1)

	go func() {
		defer t.wg.Done()

		for {
			conn, err := t.listener.AcceptTCP()
			if err != nil {
				select {
				case <-ctx.Done():
					return
				default:
					t.Debugw("Listener accept error", zap.Error(err))
				}
			}

			t.Debugf("Received connection: %s", conn.RemoteAddr().String())
			subctx, cancel := context.WithCancel(ctx)
			t.goHandleClose(subctx, conn)
			t.goHandleMessages(subctx, conn, cancel)
		}
	}()
}

// goHandleClose will wait for the context to finish before closing a connection.
func (t *InputOperator) goHandleClose(ctx context.Context, conn net.Conn) {
	t.wg.Add(1)

	go func() {
		defer t.wg.Done()
		<-ctx.Done()
		t.Debugf("Closing connection: %s", conn.RemoteAddr().String())
		if err := conn.Close(); err != nil {
			t.Errorf("Failed to close connection: %s", err)
		}
	}()
}

// goHandleMessages will handles messages from a tcp connection.
func (t *InputOperator) goHandleMessages(ctx context.Context, conn net.Conn, cancel context.CancelFunc) {
	t.wg.Add(1)

	go func() {
		defer t.wg.Done()
		defer cancel()

		decoder := t.decoder.Copy()

		scanner := bufio.NewScanner(conn)
		for scanner.Scan() {

			msg := scanner.Bytes()
			t.Debug("Received raw bytes: %s", msg)

			decodedMsg, err := decoder.Decode(msg)
			if err != nil {
				t.Errorw("Failed to decode message", zap.Error(err))
				continue
			}

			entry, err := t.NewEntry(decodedMsg)
			if err != nil {
				t.Errorw("Failed to create entry", zap.Error(err))
				continue
			}
			t.Write(ctx, entry)
		}
		if err := scanner.Err(); err != nil {
			t.Errorw("Scanner error", zap.Error(err))
		}
	}()
}

// Stop will stop listening for log entries over TCP.
func (t *InputOperator) Stop() error {
	t.cancel()

	if err := t.listener.Close(); err != nil {
		return err
	}

	t.wg.Wait()
	return nil
}
