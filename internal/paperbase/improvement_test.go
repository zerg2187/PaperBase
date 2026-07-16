package paperbase

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubDB struct {
	upsertPaperFunc                        func(context.Context, *Paper) error
	searchPapersFunc                       func(context.Context, string, string) ([]Paper, error)
	searchPapersSemanticFunc               func(context.Context, []float32, int) ([]Paper, error)
	searchPapersSemanticWithSimilarityFunc func(context.Context, []float32, int) ([]PaperWithSimilarity, error)
	deletePaperFunc                        func(context.Context, string) error
	deletePapersFunc                       func(context.Context, []string) error
	getPapersPaginatedFunc                 func(context.Context, int, int) ([]Paper, error)
	getPapersByTagFunc                     func(context.Context, int, int, int) ([]Paper, error)
	getAllTagsFunc                         func(context.Context) ([]Tag, error)
	createTagFunc                          func(context.Context, string, string) (*Tag, error)
	updateTagFunc                          func(context.Context, int, string, string) error
	deleteTagFunc                          func(context.Context, int) error
	getPaperTagsFunc                       func(context.Context, string) ([]Tag, error)
	setPaperTagsFunc                       func(context.Context, string, []int) error
	logOperationFunc                       func(context.Context, *AuthInfo, string, string, map[string]interface{}, string, string) error
	searchGuestPapersFunc                  func(context.Context, string, string) ([]Paper, error)
	closeFunc                              func() error

	guestMu     sync.Mutex
	guestStore  map[string]map[string]Paper // sessionID -> paperID -> Paper
}

func (s *stubDB) UpsertPaper(ctx context.Context, paper *Paper) error {
	if s.upsertPaperFunc != nil {
		return s.upsertPaperFunc(ctx, paper)
	}
	return nil
}

func (s *stubDB) SearchPapers(ctx context.Context, query string, mode string) ([]Paper, error) {
	if s.searchPapersFunc != nil {
		return s.searchPapersFunc(ctx, query, mode)
	}
	return nil, nil
}

func (s *stubDB) SearchPapersSemantic(ctx context.Context, queryVector []float32, limit int) ([]Paper, error) {
	if s.searchPapersSemanticFunc != nil {
		return s.searchPapersSemanticFunc(ctx, queryVector, limit)
	}
	return nil, nil
}

func (s *stubDB) SearchPapersSemanticWithSimilarity(ctx context.Context, queryVector []float32, limit int) ([]PaperWithSimilarity, error) {
	if s.searchPapersSemanticWithSimilarityFunc != nil {
		return s.searchPapersSemanticWithSimilarityFunc(ctx, queryVector, limit)
	}
	return nil, nil
}

func (s *stubDB) DeletePaper(ctx context.Context, id string) error {
	if s.deletePaperFunc != nil {
		return s.deletePaperFunc(ctx, id)
	}
	return nil
}

func (s *stubDB) DeletePapers(ctx context.Context, ids []string) error {
	if s.deletePapersFunc != nil {
		return s.deletePapersFunc(ctx, ids)
	}
	return nil
}

func (s *stubDB) GetPapersPaginated(ctx context.Context, offset int, limit int) ([]Paper, error) {
	if s.getPapersPaginatedFunc != nil {
		return s.getPapersPaginatedFunc(ctx, offset, limit)
	}
	return nil, nil
}

func (s *stubDB) GetPapersByTag(ctx context.Context, tagID int, offset int, limit int) ([]Paper, error) {
	if s.getPapersByTagFunc != nil {
		return s.getPapersByTagFunc(ctx, tagID, offset, limit)
	}
	return nil, nil
}

func (s *stubDB) GetAllTags(ctx context.Context) ([]Tag, error) {
	if s.getAllTagsFunc != nil {
		return s.getAllTagsFunc(ctx)
	}
	return nil, nil
}

func (s *stubDB) CreateTag(ctx context.Context, name string, color string) (*Tag, error) {
	if s.createTagFunc != nil {
		return s.createTagFunc(ctx, name, color)
	}
	return &Tag{ID: 1, Name: name, Color: color}, nil
}

func (s *stubDB) UpdateTag(ctx context.Context, id int, name string, color string) error {
	if s.updateTagFunc != nil {
		return s.updateTagFunc(ctx, id, name, color)
	}
	return nil
}

func (s *stubDB) DeleteTag(ctx context.Context, id int) error {
	if s.deleteTagFunc != nil {
		return s.deleteTagFunc(ctx, id)
	}
	return nil
}

func (s *stubDB) GetPaperTags(ctx context.Context, paperID string) ([]Tag, error) {
	if s.getPaperTagsFunc != nil {
		return s.getPaperTagsFunc(ctx, paperID)
	}
	return nil, nil
}

