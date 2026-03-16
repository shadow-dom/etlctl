package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"shadow-dom/etlctl/pkg/util/etl"

	amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitMQSource struct {
	name          string
	url           string
	queue         string
	prefetchCount int
	timeout       time.Duration
	conn          *amqp.Connection
	ch            *amqp.Channel
	msgs          <-chan amqp.Delivery
}

func (s *RabbitMQSource) Name() string     { return s.name }
func (s *RabbitMQSource) Type() string     { return "rabbitmq" }
func (s *RabbitMQSource) IsListener() bool { return true }

func (s *RabbitMQSource) Connect() error {
	conn, err := amqp.Dial(s.url)
	if err != nil {
		return fmt.Errorf("rabbitmq dial failed: %w", err)
	}
	s.conn = conn

	ch, err := conn.Channel()
	if err != nil {
		return fmt.Errorf("rabbitmq channel failed: %w", err)
	}
	s.ch = ch

	if err := ch.Qos(s.prefetchCount, 0, false); err != nil {
		return fmt.Errorf("rabbitmq qos failed: %w", err)
	}

	_, err = ch.QueueDeclare(s.queue, true, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("rabbitmq queue declare failed: %w", err)
	}

	msgs, err := ch.Consume(s.queue, "", false, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("rabbitmq consume failed: %w", err)
	}
	s.msgs = msgs

	return nil
}

func (s *RabbitMQSource) Extract(query string) ([]map[string]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), s.timeout)
	defer cancel()

	var results []map[string]string

	for {
		select {
		case msg, ok := <-s.msgs:
			if !ok {
				return results, nil
			}

			var record map[string]any
			if err := json.Unmarshal(msg.Body, &record); err != nil {
				msg.Nack(false, false)
				continue
			}

			row := make(map[string]string)
			for k, v := range record {
				row[k] = fmt.Sprintf("%v", v)
			}
			results = append(results, row)
			msg.Ack(false)

		case <-ctx.Done():
			return results, nil
		}
	}
}

func (s *RabbitMQSource) Close() error {
	if s.ch != nil {
		s.ch.Close()
	}
	if s.conn != nil {
		s.conn.Close()
	}
	return nil
}

func (s *RabbitMQSource) ListTables() ([]string, error) {
	return []string{s.queue}, nil
}

func (s *RabbitMQSource) DescribeTable(table string) (*etl.TableSchema, error) {
	return &etl.TableSchema{Table: s.queue}, nil
}

func init() {
	etl.RegisterSource("rabbitmq", func(name string, config map[string]string) (etl.Source, error) {
		url := config["url"]
		if url == "" {
			return nil, fmt.Errorf("rabbitmq source %q requires 'url' in connection config", name)
		}
		queue := config["queue"]
		if queue == "" {
			return nil, fmt.Errorf("rabbitmq source %q requires 'queue' in connection config", name)
		}

		prefetch := 10
		if v := config["prefetch_count"]; v != "" {
			n, err := strconv.Atoi(v)
			if err != nil {
				return nil, fmt.Errorf("rabbitmq source %q: invalid prefetch_count: %w", name, err)
			}
			prefetch = n
		}

		timeout := 5 * time.Second
		if v := config["timeout"]; v != "" {
			d, err := time.ParseDuration(v)
			if err != nil {
				return nil, fmt.Errorf("rabbitmq source %q: invalid timeout: %w", name, err)
			}
			timeout = d
		}

		return &RabbitMQSource{
			name:          name,
			url:           url,
			queue:         queue,
			prefetchCount: prefetch,
			timeout:       timeout,
		}, nil
	})
}
