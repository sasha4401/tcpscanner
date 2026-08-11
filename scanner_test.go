package tcpscanner

import (
	"context"
	"errors"
	"net"
	"strconv"
	"testing"
)

func TestScan_OpenPort(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()

	port := listener.Addr().(*net.TCPAddr).Port

	scanner, err := New()
	if err != nil {
		t.Fatal(err)
	}

	results, err := scanner.Scan(
		context.Background(),
		Hosts("127.0.0.1"),
		List(strconv.Itoa(port)),
	)

	if err != nil {
		t.Fatal(err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	result := results[0]

	if result.State != StateOpen {
		t.Fatalf("expected open, got %s, err=%v", result.State, result.Err)
	}

	if result.Port != uint16(port) {
		t.Fatalf("expected port %d, got %d", port, result.Port)
	}

	if result.IP == nil {
		t.Fatal("expected IP")
	}

	if result.Duration < 0 {
		t.Fatal("expected positive duration")
	}
}

func TestScan_ClosedPort(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:5")
	if err != nil {
		t.Fatal(err)
	}

	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()

	scanner, err := New()
	if err != nil {
		t.Fatal(err)
	}

	results, err := scanner.Scan(
		context.Background(),
		Hosts("127.0.0.1"),
		List(strconv.Itoa(port)),
	)
	if err != nil {
		t.Fatal(err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	if results[0].State != StateClosed {
		t.Fatalf(
			"expected closed, got %s, err=%v",
			results[0].State,
			results[0].Err,
		)
	}
}

func TestScan_MultiplePorts(t *testing.T) {
	listener1, err := net.Listen("tcp", "127.0.0.1:10")
	if err != nil {
		t.Fatal(err)
	}
	defer listener1.Close()

	listener2, err := net.Listen("tcp", "127.0.0.1:11")
	if err != nil {
		t.Fatal(err)
	}
	defer listener2.Close()

	port1 := listener1.Addr().(*net.TCPAddr).Port
	port2 := listener2.Addr().(*net.TCPAddr).Port

	scanner, err := New(WithConcurrency(2))
	if err != nil {
		t.Fatal(err)
	}

	results, err := scanner.Scan(
		context.Background(),
		Hosts("127.0.0.1"),
		List(strconv.Itoa(port1), strconv.Itoa(port2)),
	)
	if err != nil {
		t.Fatal(err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	for _, result := range results {
		if result.State != StateOpen {
			t.Errorf(
				"port %d: expected open, got %s",
				result.Port,
				result.State,
			)
		}
	}
}

func TestScan_MultipleHostsAndPorts(t *testing.T) {
	listener1, err := net.Listen("tcp", "127.0.0.1:80")
	if err != nil {
		t.Fatal(err)
	}
	defer listener1.Close()

	listener2, err := net.Listen("tcp", "127.0.0.1:79")
	if err != nil {
		t.Fatal(err)
	}
	defer listener2.Close()

	port1 := uint16(listener1.Addr().(*net.TCPAddr).Port)
	port2 := uint16(listener2.Addr().(*net.TCPAddr).Port)

	scanner, err := New(WithConcurrency(2))
	if err != nil {
		t.Fatal(err)
	}

	results, err := scanner.Scan(
		context.Background(),
		Hosts("127.0.0.1", "localhost"),
		List(strconv.Itoa(int(port1)), strconv.Itoa(int(port2)), "81"),
	)
	if err != nil {
		t.Fatal(err)
	}

	if len(results) != 6 {
		t.Fatalf("expected 6 results, got %d", len(results))
	}
}

func TestScan_Canceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	scanner, err := New()
	if err != nil {
		t.Fatal(err)
	}

	_, err = scanner.Scan(
		ctx,
		Hosts("127.0.0.1"),
		List("80"),
	)

	if err == nil {
		t.Fatal("expected error")
	}

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}
