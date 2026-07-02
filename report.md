Summary of the failure
- Two tests failed: TestGuestRegisterPaperRejectsDuplicateID and TestGuestPapersAreMarkedOwned.
- The test expected the second guest registration of the same arXiv paper (with a versioned variant: "2406.11717" and "arxiv:2406.11717v2") to be rejected with HTTP 409 and the message "論文IDが既に存在します", but the handler returned 201 Created instead.
- Logs show duplicate-related errors in other places, but the API response in tests was not returning conflict. This indicates duplicate-detection is not being applied (or is applied to a different canonical form) for guest registrations. The root causes are:
  1. IDs are not being normalized/canonicalized before existence checks (so "2406.11717" ≠ "arxiv:2406.11717v2").
  2. The guest-path duplicate check is missing (or bypassed) in the code path used in tests (guest storage / in-memory fallback needs to be checked/implemented).

Fix approach
1. Canonicalize paper IDs (especially arXiv IDs) early — strip "arxiv:" prefix and trailing version suffixes like "v2".
2. Ensure RegisterPaper performs an existence check against both persistent DB and the guest-store (in-memory fallback) before storing/processing the paper.
3. When a duplicate is detected, return HTTP 409 with the expected Japanese message "論文IDが既に存在します".
4. For tests that run without a real DB (the CI tests use that path), provide an in-memory guest store (within the PaperService or Handlers) used when the DB is nil so guest duplicate checks work in tests.
5. Add/adjust unit-test-compatible stub implementations if needed.

Concrete code suggestions

1) Add ID normalization helper (new file or in the relevant file where RegisterPaper lives):
```go
import (
    "regexp"
    "strings"
)

var arxivVersionRe = regexp.MustCompile(`(?i)\bv\d+$`)

func normalizeArxivID(raw string) string {
    id := strings.TrimSpace(strings.ToLower(raw))
    id = strings.TrimPrefix(id, "arxiv:")
    id = strings.TrimSpace(id)
    // remove trailing version e.g. "2406.11717v2" -> "2406.11717"
    id = arxivVersionRe.ReplaceAllString(id, "")
    return id
}
```

2) Ensure RegisterPaper uses the normalized ID and checks existence before storing/processing (illustrative snippet to integrate into your handler):
```go
// parse body into req struct; assume req.ArxivID exists
paperID := normalizeArxivID(req.ArxivID)

// get session id from context (guest calls use session id)
sessionID := GetSessionIDFromContext(r.Context()) // adapt to actual helper

// 1) check DB-level duplicate if DB exists
if handlers.db != nil {
    // Prefer a dedicated DB method if available:
    exists, err := handlers.db.GuestPaperExists(r.Context(), sessionID, paperID)
    if err != nil {
        // log and return 500
    }
    if exists {
        http.Error(w, `{"status":"error","message":"論文IDが既に存在します"}`, http.StatusConflict)
        return
    }
} else {
    // 2) check in-memory guest store fallback (see next snippet for in-memory store impl)
    if handlers.paperService != nil && handlers.paperService.GuestPaperExists(sessionID, paperID) {
        http.Error(w, `{"status":"error","message":"論文IDが既に存在します"}`, http.StatusConflict)
        return
    }
}

// proceed with registration using canonical paperID
req.ArxivID = paperID
// ... rest of the pipeline (fetch metadata, vectorize, store)
```

