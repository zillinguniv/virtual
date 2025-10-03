package metrics

import (
	"fmt"
	"sync"
	"time"
)

type Gauge struct {
	mu   sync.Mutex
	name string
	val  int64
}

func NewGauge(name string) *Gauge {
	return &Gauge{name: name}
}

func (g *Gauge) Set(v int64) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.val = v
}

func (g *Gauge) Value() int64 {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.val
}

func (g *Gauge) Print() {
	fmt.Printf("[METRIC] %s = %d\n", g.name, g.Value())
}

// Simple timer (simulate ResettingTimer)
type Timer struct {
	mu      sync.Mutex
	name    string
	records []time.Duration
}

func NewTimer(name string) *Timer {
	return &Timer{name: name}
}

func (t *Timer) Record(d time.Duration) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.records = append(t.records, d)
}

func (t *Timer) ResetAndAvg() time.Duration {
	t.mu.Lock()
	defer t.mu.Unlock()
	if len(t.records) == 0 {
		return 0
	}
	var total time.Duration
	for _, r := range t.records {
		total += r
	}
	avg := total / time.Duration(len(t.records))
	t.records = nil
	return avg
}
