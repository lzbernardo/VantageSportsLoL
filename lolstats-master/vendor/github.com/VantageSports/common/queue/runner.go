package queue

import (
	"fmt"
	"time"

	"github.com/VantageSports/common/log"

	"cloud.google.com/go/pubsub"
	"golang.org/x/net/context"
)

// Handler is a function that accepts a context, a single-message, and performs
// some action in response. Handlers return error if processing failed (and
// should be retried), and nil otherwise. Handlers should be careful not to
// go too long between checking whether the supplied context has been canceled
// so that they don't do unecessary work in the event that the task lease was
// given up (or revoked) for some reason.
type Handler func(ctx context.Context, message *pubsub.Message) error

// TaskRunner runs a Handler in response to every message received for a
// given subscription.
type TaskRunner struct {
	sub           *pubsub.Subscription
	opts          []pubsub.PullOption
	ExitOnPullErr bool
	SleepAfterErr time.Duration
}

func NewTaskRunner(client *pubsub.Client, subscription string, maxPrefetch int, maxExtension time.Duration) *TaskRunner {
	return &TaskRunner{
		sub: client.Subscription(subscription),
		opts: []pubsub.PullOption{
			pubsub.MaxPrefetch(maxPrefetch),
			pubsub.MaxExtension(maxExtension),
		},
		SleepAfterErr: time.Second * 3,
	}
}

func (tr *TaskRunner) Start(ctx context.Context, handler Handler) error {
	for ctx.Err() == nil {
		err := tr.Process(ctx, handler)
		if err != nil {
			log.Warning(err)
			if tr.ExitOnPullErr {
				break
			}
			time.Sleep(tr.SleepAfterErr)
		}
	}

	log.Warning(fmt.Sprintf("stopping taskrunner: %v\n", ctx.Err()))
	return ctx.Err()
}

func (tr *TaskRunner) Process(ctx context.Context, handler Handler) error {
	it, err := tr.sub.Pull(ctx, tr.opts...)
	if err != nil {
		return err
	}
	defer it.Stop()

	msg, err := it.Next()
	if err != nil {
		return err
	}

	err = handler(ctx, msg)
	msg.Done(err == nil)
	return err
}