3) Add an in-memory guest store inside PaperService (or Handlers) used when DB is nil — example fields and methods:
```go
// inside PaperService struct (or Handlers)
type PaperService struct {
    // existing fields...
    guestMu     sync.Mutex
    guestStore  map[string]map[string]Paper // sessionID -> (paperID -> Paper)
}

// initialize in NewPaperService:
func NewPaperService(/*...*/) *PaperService {
    ps := &PaperService{ /* existing init */ }
    ps.guestStore = make(map[string]map[string]Paper)
    return ps
}

// store guest paper
func (s *PaperService) StoreGuestPaper(sessionID string, p *Paper) error {
    s.guestMu.Lock()
    defer s.guestMu.Unlock()
    if _, ok := s.guestStore[sessionID]; !ok {
        s.guestStore[sessionID] = make(map[string]Paper)
    }
    if _, exists := s.guestStore[sessionID][p.ID]; exists {
        return ErrDuplicatePaper // or a dedicated error
    }
    s.guestStore[sessionID][p.ID] = *p
    return nil
}

func (s *PaperService) GuestPaperExists(sessionID, paperID string) bool {
    s.guestMu.Lock()
    defer s.guestMu.Unlock()
    if sessMap, ok := s.guestStore[sessionID]; ok {
        _, exists := sessMap[paperID]
        return exists
    }
    return false
}

func (s *PaperService) GetGuestPapers(sessionID string, offset, limit int) []Paper {
    s.guestMu.Lock()
    defer s.guestMu.Unlock()
    out := []Paper{}
    if sessMap, ok := s.guestStore[sessionID]; ok {
        for _, p := range sessMap {
            out = append(out, p)
        }
    }
    // apply pagination if needed
    return out
}
```

4) Map DB errors to the correct HTTP response text (register handler):
```go
if err != nil {
    if errors.Is(err, ErrDuplicatePaper) {
        http.Error(w, `{"status":"error","message":"論文IDが既に存在します"}`, http.StatusConflict)
        return
    }
    // other errors -> 500
}
```

5) Tests adjustments (if needed)
- If any tests rely on the stubDB's guest methods, ensure stubDB implements GuestPaperExists / StoreGuestPaper / GetGuestPapers in a way that simulates real behavior for guest flows. However, do not change newMockPipelineHandlers to attach a non-nil DB there — tests check handlers.db != nil to decide whether to skip (they intend to skip only when a "real DB" is attached). Instead, the in-memory guest store above will handle the no-DB path used by tests.

Example minimal stub methods (if you prefer to implement them inside improvement_test.go rather than production code):
```go
// in the test stubDB struct (for tests that explicitly set handlers.db)
guestStore map[string]map[string]Paper
guestMu    sync.Mutex

func (s *stubDB) StoreGuestPaper(ctx context.Context, sessionID string, paper *Paper) error {
    s.guestMu.Lock()
    defer s.guestMu.Unlock()
    if s.guestStore == nil {
        s.guestStore = map[string]map[string]Paper{}
    }
    if _, ok := s.guestStore[sessionID]; !ok {
        s.guestStore[sessionID] = map[string]Paper{}
    }
    if _, exists := s.guestStore[sessionID][paper.ID]; exists {
        return ErrDuplicatePaper
    }
    s.guestStore[sessionID][paper.ID] = *paper
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

func (s *stubDB) GetGuestPapers(ctx context.Context, sessionID string, offset int, limit int) ([]Paper, error) {
    s.guestMu.Lock()
    defer s.guestMu.Unlock()
    out := []Paper{}
    if s.guestStore == nil {
        return out, nil
    }
    if sessMap, ok := s.guestStore[sessionID]; ok {
        for _, p := range sessMap {
            out = append(out, p)
        }
    }
    return out, nil
}
```

Why this will fix the tests
- Normalizing arXiv IDs ensures "2406.11717" and "arxiv:2406.11717v2" canonicalize to the same key and will be caught by existence checks.
- Checking both DB and guest-store prevents creating duplicate guest entries when DB is nil (test environment).
- Returning the expected HTTP 409 with the exact Japanese message satisfies the assertions in the tests.
- Marking guest-owned papers: when GetPapersPaginated/handler merges DB results with guest-store results, the test will see owned papers and IsOwnedByMe will be set.

Suggested next steps
1. Apply the normalization and duplicate-check changes to the RegisterPaper handler and PaperService (or Handlers).
2. Add the in-memory guest-store implementation in PaperService (and/or the test stub if tests explicitly set handlers.db).
3. Run tests locally: go test ./... to ensure the two failing tests pass.
4. Commit and push changes; re-run CI.

If you want, I can:
- Produce a patch/PR with the exact file diffs (where RegisterPaper and PaperService live), or
- Inspect the exact handler and service files in the repo to produce line-precise edits.
