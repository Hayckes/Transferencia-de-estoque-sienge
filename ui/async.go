package ui

type AsyncRunner struct {
	Dispatch func(func())
}

func NewAsyncRunner(dispatch func(func())) AsyncRunner {
	if dispatch == nil {
		dispatch = func(fn func()) { fn() }
	}

	return AsyncRunner{Dispatch: dispatch}
}

func (r AsyncRunner) Run(operation func() error, done func(error)) {
	go func() {
		err := operation()
		r.Dispatch(func() {
			if done != nil {
				done(err)
			}
		})
	}()
}
