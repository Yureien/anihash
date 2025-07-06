package anidb

import "sync"

type syncVar[T any] struct {
	val T
	mu  sync.Mutex
}

func (s *syncVar[T]) get() T {
	s.mu.Lock()
	v := s.val
	s.mu.Unlock()
	return v
}

func (s *syncVar[T]) set(v T) {
	s.mu.Lock()
	s.val = v
	s.mu.Unlock()
}
