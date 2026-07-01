// Copyright (c) 2025 ynet Authors
// SPDX-License-Identifier: Apache-2.0

package rocketmq

import (
	"errors"
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/apache/rocketmq-client-go/v2"
	"github.com/apache/rocketmq-client-go/v2/consumer"
	"github.com/apache/rocketmq-client-go/v2/primitive"
	"github.com/apache/rocketmq-client-go/v2/producer"

	"github.com/coze-dev/coze-loop/backend/infra/mq"
)

type Factory struct{}

func NewFactory() mq.IFactory {
	return &Factory{}
}

func (f *Factory) NewProducer(config mq.ProducerConfig) (mq.IProducer, error) {
	if len(config.Addr) == 0 {
		return nil, errors.New("addr is empty")
	}
	opts := []producer.Option{
		producer.WithNsResolver(NewCustomResolver(resolveNamesrvAddrs(config.Addr))),
		producer.WithRetry(config.RetryTimes),
	}
	if config.ProduceTimeout > 0 {
		opts = append(opts, producer.WithSendMsgTimeout(config.ProduceTimeout))
	}
	if config.RetryTimes > 0 {
		opts = append(opts, producer.WithRetry(config.RetryTimes))
	}
	if config.ProducerGroup != nil {
		opts = append(opts, producer.WithGroupName(*config.ProducerGroup))
	}
	if getRmqNamesrvUser() != "" && getRmqNamesrvPassword() != "" {
		opts = append(opts, producer.WithCredentials(primitive.Credentials{
			AccessKey: getRmqNamesrvUser(),
			SecretKey: getRmqNamesrvPassword(),
		}))
	}

	p, err := rocketmq.NewProducer(opts...)
	if err != nil {
		return nil, err
	}
	return &Producer{producer: p}, nil
}

func (f *Factory) NewConsumer(config mq.ConsumerConfig) (mq.IConsumer, error) {
	if len(config.Addr) == 0 {
		return nil, errors.New("addr is empty")
	}
	if config.Topic == "" {
		return nil, errors.New("topic is empty")
	}
	if config.ConsumerGroup == "" {
		return nil, errors.New("consumer group is empty")
	}

	opts := []consumer.Option{
		consumer.WithNsResolver(NewCustomResolver(resolveNamesrvAddrs(config.Addr))),
		consumer.WithGroupName(config.ConsumerGroup),
		consumer.WithConsumerOrder(config.Orderly),
	}
	if config.ConsumeGoroutineNums > 0 {
		opts = append(opts, consumer.WithConsumeGoroutineNums(config.ConsumeGoroutineNums))
	}
	if config.ConsumeTimeout > 0 {
		opts = append(opts, consumer.WithConsumeTimeout(config.ConsumeTimeout))
	}
	var selector *consumer.MessageSelector
	if config.TagExpression != "" {
		selector = &consumer.MessageSelector{
			Type:       consumer.TAG,
			Expression: config.TagExpression,
		}
	}

	c, err := rocketmq.NewPushConsumer(opts...)
	if err != nil {
		return nil, err
	}
	return &Consumer{
		consumer: c,
		topic:    config.Topic,
		selector: selector,
	}, nil
}

func NewCustomResolver(addrs []string) primitive.NsResolver {
	return &customResolver{addrs: addrs}
}

type customResolver struct {
	addrs []string
}

func (c *customResolver) Resolve() []string {
	return normalizeNamesrvAddrs(c.addrs)
}

func (c *customResolver) Description() string {
	return fmt.Sprintf("custom resolver: %v", c.addrs)
}

func resolveNamesrvAddrs(fallback []string) []string {
	domain := getRmqNamesrvDomain()
	port := getRmqNamesrvPort()
	if domain != "" && port != "" {
		return normalizeNamesrvAddrs([]string{net.JoinHostPort(domain, port)})
	}

	addrs := make([]string, 0, len(fallback))
	for _, addr := range fallback {
		addr = strings.TrimSpace(addr)
		if addr != "" {
			addrs = append(addrs, addr)
		}
	}
	return normalizeNamesrvAddrs(addrs)
}

func normalizeNamesrvAddrs(addrs []string) []string {
	ret := make([]string, 0, len(addrs))
	for _, addr := range addrs {
		addr = strings.TrimSpace(addr)
		if addr == "" {
			continue
		}
		ret = append(ret, resolveNamesrvAddr(addr))
	}
	return ret
}

func resolveNamesrvAddr(addr string) string {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return addr
	}
	if net.ParseIP(host) != nil {
		return addr
	}
	ips, err := net.LookupHost(host)
	if err != nil || len(ips) == 0 {
		return addr
	}
	return net.JoinHostPort(ips[0], port)
}

func getRmqNamesrvDomain() string {
	return getLoopEnv("RMQ_NAMESRV_DOMAIN")
}

func getRmqNamesrvPort() string {
	return getLoopEnv("RMQ_NAMESRV_PORT")
}

func getRmqNamesrvUser() string {
	return getLoopEnv("RMQ_NAMESRV_USER")
}

func getRmqNamesrvPassword() string {
	return getLoopEnv("RMQ_NAMESRV_PASSWORD")
}

func getLoopEnv(name string) string {
	if v := os.Getenv("COZE_LOOP_" + name); v != "" {
		return v
	}
	return os.Getenv("YNET_LOOP_" + name)
}
