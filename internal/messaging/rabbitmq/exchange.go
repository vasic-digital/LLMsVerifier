package rabbitmq

import (
	"context"

	amqp "github.com/rabbitmq/amqp091-go"
)

// ExchangeType represents the type of AMQP exchange.
type ExchangeType string

const (
	ExchangeDirect  ExchangeType = "direct"
	ExchangeFanout  ExchangeType = "fanout"
	ExchangeTopic   ExchangeType = "topic"
	ExchangeHeaders ExchangeType = "headers"
)

// ExchangeConfig contains configuration for declaring an exchange.
type ExchangeConfig struct {
	Name       string       `json:"name" yaml:"name"`
	Type       ExchangeType `json:"type" yaml:"type"`
	Durable    bool         `json:"durable" yaml:"durable"`
	AutoDelete bool         `json:"auto_delete" yaml:"auto_delete"`
	Internal   bool         `json:"internal" yaml:"internal"`
	NoWait     bool         `json:"no_wait" yaml:"no_wait"`
	Arguments  amqp.Table   `json:"arguments" yaml:"arguments"`
}

// DefaultExchangeConfig returns default exchange configuration.
func DefaultExchangeConfig(name string, exchangeType ExchangeType) *ExchangeConfig {
	return &ExchangeConfig{
		Name:       name,
		Type:       exchangeType,
		Durable:    true,
		AutoDelete: false,
		Internal:   false,
		NoWait:     false,
	}
}

// DeclareExchange declares an exchange on the broker.
func (b *Broker) DeclareExchange(ctx context.Context, config *ExchangeConfig) error {
	if !b.IsConnected() {
		return ErrNotConnected
	}

	channel, err := b.conn.Channel()
	if err != nil {
		return err
	}
	defer channel.Close()

	err = channel.ExchangeDeclare(
		config.Name,
		string(config.Type),
		config.Durable,
		config.AutoDelete,
		config.Internal,
		config.NoWait,
		config.Arguments,
	)

	if err != nil {
		return err
	}

	b.logger.WithField("exchange", config.Name).Info("Exchange declared")
	return nil
}

// DeleteExchange deletes an exchange.
func (b *Broker) DeleteExchange(ctx context.Context, name string, ifUnused bool) error {
	if !b.IsConnected() {
		return ErrNotConnected
	}

	channel, err := b.conn.Channel()
	if err != nil {
		return err
	}
	defer channel.Close()

	return channel.ExchangeDelete(name, ifUnused, false)
}

// BindQueue binds a queue to an exchange.
func (b *Broker) BindQueue(ctx context.Context, queue, exchange, routingKey string, args amqp.Table) error {
	if !b.IsConnected() {
		return ErrNotConnected
	}

	channel, err := b.conn.Channel()
	if err != nil {
		return err
	}
	defer channel.Close()

	err = channel.QueueBind(queue, routingKey, exchange, false, args)
	if err != nil {
		return err
	}

	b.logger.WithFields(map[string]interface{}{
		"queue":       queue,
		"exchange":    exchange,
		"routing_key": routingKey,
	}).Info("Queue bound to exchange")

	return nil
}

// UnbindQueue unbinds a queue from an exchange.
func (b *Broker) UnbindQueue(ctx context.Context, queue, exchange, routingKey string, args amqp.Table) error {
	if !b.IsConnected() {
		return ErrNotConnected
	}

	channel, err := b.conn.Channel()
	if err != nil {
		return err
	}
	defer channel.Close()

	return channel.QueueUnbind(queue, routingKey, exchange, args)
}

// BindExchange binds an exchange to another exchange.
func (b *Broker) BindExchange(ctx context.Context, destination, source, routingKey string, args amqp.Table) error {
	if !b.IsConnected() {
		return ErrNotConnected
	}

	channel, err := b.conn.Channel()
	if err != nil {
		return err
	}
	defer channel.Close()

	return channel.ExchangeBind(destination, routingKey, source, false, args)
}

// UnbindExchange unbinds an exchange from another exchange.
func (b *Broker) UnbindExchange(ctx context.Context, destination, source, routingKey string, args amqp.Table) error {
	if !b.IsConnected() {
		return ErrNotConnected
	}

	channel, err := b.conn.Channel()
	if err != nil {
		return err
	}
	defer channel.Close()

	return channel.ExchangeUnbind(destination, routingKey, source, false, args)
}

// SetupTopology declares the standard HelixAgent exchanges and queues.
func (b *Broker) SetupTopology(ctx context.Context) error {
	// Declare task exchange
	if err := b.DeclareExchange(ctx, DefaultExchangeConfig("helixagent.tasks", ExchangeDirect)); err != nil {
		return err
	}

	// Declare events exchange (topic for routing)
	if err := b.DeclareExchange(ctx, DefaultExchangeConfig("helixagent.events", ExchangeTopic)); err != nil {
		return err
	}

	// Declare notifications exchange (fanout for broadcast)
	if err := b.DeclareExchange(ctx, DefaultExchangeConfig("helixagent.notifications", ExchangeFanout)); err != nil {
		return err
	}

	// Declare dead letter exchange
	if err := b.DeclareExchange(ctx, DefaultExchangeConfig("helixagent.dlx", ExchangeDirect)); err != nil {
		return err
	}

	// Declare standard queues
	queues := []string{
		"helixagent.tasks.background",
		"helixagent.tasks.llm",
		"helixagent.tasks.debate",
		"helixagent.tasks.verification",
		"helixagent.tasks.notifications",
		"helixagent.dlq",
		"helixagent.retry",
	}

	for _, queue := range queues {
		if err := b.DeclareQueue(ctx, queue); err != nil {
			return err
		}

		// Bind to task exchange
		if err := b.BindQueue(ctx, queue, "helixagent.tasks", queue, nil); err != nil {
			return err
		}
	}

	// Bind DLQ to dead letter exchange
	if err := b.BindQueue(ctx, "helixagent.dlq", "helixagent.dlx", "dlq", nil); err != nil {
		return err
	}

	b.logger.Info("RabbitMQ topology setup complete")
	return nil
}
