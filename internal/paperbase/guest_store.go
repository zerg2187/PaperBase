package paperbase

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// GuestStore はゲストセッションの論文をインメモリで保持するインターフェース
type GuestStore interface {
	StorePaper(sessionID string, paper *Paper) error
	GetPapers(sessionID string, offset, limit int) []Paper
	CountPapers(sessionID string) int
	DeletePaper(sessionID, paperID string) error
	SearchPapers(sessionID, query string) []Paper
	Exists(sessionID, paperID string) bool
}

const (
	guestPaperRegisterLimit = 5
	guestSessionTTL         = 24 * time.Hour
)

type guestSession struct {
	papers     []*Paper
	lastAccess time.Time
}

type memoryGuestStore struct {
	mu       sync.RWMutex
	sessions map[string]*guestSession
}

// NewGuestStore は新しいゲストストレージを作成する
func NewGuestStore() GuestStore {
	return &memoryGuestStore{
		sessions: make(map[string]*guestSession),
	}
}

func (s *memoryGuestStore) cleanupLocked() {
	now := nowFunc()
	for id, sess := range s.sessions {
		if now.Sub(sess.lastAccess) > guestSessionTTL {
			delete(s.sessions, id)
		}
	}
}

func (s *memoryGuestStore) StorePaper(sessionID string, paper *Paper) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.cleanupLocked()

	sess, ok := s.sessions[sessionID]
	if !ok {
		sess = &guestSession{
			papers:     make([]*Paper, 0),
			lastAccess: nowFunc(),
		}
		s.sessions[sessionID] = sess
	}
	sess.lastAccess = nowFunc()

	for _, p := range sess.papers {
		if p.ID == paper.ID {
			return ErrDuplicatePaper
		}
	}
	if len(sess.papers) >= guestPaperRegisterLimit {
		return fmt.Errorf("ゲストは最大 %d 件まで論文を登録できます", guestPaperRegisterLimit)
	}

	paperCopy := *paper
	paperCopy.IsOwnedByMe = true
	sess.papers = append(sess.papers, &paperCopy)
	return nil
}

func (s *memoryGuestStore) GetPapers(sessionID string, offset, limit int) []Paper {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.cleanupLocked()

	sess, ok := s.sessions[sessionID]
	if !ok {
		return []Paper{}
	}
	sess.lastAccess = nowFunc()

	total := len(sess.papers)
	if offset >= total {
		return []Paper{}
	}
	end := offset + limit
	if end > total {
		end = total
	}

	out := make([]Paper, 0, end-offset)
	for i := offset; i < end; i++ {
		paper := *sess.papers[i]
		paper.IsOwnedByMe = true
		out = append(out, paper)
	}
	return out
}

func (s *memoryGuestStore) CountPapers(sessionID string) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.cleanupLocked()

	sess, ok := s.sessions[sessionID]
	if !ok {
		return 0
	}
	sess.lastAccess = nowFunc()
	return len(sess.papers)
}

func (s *memoryGuestStore) DeletePaper(sessionID, paperID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.cleanupLocked()

	sess, ok := s.sessions[sessionID]
	if !ok {
		return fmt.Errorf("論文が見つかりません: %s", paperID)
	}
	sess.lastAccess = nowFunc()

	for i, p := range sess.papers {
		if p.ID == paperID {
			sess.papers = append(sess.papers[:i], sess.papers[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("論文が見つかりません: %s", paperID)
}

func (s *memoryGuestStore) SearchPapers(sessionID, query string) []Paper {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.cleanupLocked()

	sess, ok := s.sessions[sessionID]
	if !ok {
		return []Paper{}
	}
	sess.lastAccess = nowFunc()

	q := strings.ToLower(query)
	out := make([]Paper, 0)
	for _, p := range sess.papers {
		if strings.Contains(strings.ToLower(p.Title), q) ||
			strings.Contains(strings.ToLower(p.Abstract), q) {
			paper := *p
			paper.IsOwnedByMe = true
			out = append(out, paper)
		}
	}
	return out
}

func (s *memoryGuestStore) Exists(sessionID, paperID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.cleanupLocked()

	sess, ok := s.sessions[sessionID]
	if !ok {
		return false
	}
	sess.lastAccess = nowFunc()
	for _, p := range sess.papers {
		if p.ID == paperID {
			return true
		}
	}
	return false
}

// Ensure memoryGuestStore satisfies GuestStore
var _ GuestStore = (*memoryGuestStore)(nil)
