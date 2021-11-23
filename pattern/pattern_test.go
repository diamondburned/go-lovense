package pattern

import (
	"bufio"
	"io"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/go-test/deep"
)

func TestSepReader(t *testing.T) {
	b := []byte("0,1,2,3,4,5,6,7,8,9,10")
	n := []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10}

	r := sepReader{
		b: b,
		s: ',',
	}

	var i int

	for b := r.next(); b != nil; b = r.next() {
		v, err := strconv.Atoi(string(b))
		if err != nil {
			t.Error("b not int:", err)
			continue
		}
		if v != n[i] {
			t.Errorf("expected %d, got %d", n[i], v)
			continue
		}
		i++
	}
}

func TestParseV1(t *testing.T) {
	f := openFile(t, "testdata/edge")
	b := bufio.NewReaderSize(f, 38)

	p, err := Parse(b)
	if err != nil {
		t.Fatal("cannot parse testdata/edge:", err)
	}

	expect := &Pattern{
		Header: Header{
			Version:  1,
			Type:     "Edge",
			Features: []Feature{Vibrate1, Vibrate2},
			Interval: 100 * time.Millisecond,
			MD5Sum:   "deadbeef",
		},
		Points: Points{
			{0, 1}, {1, 0}, {1, 0}, {0, 1}, {20, 0}, {0, 20}, {20, 20}, {0, 0},
			{0, 0}, {0, 0}, {0, 0}, {0, 0}, {0, 0}, {0, 0}, {0, 0}, {0, 0},
			{0, 0}, {0, 0}, {0, 0}, {0, 0}, {0, 0}, {0, 0}, {0, 0}, {0, 0},
			{0, 0}, {0, 0}, {0, 0}, {0, 0}, {0, 0}, {0, 0},
		},
	}

	if diff := deep.Equal(p, expect); diff != nil {
		t.Fatalf("unexpected pattern: %s", diff)
	}
}

func TestParseV0(t *testing.T) {
	f := openFile(t, "testdata/v0")
	b := bufio.NewReaderSize(f, 12)

	p, err := Parse(b)
	if err != nil {
		t.Fatal("cannot parse testdata/v0:", err)
	}

	expect := &Pattern{
		Header: Header{
			Version:  0,
			Type:     "",
			Features: []Feature{Vibrate},
			Interval: 100 * time.Millisecond,
			MD5Sum:   "",
		},
		Points: Points{
			{0}, {0}, {0}, {0}, {0}, {0}, {0}, {0}, {0}, {0}, {0}, {0},
			{0}, {0}, {0}, {0}, {0}, {0}, {0}, {0}, {0}, {0}, {0}, {0},
			{0}, {0}, {0}, {0}, {0}, {0}, {0}, {0}, {0}, {0}, {0}, {0},
			{0}, {0}, {0}, {0}, {0}, {8}, {8}, {8}, {7}, {7}, {7}, {6},
			{0}, {0}, {0}, {0}, {0}, {0}, {0}, {0}, {0}, {0}, {0}, {0},
			{0}, {0}, {0}, {0}, {0}, {0}, {0}, {0}, {0}, {0}, {0}, {0},
			{0}, {0}, {0}, {0}, {0}, {0}, {0}, {0}, {0}, {0}, {0}, {0},
			{0}, {0}, {0}, {0}, {0}, {0}, {0}, {0}, {0}, {0}, {0}, {0},
			{0}, {0}, {0}, {0}, {0}, {0}, {0}, {3}, {3}, {4}, {4}, {3},
		},
	}

	if diff := deep.Equal(p, expect); diff != nil {
		t.Fatalf("unexpected pattern: %s", diff)
	}
}

func openFile(t *testing.T, name string) io.Reader {
	f, err := os.Open(name)
	if err != nil {
		t.Fatalf("cannot read %s: %v", name, err)
	}
	t.Cleanup(func() { f.Close() })

	return f
}
