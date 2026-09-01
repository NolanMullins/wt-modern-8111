package polling

import "github.com/NolanMullins/wt-modern-8111/internal/warthunder"

func (s *Service) currentMapGeneration() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.mapGeneration
}

func (s *Service) currentMapEpoch() uint64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.mapEpoch
}

func (s *Service) resetMapSessionLocked() {
	s.mapEpoch++
	s.raw.MapObjects = make([]warthunder.MapObject, 0)
	s.sources["mapObjects"] = &sourceRecord{}
	s.resetFeedSessionLocked()
	s.resetRTBLocked()
	s.allyMarks = nil
	s.destroyed = false
	if s.identity != nil {
		s.identity.ResetSession()
	}
}

func (s *Service) resetGameSessionLocked() {
	s.resetMapSessionLocked()
	s.mapImage = nil
	s.mapImageType = ""
	s.mapGeneration = 0
}
