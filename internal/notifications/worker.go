package notifications

import (
	"context"
	"errors"
	"log/slog"
	"time"
)

type Worker struct {
	queue   Queue
	senders []Sender
}

func NewWorker(queue Queue, senders []Sender) *Worker {
	return &Worker{
		queue:   queue,
		senders: senders,
	}
}

func (w *Worker) Run(ctx context.Context) {
	for {
		notification, err := w.queue.Consume(ctx)

		if err != nil {
			slog.Error(
				"ошибка Consume",
				"error", err,
				"is_context_canceled", errors.Is(err, context.Canceled),
				"ctx_err", ctx.Err(),
			)

			if errors.Is(err, context.Canceled) {
				slog.Info("notification worker получил отмену контекста")
				return
			}

			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Second):
				continue
			}
		}

		if notification == (Notification{}) {
			continue
		}

		for _, sender := range w.senders {
			if err := sender.Send(notification); err != nil {
				slog.Error(
					"не удалось отправить уведомление",
					"error", err,
				)
			}
		}
	}
}
