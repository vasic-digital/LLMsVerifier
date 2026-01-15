package rabbitmq

import (
	"context"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/sirupsen/logrus"
)

// ConnectionState represents the state of the connection.
type ConnectionState int

const (
	StateDisconnected ConnectionState = iota
	StateConnecting
	StateConnected
	StateReconnecting
	StateClosed
)

// Connection manages an AMQP connection with automatic reconnection.
type Connection struct {
	mu sync.RWMutex

	config     *Config
	logger     *logrus.Logger
	conn       *amqp.Connection
	state      ConnectionState
	notifyConn chan *amqp.Error

	// Reconnection state
	reconnectAttempts int
	lastConnectTime   time.Time

	// Close signaling
	closeCh chan struct{}
	done    chan struct{}

	// Callbacks
	onConnect    func()
	onDisconnect func(error)
	onReconnect  func(int)
}

// NewConnection creates a new Connection.
func NewConnection(config *Config, logger *logrus.Logger) *Connection {
	if config == nil {
		config = DefaultConfig()
	}
	if logger == nil {
		logger = logrus.New()
	}

	return &Connection{
		config:  config,
		logger:  logger,
		state:   StateDisconnected,
		closeCh: make(chan struct{}),
		done:    make(chan struct{}),
	}
}

// Connect establishes the AMQP connection.
func (c *Connection) Connect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.state == StateClosed {
		return ErrConnectionClosed
	}
	if c.state == StateConnected {
		return nil
	}

	c.state = StateConnecting
	return c.connect(ctx)
}

// connect performs the actual connection (must be called with lock held).
func (c *Connection) connect(ctx context.Context) error {
	cfg := amqp.Config{
		Heartbeat: c.config.HeartbeatInterval,
		Locale:    "en_US",
	}

	if c.config.TLS != nil {
		cfg.TLSClientConfig = c.config.TLS
	}

	if c.config.ChannelMax > 0 {
		cfg.ChannelMax = uint16(c.config.ChannelMax)
	}
	if c.config.FrameSize > 0 {
		cfg.FrameSize = c.config.FrameSize
	}

	// Connect with timeout
	var conn *amqp.Connection
	var err error

	done := make(chan struct{})
	go func() {
		conn, err = amqp.DialConfig(c.config.AMQPURI(), cfg)
		close(done)
	}()

	select {
	case <-done:
		if err != nil {
			c.state = StateDisconnected
			c.logger.WithError(err).Error("Failed to connect to RabbitMQ")
			return err
		}
	case <-time.After(c.config.ConnectionTimeout):
		c.state = StateDisconnected
		return ErrConnectionTimeout
	case <-ctx.Done():
		c.state = StateDisconnected
		return ctx.Err()
	}

	c.conn = conn
	c.state = StateConnected
	c.lastConnectTime = time.Now()
	c.reconnectAttempts = 0

	// Setup disconnect notification
	c.notifyConn = make(chan *amqp.Error, 1)
	c.conn.NotifyClose(c.notifyConn)

	// Start reconnection monitor
	go c.monitorConnection()

	c.logger.Info("Connected to RabbitMQ")

	if c.onConnect != nil {
		go c.onConnect()
	}

	return nil
}

// monitorConnection monitors the connection and triggers reconnection.
func (c *Connection) monitorConnection() {
	for {
		select {
		case <-c.closeCh:
			return
		case err, ok := <-c.notifyConn:
			if !ok {
				return
			}

			c.mu.Lock()
			if c.state == StateClosed {
				c.mu.Unlock()
				return
			}
			c.state = StateReconnecting
			c.mu.Unlock()

			c.logger.WithError(err).Warn("RabbitMQ connection lost, reconnecting...")

			if c.onDisconnect != nil {
				c.onDisconnect(err)
			}

			c.reconnect()
		}
	}
}

// reconnect attempts to reconnect with exponential backoff.
func (c *Connection) reconnect() {
	backoff := c.config.ReconnectInterval

	for {
		select {
		case <-c.closeCh:
			return
		default:
		}

		c.mu.Lock()
		c.reconnectAttempts++
		attempt := c.reconnectAttempts

		if c.config.MaxReconnectAttempts > 0 && attempt > c.config.MaxReconnectAttempts {
			c.state = StateClosed
			c.mu.Unlock()
			c.logger.Error("Max reconnection attempts reached, giving up")
			return
		}
		c.mu.Unlock()

		c.logger.WithField("attempt", attempt).Info("Attempting to reconnect...")

		if c.onReconnect != nil {
			c.onReconnect(attempt)
		}

		ctx, cancel := context.WithTimeout(context.Background(), c.config.ConnectionTimeout)
		c.mu.Lock()
		err := c.connect(ctx)
		c.mu.Unlock()
		cancel()

		if err == nil {
			c.logger.Info("Successfully reconnected to RabbitMQ")
			return
		}

		c.logger.WithError(err).WithField("backoff", backoff).Warn("Reconnection failed, retrying...")

		select {
		case <-c.closeCh:
			return
		case <-time.After(backoff):
		}

		// Exponential backoff
		backoff = time.Duration(float64(backoff) * 1.5)
		if backoff > c.config.MaxReconnectInterval {
			backoff = c.config.MaxReconnectInterval
		}
	}
}

// Close closes the connection.
func (c *Connection) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.state == StateClosed {
		return nil
	}

	c.state = StateClosed
	close(c.closeCh)

	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// Channel returns a new AMQP channel.
func (c *Connection) Channel() (*amqp.Channel, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.state != StateConnected || c.conn == nil {
		return nil, ErrNotConnected
	}

	return c.conn.Channel()
}

// IsConnected returns true if the connection is established.
func (c *Connection) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.state == StateConnected && c.conn != nil
}

// State returns the current connection state.
func (c *Connection) State() ConnectionState {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.state
}

// OnConnect sets the callback for successful connection.
func (c *Connection) OnConnect(fn func()) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.onConnect = fn
}

// OnDisconnect sets the callback for disconnection.
func (c *Connection) OnDisconnect(fn func(error)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.onDisconnect = fn
}

// OnReconnect sets the callback for reconnection attempts.
func (c *Connection) OnReconnect(fn func(int)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.onReconnect = fn
}

// Connection errors.
var (
	ErrNotConnected      = ConnectionError("not connected")
	ErrConnectionClosed  = ConnectionError("connection closed")
	ErrConnectionTimeout = ConnectionError("connection timeout")
)

// ConnectionError represents a connection error.
type ConnectionError string

func (e ConnectionError) Error() string {
	return string(e)
}
