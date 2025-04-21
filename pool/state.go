package pool

type connState struct {
	conn Connection
	err error
	wait chan struct{}
}

func (s *connState) status() status {
	if s.err != nil {
		return failed
	}
	if s.conn == nil {
		return opening
	}

	return established
}
