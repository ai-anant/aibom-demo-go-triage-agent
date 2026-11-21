// Package store keeps tickets in memory for the duration of the demo process.
package store

import (
	"fmt"
	"sync"
	"time"

	"github.com/ai-anant/aibom-demo-go-triage-agent/internal/triage"
)

// Record is a stored ticket together with its classification.
type Record struct {
	triage.Ticket
	Result triage.Result `json:"result"`
}

// Store is a concurrency-safe in-memory ticket store.
type Store struct {
	mu      sync.RWMutex
	records map[string]Record
	order   []string
	next    int
}

// New returns an empty store.
func New() *Store {
	return &Store{records: map[string]Record{}}
}

// Add stores a ticket and assigns an identifier.
func (s *Store) Add(ticket triage.Ticket, result triage.Result) Record {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.next++
	ticket.ID = fmt.Sprintf("TRG-%04d", s.next)
	if ticket.Created.IsZero() {
		ticket.Created = time.Now().UTC()
	}

	record := Record{Ticket: ticket, Result: result}
	s.records[ticket.ID] = record
	s.order = append(s.order, ticket.ID)
	return record
}

// Get returns one record by identifier.
func (s *Store) Get(id string) (Record, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	record, ok := s.records[id]
	return record, ok
}

// List returns every record in insertion order.
func (s *Store) List() []Record {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Record, 0, len(s.order))
	for _, id := range s.order {
		out = append(out, s.records[id])
	}
	return out
}
