package notifications

import (
	"context"
	"errors"
	"log/slog"
	"time"
)

// Worker получает уведомления из очереди и передаёт их зарегистрированным отправителям
type Worker struct {
	queue   Queue
	senders []Sender
}

// NewWorker создаёт worker с указанной очередью и отправителями
func NewWorker(queue Queue, senders []Sender) *Worker {
	return &Worker{
		queue:   queue,
		senders: senders,
	}
}

// Run запускает цикл обработки уведомлений до отмены контекста.
// Ошибки чтения из очереди повторяются с задержкой
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
			if err := sender.Send(ctx, notification); err != nil {
				slog.Error(
					"не удалось отправить уведомление",
					"error", err,
				)
			}
		}
	}
}
