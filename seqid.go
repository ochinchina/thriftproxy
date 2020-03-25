package main

import (
	"sync"
	"sync/atomic"
)

type SeqIdAllocator struct {
	nextId int32
}

func NewSeqIdAllocator() *SeqIdAllocator {
	return &SeqIdAllocator{nextId: 0}
}

func (s *SeqIdAllocator) AllocId() int {
	return int(atomic.AddInt32(&s.nextId, 1))
}

type SeqIdMapper struct {
	sync.Mutex
	// map from newSeqId to oldSeqId
	idMapper map[int]int
}

func NewSeqIdMapper() *SeqIdMapper {
	return &SeqIdMapper{idMapper: make(map[int]int)}
}

func (s *SeqIdMapper) MapTo(oldSeqId int, newSeqId int) {
	s.Lock()
	defer s.Unlock()
	s.idMapper[newSeqId] = oldSeqId
}

func (s *SeqIdMapper) RemoveMap(newSeqId int) (int, bool) {
	s.Lock()
	defer s.Unlock()
	if oldSeqId, ok := s.idMapper[newSeqId]; ok {
		delete(s.idMapper, newSeqId)
		return oldSeqId, true
	}
	return 0, false
}

func (s *SeqIdMapper) Size() int {
	s.Lock()
	defer s.Unlock()

	return len(s.idMapper)
}