func (s *stubDB) SetPaperTags(ctx context.Context, paperID string, tagIDs []int) error {
	if s.setPaperTagsFunc != nil {
		return s.setPaperTagsFunc(ctx, paperID, tagIDs)
	}
	return nil
}

func (s *stubDB) LogOperation(ctx context.Context, info *AuthInfo, action, target string, details map[string]interface{}, ip, ua string) error {
	if s.logOperationFunc != nil {
		return s.logOperationFunc(ctx, info, action, target, details, ip, ua)
	}
	return nil
}

func (s *stubDB) Close() error {
	if s.closeFunc != nil {
		return s.closeFunc()
	}
	return nil
}

func (s *stubDB) StoreGuestPaper(ctx context.Context, sessionID string, paper *Paper) error {
	s.guestMu.Lock()
	defer s.guestMu.Unlock()
	if s.guestStore == nil {
		s.guestStore = make(map[string]map[string]Paper)
	}
	if _, ok := s.guestStore[sessionID]; !ok {
		s.guestStore[sessionID] = make(map[string]Paper)
	}
	if _, exists := s.guestStore[sessionID][paper.ID]; exists {
		return ErrDuplicatePaper
	}
	s.guestStore[sessionID][paper.ID] = *paper
	return nil
}

func (s *stubDB) GetGuestPapers(ctx context.Context, sessionID string, offset int, limit int) ([]Paper, error) {
	s.guestMu.Lock()
	defer s.guestMu.Unlock()
	if s.guestStore == nil {
		return nil, nil
	}
	sessMap, ok := s.guestStore[sessionID]
	if !ok {
		return nil, nil
	}
	papers := make([]Paper, 0, len(sessMap))
	for _, p := range sessMap {
		p.IsOwnedByMe = true
		papers = append(papers, p)
	}
	return papers, nil
}

func (s *stubDB) GetGuestPaperCount(ctx context.Context, sessionID string) (int, error) {
	s.guestMu.Lock()
	defer s.guestMu.Unlock()
	if s.guestStore == nil {
		return 0, nil
	}
	if sessMap, ok := s.guestStore[sessionID]; ok {
		return len(sessMap), nil
	}
	return 0, nil
}

func (s *stubDB) DeleteGuestPaper(ctx context.Context, sessionID string, paperID string) error {
	s.guestMu.Lock()
	defer s.guestMu.Unlock()
	if s.guestStore == nil {
		return nil
	}
	if sessMap, ok := s.guestStore[sessionID]; ok {
		delete(sessMap, paperID)
	}
	return nil
}

func (s *stubDB) GuestPaperExists(ctx context.Context, sessionID string, paperID string) (bool, error) {
	s.guestMu.Lock()
	defer s.guestMu.Unlock()
	if s.guestStore == nil {
		return false, nil
	}
	if sessMap, ok := s.guestStore[sessionID]; ok {
		_, exists := sessMap[paperID]
		return exists, nil
	}
	return false, nil
}

func (s *stubDB) SearchGuestPapers(ctx context.Context, sessionID string, query string) ([]Paper, error) {
	if s.searchGuestPapersFunc != nil {
		return s.searchGuestPapersFunc(ctx, sessionID, query)
	}
	s.guestMu.Lock()
	defer s.guestMu.Unlock()
	if s.guestStore == nil {
		return nil, nil
	}
	sessMap, ok := s.guestStore[sessionID]
	if !ok {
		return nil, nil
	}
	papers := make([]Paper, 0)
	for _, p := range sessMap {
		p.IsOwnedByMe = true
		papers = append(papers, p)
	}
	return papers, nil
}

func (s *stubDB) SearchGuestPapersSemantic(ctx context.Context, sessionID string, queryVector []float32, limit int) ([]PaperWithSimilarity, error) {
	s.guestMu.Lock()
	defer s.guestMu.Unlock()
	sessMap, ok := s.guestStore[sessionID]
	if !ok {
		return nil, nil
	}
	results := make([]PaperWithSimilarity, 0, len(sessMap))
	for _, p := range sessMap {
		results = append(results, PaperWithSimilarity{
			ID:          p.ID,
			Title:       p.Title,
			Authors:     p.Authors,
			Abstract:    p.Abstract,
			Venue:       p.Venue,
			Year:        p.Year,
			BibTeX:      p.BibTeX,
			Similarity:  1,
			Tags:        []Tag{},
			IsOwnedByMe: true,
		})
	}
	return results, nil
}

func (s *stubDB) CleanupOldGuestPapers(ctx context.Context, olderThan time.Duration) (int, error) {
	return 0, nil
}

