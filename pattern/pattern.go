package pattern

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"strconv"
	"time"
)

// Pattern describes a pattern file.
type Pattern struct {
	Header
	Points [][]Point
}

// Parse consumes r fully and returns the Lovense pattern reader and all its
// points. It adds onto Reader a few guarantees.
func Parse(r io.Reader) (*Pattern, error) {
	reader := NewReader(r)

	h, err := reader.ReadHeader()
	if err != nil {
		return nil, fmt.Errorf("cannot read header: %w", err)
	}

	p, err := reader.ReadAllPoints()
	if err != nil {
		return nil, fmt.Errorf("cannot read all points: %w", err)
	}

	if len(p) > 0 && len(p[0]) != len(h.Features) {
		return nil, fmt.Errorf("mismatch: %d motors != %d in points", len(h.Features), len(p[0]))
	}

	return &Pattern{
		Header: h,
		Points: p,
	}, nil
}

// Header describes the header of a Lovense pattern file. It is everything that
// sits before a hash symbol (#).
type Header struct {
	Version  int           // V
	Type     string        // T
	Features []Feature     // F
	Interval time.Duration // S
	MD5Sum   string        // M
}

// Feature is the type for the values in the F field.
type Feature string

// Feature constants that are known from the Lovense Desktop's source code. Note
// that no validation is done against these features.
const (
	AirPump  Feature = "p"
	Rotate   Feature = "r"
	Vibrate  Feature = "v"
	Vibrate1 Feature = "v1"
	Vibrate2 Feature = "v2"
)

// Point describes a single data point inside a Lovense pattern file. Its range
// is, in interval notation, [0, 20].
type Point uint8

// AsFloat converts the point into a floating-point number of range [0, 1].
func (pt Point) AsFloat() float64 {
	return float64(pt) / 20
}

// Reader provides a Lovense pattern reader.
type Reader struct {
	buf *bufio.Reader
	pts []Point
}

const bufferSize = 1 << 9 // 512 bytes

// NewReader creates a new reader from the given io.Reader.
func NewReader(r io.Reader) *Reader {
	buffer, ok := r.(*bufio.Reader)
	if !ok {
		buffer = bufio.NewReaderSize(r, bufferSize)
	}
	return &Reader{buffer, nil}
}

// ReadHeader reads the header. Note that the method will consume more bytes
// from the io.Reader than it needs to, since the reader is buffered.
func (r *Reader) ReadHeader() (Header, error) {
	header := Header{
		Version:  1,
		Interval: 100 * time.Millisecond,
	}

	// This reads maximum r.buf.Size() bytes.
	b, err := r.buf.ReadSlice('#')
	if err != nil {
		return header, err
	}

	// Discard the delimiter byte.
	r.buf.Discard(1)

	fields := bytes.Split(b, []byte(";"))

	for _, field := range fields {
		parts := bytes.SplitN(field, []byte(":"), 2)
		if len(parts) != 2 {
			continue
		}

		switch string(parts[0]) {
		case "V":
			v, err := strconv.Atoi(string(parts[1]))
			if err != nil {
				return header, fmt.Errorf("invalid version %q: %v", parts[1], err)
			}
			header.Version = v
		case "T":
			header.Type = string(parts[1])
		case "F":
			motors := bytes.Split(parts[1], []byte(","))
			header.Features = make([]Feature, len(motors))
			for i, motor := range motors {
				header.Features[i] = Feature(motor)
			}
		case "S":
			d, err := strconv.Atoi(string(parts[1]))
			if err != nil {
				return header, fmt.Errorf("invalid S value %q: %v", parts[1], err)
			}
			header.Interval = time.Duration(d) * time.Millisecond
		case "M":
			header.MD5Sum = string(parts[1])
		}
	}

	return header, nil
}

// ReadData reads a list of motor data points. The returned list is valid until
// the next call.
func (r *Reader) ReadPoints() ([]Point, error) {
	// TODO: retry until EOF or valid to skip spaces.
	b, err := r.buf.ReadSlice(';')
	if err != nil {
		return nil, err
	}

	parts := bytes.Split(b, []byte(","))
	points := make([]Point, len(parts))

	for i, part := range parts {
		v, err := strconv.Atoi(string(part))
		if err != nil {
			return nil, fmt.Errorf("invalid point %q: %v", part, err)
		}
		points[i] = Point(v)
	}

	return points, nil
}

// ReadAllData reads all data points. It guarantees that all point pairs in the
// slice will be equally sized.
func (r *Reader) ReadAllPoints() ([][]Point, error) {
	// backing slice that contains all points flattened out
	var backing []Point
	stride := -1

	// Peak to get the size for preallocating backing. We'll leave getting the
	// stride to the actual loop.
	b, err := r.buf.Peek(r.buf.Buffered())
	if err == nil {
		n := bytes.Count(b, []byte(";")) + bytes.Count(b, []byte(","))
		backing = make([]Point, 0, n)
	}

	for err == nil {
		b, err = r.buf.ReadSlice(';')
		if err != nil && !errors.Is(err, io.EOF) {
			// Early bail if the error isn't EOF.
			return nil, fmt.Errorf("cannot read: %w", err)
		}

		// Trim the trailing semicolon out, since ReadSlice includes it.
		b = bytes.TrimSuffix(b, []byte(";"))
		b = bytes.Trim(b, "\n")

		if len(b) == 0 {
			continue
		}

		// Count the stride for the first point tuple if we haven't one.
		if stride == -1 {
			// Add 1, since each number gets its comma except for the first
			// one.
			stride = bytes.Count(b, []byte(",")) + 1
		}

		pr := sepReader{b: b, s: ','}
		for i := 0; i < stride; i++ {
			v := pr.next()
			if v == nil {
				return nil, fmt.Errorf("%q doesn't have %d points", b, stride)
			}

			p, err := strconv.Atoi(string(v))
			if err != nil {
				return nil, fmt.Errorf("invalid point: %w", err)
			}

			backing = append(backing, Point(p))
		}
	}

	pairs := make([][]Point, 0, len(backing)/stride)

	for head := 0; head < len(backing); {
		tail := head + stride
		pairs = append(pairs, backing[head:tail])
		head = tail
	}

	return pairs, nil
}

type sepReader struct {
	b    []byte
	tail int
	s    byte
}

// next reads the next byte slice delimited by s.s. If there's nothing left, nil
// is returned.
func (s *sepReader) next() []byte {
	s.tail = bytes.IndexByte(s.b, s.s)
	if s.tail == -1 {
		b := s.b
		s.b = nil
		return b
	}

	b := s.b[:s.tail]
	s.b = s.b[s.tail+1:]
	return b
}
