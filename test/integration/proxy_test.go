//go:build integration

package integration_test

import (
	"io"
	"net"
	"sync"
	"testing"
)

type databaseProxy struct {
	address  string
	target   string
	listener net.Listener
	workers  sync.WaitGroup
}

func (p *databaseProxy) start(t *testing.T) {
	t.Helper()
	listener, err := net.Listen("tcp", p.address)
	if err != nil {
		t.Fatalf("start PostgreSQL proxy: %v", err)
	}
	p.address, p.listener = listener.Addr().String(), listener
	p.workers.Add(1)
	go func() {
		defer p.workers.Done()
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			p.workers.Add(1)
			go func() {
				defer p.workers.Done()
				defer conn.Close()
				upstream, err := net.Dial("tcp", p.target)
				if err != nil {
					return
				}
				defer upstream.Close()
				done := make(chan struct{})
				go func() {
					_, _ = io.Copy(conn, upstream)
					conn.Close()
					close(done)
				}()
				_, _ = io.Copy(upstream, conn)
				upstream.Close()
				<-done
			}()
		}
	}()
}

func (p *databaseProxy) stop(t *testing.T) {
	t.Helper()
	if p.listener == nil {
		return
	}
	if err := p.listener.Close(); err != nil {
		t.Errorf("stop PostgreSQL proxy: %v", err)
	}
	p.workers.Wait()
	p.listener = nil
}