func newMockPipelineHandlers() *Handlers {
	handlers := NewHandlers(Config{}, "test-token")
	handlers.paperService = NewPaperService(
		&MockArxivClient{},
		&MockSemanticScholarClient{},
		&MockCrossrefClient{},
		&MockDataCiteClient{},
		&MockGeminiClient{},
		handlers.db,
	)
	return handlers
}

func TestGuestCannotSeeDatabasePapers(t *testing.T) {
	var listCalls int
	var searchCalls int
	handlers := newMockPipelineHandlers()
	handlers.db = &stubDB{
		getPapersPaginatedFunc: func(context.Context, int, int) ([]Paper, error) {
			listCalls++
			return []Paper{{ID: "db-paper", Title: "DB Paper"}}, nil
		},
		searchPapersFunc: func(context.Context, string, string) ([]Paper, error) {
			searchCalls++
			return []Paper{{ID: "db-paper", Title: "DB Paper"}}, nil
		},
	}

	listReq, err := http.NewRequest(http.MethodGet, "/api/papers", nil)
	require.NoError(t, err)
	listReq = listReq.WithContext(MockAuthContext(listReq.Context(), false, "guest-session"))

	listRR := httptest.NewRecorder()
	handlers.GetPapersPaginated(listRR, listReq)
	require.Equal(t, http.StatusOK, listRR.Code)

	var listResults []SearchResult
	require.NoError(t, json.NewDecoder(listRR.Body).Decode(&listResults))
	assert.Empty(t, listResults)
	assert.Equal(t, 0, listCalls)

	searchReq, err := http.NewRequest(http.MethodGet, "/api/search?q=paper&mode=keyword", nil)
	require.NoError(t, err)
	searchReq = searchReq.WithContext(MockAuthContext(searchReq.Context(), false, "guest-session"))

	searchRR := httptest.NewRecorder()
	handlers.SearchPapers(searchRR, searchReq)
	require.Equal(t, http.StatusOK, searchRR.Code)

	var searchResults []SearchResult
	require.NoError(t, json.NewDecoder(searchRR.Body).Decode(&searchResults))
	assert.Empty(t, searchResults)
	assert.Equal(t, 0, searchCalls)
}

func TestGuestRegisterPaperRejectsDuplicateID(t *testing.T) {
	handlers := newMockPipelineHandlers()
	if handlers.db != nil {
		t.Skip("Skipping test with real DB - requires proper cleanup between test runs")
	}
	handlers.db = &stubDB{}

	req1, err := http.NewRequest(http.MethodPost, "/api/papers", strings.NewReader(`{"arxiv_id":"2406.11717"}`))
	require.NoError(t, err)
	req1 = req1.WithContext(MockAuthContext(req1.Context(), false, "guest-session"))

	rr1 := httptest.NewRecorder()
	handlers.RegisterPaper(rr1, req1)
	require.Equal(t, http.StatusCreated, rr1.Code, rr1.Body.String())

	req2, err := http.NewRequest(http.MethodPost, "/api/papers", strings.NewReader(`{"arxiv_id":"arxiv:2406.11717v2"}`))
	require.NoError(t, err)
	req2 = req2.WithContext(MockAuthContext(req2.Context(), false, "guest-session"))

	rr2 := httptest.NewRecorder()
	handlers.RegisterPaper(rr2, req2)
	assert.Equal(t, http.StatusConflict, rr2.Code)
	assert.Contains(t, rr2.Body.String(), "論文IDが既に存在します")
}

func TestGuestPapersAreMarkedOwned(t *testing.T) {
	handlers := newMockPipelineHandlers()
	if handlers.db != nil {
		t.Skip("Skipping test with real DB - requires proper cleanup between test runs")
	}
	handlers.db = &stubDB{}

	registerReq, err := http.NewRequest(http.MethodPost, "/api/papers", strings.NewReader(`{"arxiv_id":"2406.11717"}`))
	require.NoError(t, err)
	registerReq = registerReq.WithContext(MockAuthContext(registerReq.Context(), false, "guest-session"))

	registerRR := httptest.NewRecorder()
	handlers.RegisterPaper(registerRR, registerReq)
	require.Equal(t, http.StatusCreated, registerRR.Code, registerRR.Body.String())

	listReq, err := http.NewRequest(http.MethodGet, "/api/papers", nil)
	require.NoError(t, err)
	listReq = listReq.WithContext(MockAuthContext(listReq.Context(), false, "guest-session"))

	listRR := httptest.NewRecorder()
	handlers.GetPapersPaginated(listRR, listReq)
	require.Equal(t, http.StatusOK, listRR.Code)

	var results []SearchResult
	require.NoError(t, json.NewDecoder(listRR.Body).Decode(&results))
	require.Len(t, results, 1)
	assert.True(t, results[0].IsOwnedByMe)
}

