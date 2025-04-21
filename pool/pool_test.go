package pool

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type testConn struct {
	adress int32
}

func (t *testConn) Close() error {
	return nil
}

func TestPoolDialAndGetConnection(t *testing.T) {
	dial := make(chan struct{}, 1)
	ready := make(chan struct{}, 1)
	dialer := func(address int32) (Connection, error) {
		ready <- struct{}{}
		<-dial
		return &testConn{adress: address}, nil
	}
	p, err := New(dialer)
	if err != nil {
		t.Fatalf("failed to create pool: %v", err)
	}
	received := make(chan Connection, 1)
	go func() {
		conn, err := p.GetConnection(1)
		if err != nil {
			received <- nil
			return
		}
		received <- conn
	}()
	<-ready
	dial <- struct{}{}
	conn := <-received
	if conn == nil {
		t.Fatalf("expected connection, got nil")
	}
	if tc, ok := conn.(*testConn); !ok || tc.adress != 1 {
		t.Errorf("expected address 1, got %+v", conn)
	}
}

func TestPoolShutdown(t *testing.T) {
	dial := make(chan struct{}, 1)
	ready := make(chan struct{}, 1)
	dialer := func(address int32) (Connection, error) {
		ready <- struct{}{}
		<-dial
		return &testConn{adress: address}, nil
	}
	p, err := New(dialer)
	if err != nil {
		t.Fatalf("failed to create pool: %v", err)
	}
	received := make(chan error, 1)
	go func() {
		_, err := p.GetConnection(1)
		received <- err
	}()
	<-ready
	go func() {
		time.Sleep(time.Millisecond)
		dial <- struct{}{}
	}()
	p.Shutdown()
	err = <-received
	if err == nil || err.Error() != "connection pool's closed" {
		t.Errorf("expected closed error, got %v", err)
	}
}

func TestPoolConcurrentDial(t *testing.T) {
	var cnt uint64
	dialer := func(address int32) (Connection, error) {
		return &testConn{adress: address}, nil
	}
	p, err := New(func(address int32) (Connection, error) {
		atomic.AddUint64(&cnt, 1)
		return dialer(address)
	})
	if err != nil {
		t.Fatalf("failed to create pool: %v", err)
	}
	var wg sync.WaitGroup
	n := 10
	wg.Add(n)
	received := make(chan Connection, n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			conn, _ := p.GetConnection(1)
			received <- conn
		}()
	}
	wg.Wait()
	close(received)
	for conn := range received {
		if tc, ok := conn.(*testConn); !ok || tc.adress != 1 {
			t.Errorf("expected address 1, got %+v", conn)
		}
	}
	if cnt != 1 {
		t.Errorf("expected 1 dial, got %d", cnt)
	}
}

func TestConnectAfterError(t *testing.T) {
	var expectErr error
	dialer := func(address int32) (Connection, error) {
		if expectErr != nil {
			return nil, expectErr
		}
		return &testConn{adress: address}, nil
	}
	p, err := New(dialer)
	if err != nil {
		t.Fatalf("failed to create pool: %v", err)
	}
	expectErr = errors.New("test")
	conn, err := p.GetConnection(1)
	if conn != nil {
		t.Errorf("expected nil conn, got %+v", conn)
	}
	if !errors.Is(err, expectErr) {
		t.Errorf("expected error %v, got %v", expectErr, err)
	}

	expectErr = nil
	conn, err = p.GetConnection(1)
	if conn == nil || err != nil {
		t.Errorf("expected valid conn, got conn=%+v, err=%v", conn, err)
	}
}

func TestDifferentNotBlocked(t *testing.T) {
	addr := int32(10)
	dial := make(chan struct{}, 1)
	ready := make(chan struct{}, 1)
	dialer := func(address int32) (Connection, error) {
		if address == addr {
			ready <- struct{}{}
			<-dial
		}
		return &testConn{adress: address}, nil
	}
	p, err := New(dialer)
	if err != nil {
		t.Fatalf("failed to create pool: %v", err)
	}
	received := make(chan Connection, 1)
	go func() {
		conn, _ := p.GetConnection(addr)
		received <- conn
	}()
	<-ready
	conn, _ := p.GetConnection(1)
	if tc, ok := conn.(*testConn); !ok || tc.adress != 1 {
		t.Errorf("expected address 1, got %+v", conn)
	}
	dial <- struct{}{}
	if tc, ok := (<-received).(*testConn); !ok || tc.adress != addr {
		t.Errorf("expected address %d, got %+v", addr, tc)
	}
}
