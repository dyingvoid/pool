package pool

type Connection interface {
	Close() error
}

type Dialer func(int32) (Connection, error)