package queue

import (
	"time"

	"github.com/VantageSports/common/credentials/google"

	"cloud.google.com/go/pubsub"
	"golang.org/x/net/context"
	"google.golang.org/api/option"
)

// InitPubSub returns a new google pubsub client. If the devHost is specified,
// the client is configured to talk to a pubsub server at that address.
func InitClient(creds *google.Creds) (*pubsub.Client, error) {
	ctx := context.Background()
	return pubsub.NewClient(ctx, creds.ProjectID, option.WithTokenSource(creds.TokenSource(ctx)))
}

// EnsureTopic is a nop if the topic already exists, otherwise it is created.
func EnsureTopic(client *pubsub.Client, topic string) (*pubsub.Topic, error) {
	ctx := context.Background()
	t := client.Topic(topic)

	exists, err := t.Exists(ctx)
	if exists || err != nil {
		return t, err
	}
	return client.CreateTopic(ctx, topic)
}

// EnsureSubscription is a nop if the subscription already exists, otherwise
// it is created. Note that if the subscription name exists with a different
// ackDeadline or push config, the existing subscription is NOT changed.
func EnsureSubscription(client *pubsub.Client, topic, subscription string, ackDeadline time.Duration, pushConf *pubsub.PushConfig) (*pubsub.Subscription, error) {
	ctx := context.Background()
	s := client.Subscription(subscription)

	exists, err := client.Subscription(subscription).Exists(ctx)
	if exists || err != nil {
		return s, err
	}

	t, err := EnsureTopic(client, topic)
	if err != nil {
		return nil, err
	}
	return client.CreateSubscription(ctx, subscription, t, ackDeadline, pushConf)
}
