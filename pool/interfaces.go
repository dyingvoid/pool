package pool

type Connection interface {
	Close() error
}
