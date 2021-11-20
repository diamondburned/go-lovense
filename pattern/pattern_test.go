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

func TestParse(t *testing.T) {
	f := openFile(t, "testdata/edge")
	b := bufio.NewReaderSize(f, 38)

	header, points, err := Parse(b)
	if err != nil {
		t.Fatal("cannot parse testdata/edge:", err)
	}

	expectHeader := Header{
		Version: 1,
		Type:    "Edge",
		Motors:  []Motor{"v1", "v2"},
		Delay:   100 * time.Millisecond,
		MD5Sum:  "deadbeef",
	}

	if diff := deep.Equal(header, expectHeader); diff != nil {
		t.Fatalf("unexpected header: %s", diff)
	}

	expectPoints := [][]Point{
		{0, 1}, {1, 0}, {1, 0}, {0, 1}, {20, 0}, {0, 20}, {20, 20}, {0, 0},
		{0, 0}, {0, 0}, {0, 0}, {0, 0}, {0, 0}, {0, 0}, {0, 0}, {0, 0}, {0, 0},
		{0, 0}, {0, 0}, {0, 0}, {0, 0}, {0, 0}, {0, 0}, {0, 0}, {0, 0}, {0, 0},
		{0, 0}, {0, 0}, {0, 0}, {0, 0},
	}

	if diff := deep.Equal(points, expectPoints); diff != nil {
		t.Fatalf("unexpected points: %s", diff)
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
