package project

type App struct {
	Handler           *Handler
	DeadlineScheduler *DeadlineScheduler
	OutboxProcessor   *OutboxProcessor
}

func NewApp(h *Handler, scheduler *DeadlineScheduler, outbox *OutboxProcessor) *App {
	return &App{
		Handler:           h,
		DeadlineScheduler: scheduler,
		OutboxProcessor:   outbox,
	}
}
