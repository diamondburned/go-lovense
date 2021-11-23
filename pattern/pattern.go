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
	Points Points
}

// Parse consumes r fully and returns the Lovense pattern reader and all its
// points. It adds onto Reader a few guarantees.
func Parse(r io.Reader) (*Pattern, error) {
	reader := NewReader(r)

	h, err := reader.ReadHeader()
	if err != nil {
		return nil, fmt.Errorf("cannot read header: %w", err)
	}

	var p Points

	switch h.Version {
	case V0:
		p, err = reader.ReadAllV0Points()
		if err != nil {
			return nil, fmt.Errorf("cannot read all v0 points: %w", err)
		}
	case V1:
		p, err = reader.ReadAllV1Points()
		if err != nil {
			return nil, fmt.Errorf("cannot read all v1 points: %w", err)
		}
	case 2:
		return nil, fmt.Errorf("unknown version %d", h.Version)
	}

	if len(p) > 0 && len(p[0]) != len(h.Features) {
		return nil, fmt.Errorf("mismatch: %d motors != %d in points", len(h.Features), len(p[0]))
	}

	return &Pattern{
		Header: h,
		Points: p,
	}, nil
}

// Version is the version of the pattern.
type Version int

const (
	V0 Version = 0
	V1 Version = 1
)

// String returns version in "V:n" format.
func (v Version) String() string {
	return fmt.Sprintf("V:%d", int(v))
}

// Header describes the header of a Lovense pattern file. It is everything that
// sits before a hash symbol (#) in a version 1 pattern file. All header fields
// are not guaranteed except for Interval.
type Header struct {
	Version  Version       // V
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

// Strength describes a single strength point inside a Lovense pattern file.
type Strength uint8

// Scale scales the strength to a number within [0.0, 1.0].
func (s Strength) Scale(v Version) float64 {
	switch v {
	case V0:
		return clampF(float64(s) / 100)
	case V1:
		return clampF(float64(s) / 20)
	default:
		return 0
	}
}

func clampF(f float64) float64 {
	if f < 0 {
		return 0
	}
	if f > 1 {
		return 1
	}
	return f
}

// Point describes the strengths of motors in an instant of time, or a point, in
// a Lovense pattern file. Version 0 pattern files will always only have a
// single motor, while version 1 pattern files can have more.
type Point []Strength

// Scale scales the point (list of strengths) into floats within range [0.0,
// 1.0].
func (p Point) Scale(v Version) []float64 {
	return p.ScaleAppend(v, nil)
}

// ScaleAppend is the append version of Scale.
func (p Point) ScaleAppend(v Version, buf []float64) []float64 {
	if cap(buf) < len(p) {
		buf = make([]float64, 0, len(p))
	}
	for _, s := range p {
		buf = append(buf, s.Scale(v))
	}
	return buf
}

// Points contains a list of points, each containing a list of vibration
// strength numbers. It holds multiple points representing multiple instants of
// time incremented by the Interval.
type Points []Point

// Reader provides a Lovense pattern reader.
type Reader struct {
	buf *bufio.Reader
}

// NewReader creates a new reader from the given io.Reader.
func NewReader(r io.Reader) *Reader {
	buffer, ok := r.(*bufio.Reader)
	if !ok {
		buffer = bufio.NewReader(r)
	}
	return &Reader{buffer}
}

var spaces = [255]bool{
	' ':  true,
	'\t': true,
	'\n': true,
	'\r': true,
}

// ReadHeader reads the header. Note that the method will consume more bytes
// from the io.Reader than it needs to, since the reader is buffered.
func (r *Reader) ReadHeader() (Header, error) {
	header := Header{
		Version:  0,
		Features: []Feature{"v"},
		Interval: 100 * time.Millisecond,
	}

	// Peek the next 2 bytes. If it's "V:", then we can read the version.
	// Otherwise, it's version 0.
	versionHeader, err := r.buf.Peek(2)
	if err != nil {
		return header, fmt.Errorf("cannot peek version: %w", err)
	}

	if string(versionHeader) != "V:" {
		return header, nil
	}

	// This reads maximum r.buf.Size() bytes.
	b, err := r.buf.ReadSlice('#')
	if err != nil {
		return header, err
	}

	// Discard the delimiter byte.
	b = bytes.TrimSuffix(b, []byte("#"))

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
			header.Version = Version(v)
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

// ReadAllV0Points reads all data points in a version 0 pattern file.
// Version 0 is not capable of containing data for more than 1 motor, so the
// length of the inner slice is always 1.
func (r *Reader) ReadAllV0Points() (Points, error) {
	var points Points

	// Peak to get the size for preallocating backing. We'll leave getting the
	// stride to the actual loop.
	b, err := r.buf.Peek(r.buf.Buffered())
	if err == nil {
		n := bytes.Count(b, []byte(",")) + 1
		points = make(Points, 0, n)
	}

	for err == nil {
		b, err = r.buf.ReadSlice(',')
		if err != nil && !errors.Is(err, io.EOF) {
			return points, fmt.Errorf("cannot read v0 point: %w", err)
		}

		b = bytes.TrimSuffix(b, []byte(","))
		b = bytes.TrimSpace(b)

		if len(b) == 0 {
			continue
		}

		p, err := strconv.ParseUint(string(b), 10, 8)
		if err != nil {
			return points, fmt.Errorf("error parsing v0 point: %w", err)
		}

		points = append(points, Point{Strength(p)})
	}

	return points, nil
}

// ReadV1Points reads a list of motor data points in a version 1 pattern file.
func (r *Reader) ReadV1Points() (Point, error) {
	// TODO: retry until EOF or valid to skip spaces.
	b, err := r.buf.ReadSlice(';')
	if err != nil {
		return nil, err
	}

	parts := bytes.Split(b, []byte(","))
	point := make(Point, len(parts))

	for i, part := range parts {
		v, err := strconv.Atoi(string(part))
		if err != nil {
			return nil, fmt.Errorf("invalid point %q: %v", part, err)
		}
		point[i] = Strength(v)
	}

	return point, nil
}

// ReadAllV2Data reads all data points in a version 1 pattern file. It
// guarantees that all point pairs in the slice will be equally sized.
func (r *Reader) ReadAllV1Points() (Points, error) {
	// backing slice that contains all points flattened out
	var backing []Strength
	stride := -1

	// Peak to get the size for preallocating backing. We'll leave getting the
	// stride to the actual loop.
	b, err := r.buf.Peek(r.buf.Buffered())
	if err == nil {
		n := bytes.Count(b, []byte(";")) + bytes.Count(b, []byte(",")) + 1
		backing = make([]Strength, 0, n)
	}

	for err == nil {
		b, err = r.buf.ReadSlice(';')
		if err != nil && !errors.Is(err, io.EOF) {
			// Early bail if the error isn't EOF.
			return nil, fmt.Errorf("cannot read: %w", err)
		}

		// Trim the trailing semicolon out, since ReadSlice includes it.
		b = bytes.TrimSuffix(b, []byte(";"))
		b = bytes.TrimSpace(b)

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

			p, err := strconv.ParseUint(string(v), 10, 8)
			if err != nil {
				return nil, fmt.Errorf("invalid point: %w", err)
			}

			backing = append(backing, Strength(p))
		}
	}

	pairs := make(Points, 0, len(backing)/stride)

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
