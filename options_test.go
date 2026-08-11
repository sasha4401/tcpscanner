package tcpscanner

import (
	"reflect"
	"testing"
)

func TestRange(t *testing.T) {
	tests := []struct {
		name string
		from int
		to   int
		want []uint16
	}{
		{
			name: "normal",
			from: 1,
			to:   3,
			want: []uint16{1, 2, 3},
		},
		{
			name: "reverse",
			from: 3,
			to:   1,
			want: []uint16{1, 2, 3},
		},
		{
			name: "zero",
			from: 0,
			to:   3,
			want: []uint16{1, 2, 3},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Range(tt.from, tt.to)

			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("expected %v, got %v", tt.want, got)
			}
		})
	}

}

func TestList(t *testing.T) {
	got := List(
		"1",
		"2",
		"2",
		"10-12",
		"11",
	)

	want := []uint16{
		1, 2, 10, 11, 12,
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

func TestList_InvalidPorts(t *testing.T) {
	got := List(
		"0",
		"65536",
		"abc",
		"1-abc",
	)

	if len(got) != 0 {
		t.Fatalf("expected no valid ports, got %v", got)
	}
}

func TestList_Empty(t *testing.T) {
	got := List()

	if len(got) != 65535 {
		t.Fatalf("expected 65535 ports, got %d", len(got))
	}

	if got[0] != 1 {
		t.Fatalf("expected first port 1, got %d", got[0])
	}

	if got[len(got)-1] != 65535 {
		t.Fatalf(
			"expected last port 65535, got %d",
			got[len(got)-1],
		)
	}
}