func TestCreateTagRejectsDuplicateName(t *testing.T) {
	handlers := NewHandlers(Config{}, "test-token")
	handlers.db = &stubDB{
		createTagFunc: func(context.Context, string, string) (*Tag, error) {
			return nil, ErrDuplicateTag
		},
	}

	req, err := http.NewRequest(http.MethodPost, "/api/tags", strings.NewReader(`{"name":"LLM","color":"#6366f1"}`))
	require.NoError(t, err)
	req = req.WithContext(MockAuthContext(req.Context(), true, "admin-session"))

	rr := httptest.NewRecorder()
	handlers.CreateTag(rr, req)

	assert.Equal(t, http.StatusConflict, rr.Code)
	assert.Contains(t, rr.Body.String(), "タグ名が既に存在します")
}

func TestUpdateTagRejectsDuplicateName(t *testing.T) {
	handlers := NewHandlers(Config{}, "test-token")
	handlers.db = &stubDB{
		updateTagFunc: func(context.Context, int, string, string) error {
			return ErrDuplicateTag
		},
	}

	req, err := http.NewRequest(http.MethodPut, "/api/tags/1", strings.NewReader(`{"name":"LLM","color":"#6366f1"}`))
	require.NoError(t, err)
	req = req.WithContext(MockAuthContext(req.Context(), true, "admin-session"))

	rr := httptest.NewRecorder()
	handlers.UpdateTag(rr, req)

	assert.Equal(t, http.StatusConflict, rr.Code)
	assert.Contains(t, rr.Body.String(), "タグ名が既に存在します")
}

func TestRegisterPaperAdminUpsertSucceeds(t *testing.T) {
	// 既存IDの再登録は UPSERT（メタデータ更新）として成功する
	var upsertCalls int
	handlers := newMockPipelineHandlers()
	handlers.db = &stubDB{
		upsertPaperFunc: func(context.Context, *Paper) error {
			upsertCalls++
			return nil
		},
	}
	handlers.paperService.DB = handlers.db

	req, err := http.NewRequest(http.MethodPost, "/api/papers", strings.NewReader(`{"arxiv_id":"2406.11717"}`))
	require.NoError(t, err)
	req = req.WithContext(MockAuthContext(req.Context(), true, "admin-session"))

	rr := httptest.NewRecorder()
	handlers.RegisterPaper(rr, req)

	assert.Equal(t, http.StatusCreated, rr.Code)
	assert.Equal(t, 1, upsertCalls)
}

func TestRegisterPaperAdminDBErrorReturns500(t *testing.T) {
	handlers := newMockPipelineHandlers()
	handlers.db = &stubDB{
		upsertPaperFunc: func(context.Context, *Paper) error {
			return errors.New("db error")
		},
	}
	handlers.paperService.DB = handlers.db

	req, err := http.NewRequest(http.MethodPost, "/api/papers", strings.NewReader(`{"arxiv_id":"2406.11717"}`))
	require.NoError(t, err)
	req = req.WithContext(MockAuthContext(req.Context(), true, "admin-session"))

	rr := httptest.NewRecorder()
	handlers.RegisterPaper(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}

func TestGuestKeywordSearchDBErrorReturns500(t *testing.T) {
	handlers := newMockPipelineHandlers()
	handlers.db = &stubDB{
		searchGuestPapersFunc: func(context.Context, string, string) ([]Paper, error) {
			return nil, errors.New("db error")
		},
	}

	req, err := http.NewRequest(http.MethodGet, "/api/search?q=transformer&mode=keyword", nil)
	require.NoError(t, err)
	req = req.WithContext(MockAuthContext(req.Context(), false, "guest-session"))

	rr := httptest.NewRecorder()
	handlers.SearchPapers(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}

func TestGuestDeletePaperNoDBReturns503(t *testing.T) {
	// DB 未接続時にゲスト削除が panic せず 503 を返すこと
	handlers := NewHandlers(Config{}, "test-token")

	req, err := http.NewRequest(http.MethodDelete, "/api/papers/2406.11717", nil)
	require.NoError(t, err)
	req = req.WithContext(MockAuthContext(req.Context(), false, "guest-session"))

	rr := httptest.NewRecorder()
	handlers.DeletePaper(rr, req)

	assert.Equal(t, http.StatusServiceUnavailable, rr.Code)
}

func TestUniquePositiveInts(t *testing.T) {
	assert.Equal(t, []int{3, 1, 2}, uniquePositiveInts([]int{3, 1, 3, 0, -1, 2, 1}))
}

var _ DatabaseClient = (*stubDB)(nil)
var _ io.Closer = (*stubDB)(nil)
