package domains

type Listening func() error

type StopListening func()