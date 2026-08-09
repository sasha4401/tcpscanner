package tcpscanner

import (
	"context"
	"log/slog"
	"net"
	"net/netip"
	"strconv"
	"strings"
	"time"
)

var maxPort uint16 = ^uint16(0)

type Option func(*Scanner)

func WithConcurrency(n int) Option {
	return func(s *Scanner) {
		s.Concurrency = n
	}
}

func WithTimeout(n time.Duration) Option {
	return func(s *Scanner) {
		s.Timeout = n
	}
}

func Range(f, l int) []uint16 {
	if l < f {
		l, f = f, l
	}

	if f <= 0 {
		f = 1
	}

	if l >= int(maxPort) {
		l = int(maxPort)
	}

	size := l - f + 1
	ports := make([]uint16, size)
	for i := range ports {
		ports[i] = uint16(f) + uint16(i)
	}

	return ports
}

func List(ports ...string) []uint16 {
	if len(ports) == 0 {
		res := make([]uint16, maxPort)
		for i := 1; i <= int(maxPort); i++ {
			res[i-1] = uint16(i)
		}

		return res
	}

	portSets := make(map[uint16]struct{})
	validPorts := make([]uint16, 0, len(ports))
	for _, v := range ports {
		if strings.Contains(v, "-") {
			ran := strings.Split(v, "-")
			if len(ran) != 2 {
				slog.Warn("Range: " + v + " incorrect")
				continue
			}

			f, err := strconv.Atoi(ran[0])
			if err != nil {
				slog.Warn("Port: " + ran[0] + " incorrect")
				continue
			}

			l, err := strconv.Atoi(ran[1])
			if err != nil {
				slog.Warn("Port: " + ran[1] + " incorrect")
				continue
			}

			res := Range(f, l)
			uniqRes := make([]uint16, 0, len(res))
			for _, v := range res {
				if _, ok := portSets[v]; !ok {
					uniqRes = append(uniqRes, v)
					portSets[v] = struct{}{}
				}
			}

			validPorts = append(validPorts, uniqRes...)
			continue
		}

		check, err := strconv.Atoi(v)
		if err != nil {
			slog.Warn("Port: " + v + " incorrect")
			continue
		}

		if check <= 0 || check > int(maxPort) {
			slog.Warn("Port: " + strconv.Itoa(check) + " incorrect")
			continue
		}

		port := uint16(check)

		if _, ok := portSets[port]; ok {
			continue
		}

		portSets[port] = struct{}{}
		validPorts = append(validPorts, port)
	}

	return validPorts
}

func Hosts(hosts ...string) []string {
	if len(hosts) == 0 {
		slog.Warn("Hosts list is Empty")
		return []string{}
	}

	validHosts := make([]string, 0, len(hosts))
	for _, v := range hosts {
		if _, err := netip.ParseAddr(v); err == nil {
			validHosts = append(validHosts, v)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		ips, err := net.DefaultResolver.LookupIP(ctx, "", v)
		cancel()
		if err != nil {
			slog.Warn("Incorrect ip: " + v)
			continue
		}

		for _, i := range ips {
			validHosts = append(validHosts, i.String())
		}
	}

	return validHosts
}
