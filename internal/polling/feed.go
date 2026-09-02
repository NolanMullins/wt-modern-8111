package polling

import (
	"fmt"
	"sort"

	"github.com/NolanMullins/wt-modern-8111/internal/telemetry"
	"github.com/NolanMullins/wt-modern-8111/internal/warthunder"
)

func (s *Service) appendFeedLocked(kind string, records []warthunder.FeedRecord) {
	now := s.now()
	for _, record := range records {
		key := fmt.Sprintf("%s:%d", kind, record.ID)
		if _, exists := s.feedKeys[key]; exists {
			continue
		}
		s.feedKeys[key] = struct{}{}
		s.raw.Feed = append(s.raw.Feed, telemetry.FeedEntry{
			Key:     key,
			Kind:    kind,
			Time:    record.Time,
			AddedAt: now,
			Sender:  record.Sender,
			Message: record.Message,
			Enemy:   record.Enemy,
		})
	}
	sort.SliceStable(s.raw.Feed, func(i, j int) bool {
		if !s.raw.Feed[i].AddedAt.Equal(s.raw.Feed[j].AddedAt) {
			return s.raw.Feed[i].AddedAt.After(s.raw.Feed[j].AddedAt)
		}
		return s.raw.Feed[i].Time > s.raw.Feed[j].Time
	})
	if len(s.raw.Feed) > maxFeedItems {
		removed := s.raw.Feed[maxFeedItems:]
		s.raw.Feed = s.raw.Feed[:maxFeedItems]
		for _, entry := range removed {
			delete(s.feedKeys, entry.Key)
		}
	}
}

func (s *Service) chatCursor() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastChatID
}

func (s *Service) hudCursors() (int, int) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastEventID, s.lastDamageID
}

func (s *Service) resetFeedSessionLocked() {
	s.lastChatID = 0
	s.lastEventID = 0
	s.lastDamageID = 0
	s.chatPrimed = false
	s.hudPrimed = false
	s.raw.Feed = make([]telemetry.FeedEntry, 0)
	s.feedKeys = make(map[string]struct{})
}
