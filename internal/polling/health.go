package polling

import (
	"time"

	"github.com/NolanMullins/wt-modern-8111/internal/telemetry"
)

type sourceRecord struct {
	lastSuccess  time.Time
	lastError    error
	firstFailure time.Time
}

const gameSessionFailureGrace = 2 * time.Second

func (s *Service) recordFailure(name string, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	record := s.sourceLocked(name)
	record.lastError = err
	now := s.now()
	if record.firstFailure.IsZero() {
		record.firstFailure = now
	}
	if name == "state" && now.Sub(record.firstFailure) >= gameSessionFailureGrace {
		s.resetGameSessionLocked()
		record.firstFailure = now
	}
}

func (s *Service) recordSuccessLocked(name string) {
	record := s.sourceLocked(name)
	record.lastSuccess = s.now()
	record.lastError = nil
	record.firstFailure = time.Time{}
}

func (s *Service) sourceLocked(name string) *sourceRecord {
	record, ok := s.sources[name]
	if !ok {
		record = &sourceRecord{}
		s.sources[name] = record
	}
	return record
}

func (s *Service) sourceStatusesLocked(now time.Time) map[string]telemetry.SourceStatus {
	statuses := make(map[string]telemetry.SourceStatus, len(s.sources))
	for name, source := range s.sources {
		status := telemetry.SourceStatus{State: "unavailable"}
		if !source.lastSuccess.IsZero() {
			lastSuccess := source.lastSuccess
			age := now.Sub(lastSuccess).Milliseconds()
			status.LastSuccess = &lastSuccess
			status.AgeMS = &age
			status.State = "fresh"
			if name != "mapImage" && age > 2*slowInterval.Milliseconds() {
				status.State = "stale"
			}
		}
		if source.lastError != nil {
			status.Error = source.lastError.Error()
			if source.lastSuccess.IsZero() || status.State == "stale" {
				status.State = "error"
			}
		}
		statuses[name] = status
	}
	return statuses
}
