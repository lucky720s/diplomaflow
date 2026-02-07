package project

type App struct {
	Handler           *Handler
	DeadlineScheduler *DeadlineScheduler
}

func NewApp(h *Handler, scheduler *DeadlineScheduler) *App {
	return &App{
		Handler:           h,
		DeadlineScheduler: scheduler,
	}
}
