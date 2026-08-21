package platform

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync/atomic"
	"time"
)

type SystemClock struct{}

func (SystemClock) Now() time.Time {
	return time.Now().UTC()
}

type MonotonicIDGenerator struct {
	counter atomic.Uint64
}

func (g *MonotonicIDGenerator) NewID(prefix string) string {
	buffer := make([]byte, 8)
	if _, err := rand.Read(buffer); err == nil {
		return fmt.Sprintf("%s-%s", prefix, hex.EncodeToString(buffer))
	}
	sequence := g.counter.Add(1)
	return fmt.Sprintf("%s-%d-%d", prefix, time.Now().UTC().UnixNano(), sequence)
}

type FixedClock struct {
	Value time.Time
}

func (c FixedClock) Now() time.Time {
	return c.Value.UTC()
}

type SequenceIDGenerator struct {
	Prefix  string
	counter atomic.Uint64
}

func (g *SequenceIDGenerator) NewID(prefix string) string {
	if g.Prefix != "" {
		prefix = g.Prefix
	}
	return fmt.Sprintf("%s-%06d", prefix, g.counter.Add(1))
}
