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
	ctx           context.Context
	cancel        context.CancelFunc
	it            *pubsub.MessageIterator
	ExitOnPullErr bool
	SleepAfterErr time.Duration
}

func NewTaskRunner(client *pubsub.Client, subscription string, maxPrefetch int, maxExtension time.Duration) (*TaskRunner, error) {
	ctx, cancel := context.WithCancel(context.Background())

	opts := []pubsub.PullOption{
		pubsub.MaxPrefetch(maxPrefetch),
		pubsub.MaxExtension(maxExtension),
	}

	it, err := client.Subscription(subscription).Pull(ctx, opts...)
	return &TaskRunner{
		ctx:           ctx,
		cancel:        cancel,
		it:            it,
		SleepAfterErr: time.Second * 3,
	}, err
}

func (tr *TaskRunner) Stop() {
	tr.cancel()
}

func (tr *TaskRunner) Start(ctx context.Context, handler Handler) error {
	var (
		err error
		msg *pubsub.Message
	)

	for tr.ctx.Err() == nil {
		msg, err = tr.it.Next()
		if err != nil {
			log.Warning(err)
			if tr.ExitOnPullErr {
				break
			}
			time.Sleep(tr.SleepAfterErr)
			continue
		}

		taskCtx, _ := context.WithCancel(tr.ctx)
		if err = handler(taskCtx, msg); err != nil {
			log.Warning(fmt.Sprintf("error: %v\n", err.Error()))
			time.Sleep(tr.SleepAfterErr)
		}
		msg.Done(err == nil)
	}

	tr.it.Stop()
	return err
}
