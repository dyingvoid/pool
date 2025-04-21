package pool

import (
	"fmt"
	"sync"
)

type Pool struct {
	dialer Dialer

	connections map[int32]*connState
	openingConns sync.WaitGroup
	mu          sync.Mutex

	closed bool
}

func (p *Pool) GetConnection(addr int32) (Connection, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return nil, NewErrClosed()
	}

	state, exist := p.connections[addr]
	if !exist {
		state = &connState{
			wait: make(chan struct{}),
		}
		p.connections[addr] = state
		p.openingConns.Add(1)
		go p.dial(state, addr)
	}
	if state.status() == opening {
		p.mu.Unlock()
		<-state.wait
		p.mu.Lock()

		// !exists makes sure dialing goroutine cleans after itself
		if state.err != nil && !exist {
			delete(p.connections, addr)
		}
	}

	return state.conn, state.err
}

func (p *Pool) AddExternalConnection(addr int32, conn Connection) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return NewErrClosed()
	}

	state, exists := p.connections[addr]
	if exists {
		if state.status() == opening {
			state.conn = conn
			close(state.wait)
		} else if state.status() == established {
			return fmt.Errorf(
				"address %d: %w",
				addr,
				NewErrEstablished(),
			)
		} else if state.status() == failed {
			state.err = nil
			state.conn = conn
		}
	} else {
		p.connections[addr] = &connState{conn: conn}
	}

	return nil
}

func (p *Pool) Shutdown() {
	p.mu.Lock()

	for _, state := range p.connections {
		status := state.status()
		if status == established {
			state.conn.Close()
		}
		if status == opening {
			close(state.wait)
		}
		state.err = NewErrClosed()
	}
	p.closed = true
	p.mu.Unlock()

	p.openingConns.Wait()
}

func (p *Pool) dial(
	state *connState,
	addr int32,
) {
	defer p.openingConns.Done()
	conn, err := p.dialer(addr)
	p.mu.Lock()
	defer p.mu.Unlock()

	if state.status() != opening {
		if err == nil {
			conn.Close()
		}
		return
	}

	state.conn = conn
	state.err = err
	close(state.wait)
}

func New(dialer Dialer) (*Pool, error) {
	if dialer == nil {
		return nil, fmt.Errorf("dialer must be non null")
	}
	return &Pool{
		connections: map[int32]*connState{},
		dialer: dialer,
	}, nil
}
