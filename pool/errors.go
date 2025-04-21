package pool

type ErrClosed struct {
	msg string
}

func (e *ErrClosed) Error() string {
	return e.msg
}

func NewErrClosed() *ErrClosed {
	return &ErrClosed{
		msg: "connection pool's closed",
	}
}

type ErrEstablished struct {
	msg string
}

func (e *ErrEstablished) Error() string {
	return e.msg
}

func NewErrEstablished() *ErrEstablished {
	return &ErrEstablished{
		msg: "connection already established",
	}
}