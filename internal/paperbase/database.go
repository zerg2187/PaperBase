package paperbase

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/lib/pq"
)

// =============================================================================
// DatabaseClient実装
// =============================================================================

type dbClientImpl struct {
	db *sql.DB
}

// NewDatabaseClient は新しいデータベースクライアントを作成する
func NewDatabaseClient(dbURL string) (DatabaseClient, error) {
	// プリペアドステートメントの問題を回避するために、接続文字列にパラメータを追加
	// binary_parameters=yes: バイナリプロトコルを有効にし、unnamed prepared statementの問題を回避
	connectorStr := dbURL
	if !strings.Contains(dbURL, "binary_parameters") {
		separator := "?"
		if strings.Contains(dbURL, "?") {
			separator = "&"
		}
		connectorStr = dbURL + separator + "binary_parameters=yes"
	}

	db, err := sql.Open("postgres", connectorStr)
	if err != nil {
		return nil, err
	}

	// 接続プールの設定 - プリペアドステートメントの問題を最小限にする
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(0) // 接続を無期限に再利用

	// 接続確認
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}

	return &dbClientImpl{db: db}, nil
}

// formatVector は []float32 を PostgreSQL vector 形式の文字列に変換する
func formatVector(v []float32) string {
	var b strings.Builder
	b.WriteString("[")
	for i, val := range v {
		fmt.Fprintf(&b, "%f", val)
		if i < len(v)-1 {
			b.WriteString(",")
		}
	}
	b.WriteString("]")
	return b.String()
}

// intYear は sql.NullInt32 を int に変換する（NULL の場合は 0）
func intYear(year sql.NullInt32) int {
	if year.Valid {
		return int(year.Int32)
	}
	return 0
}

// appendUniqueTag は重複を避けてタグを追加する
func appendUniqueTag(tags []Tag, tag Tag) []Tag {
	for _, t := range tags {
		if t.ID == tag.ID {
			return tags
		}
	}
	return append(tags, tag)
}

func (c *dbClientImpl) UpsertPaper(ctx context.Context, paper *Paper) error {
	vecStr := formatVector(paper.Embedding)

	query := `
		INSERT INTO papers (id, title, authors, abstract, venue, year, bibtex, embedding)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8);
	`

	_, err := c.db.ExecContext(ctx, query,
		paper.ID, paper.Title, pq.Array(paper.Authors),
		paper.Abstract, paper.Venue, paper.Year, paper.BibTeX, vecStr)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			return ErrDuplicatePaper
		}
		return err
	}

	return nil
}

func (c *dbClientImpl) SearchPapers(ctx context.Context, query string, mode string) ([]Paper, error) {
	// キーワード検索モード
	if mode == "keyword" {
		return c.searchByKeyword(ctx, query)
	}
	// セマンティック検索モード（デフォルト）
	// ※ セマンティック検索にはベクトルが必要なため、空結果を返す
	// 実際の検索は handlers.go で SearchPapersSemantic を使用
	return []Paper{}, fmt.Errorf("semantic search requires SearchPapersSemantic")
}

// searchByKeyword はキーワード検索を行う
func (c *dbClientImpl) searchByKeyword(ctx context.Context, query string) ([]Paper, error) {
	sqlQuery := `
		SELECT id, title, authors, abstract, venue, year, bibtex
		FROM papers
		WHERE title ILIKE $1 OR abstract ILIKE $1
		ORDER BY id
		LIMIT 50;
	`

	searchPattern := "%" + query + "%"

	rows, err := c.db.QueryContext(ctx, sqlQuery, searchPattern)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var papers []Paper
	for rows.Next() {
		var p Paper
		var authors []string
		var year sql.NullInt32
		err := rows.Scan(&p.ID, &p.Title, pq.Array(&authors), &p.Abstract, &p.Venue, &year, &p.BibTeX)
		if err != nil {
			return nil, err
		}
		p.Authors = authors
		p.Year = intYear(year)
		papers = append(papers, p)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return papers, nil
}

// SearchPapersSemantic はセマンティック検索を行う（ベクトル使用）
func (c *dbClientImpl) SearchPapersSemantic(ctx context.Context, queryVector []float32, limit int) ([]Paper, error) {
	vecStr := formatVector(queryVector)

	sqlQuery := `
		SELECT id, title, authors, abstract, venue, year, bibtex,
		       1 - (embedding <=> $1::vector) as similarity
		FROM papers
		ORDER BY embedding <=> $1::vector
		LIMIT $2;
	`

	rows, err := c.db.QueryContext(ctx, sqlQuery, vecStr, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var papers []Paper
	for rows.Next() {
		var p Paper
		var authors []string
		var year sql.NullInt32
		var similarity float64
		err := rows.Scan(&p.ID, &p.Title, pq.Array(&authors), &p.Abstract, &p.Venue, &year, &p.BibTeX, &similarity)
		if err != nil {
			return nil, err
		}
		p.Authors = authors
		p.Year = intYear(year)
		// TODO: SimilarityをPaper構造体に追加または別途返す
		papers = append(papers, p)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return papers, nil
}

// PaperWithSimilarity は類似度を含む論文データ
type PaperWithSimilarity struct {
	ID          string
	Title       string
	Authors     []string
	Abstract    string
	Venue       string
	Year        int
	BibTeX      string
	Similarity  float64
	Tags        []Tag
	IsOwnedByMe bool
}

// SearchPapersSemanticWithSimilarity はセマンティック検索を行い、類似度を含む結果を返す
func (c *dbClientImpl) SearchPapersSemanticWithSimilarity(ctx context.Context, queryVector []float32, limit int) ([]PaperWithSimilarity, error) {
	vecStr := formatVector(queryVector)

	sqlQuery := `
		WITH matched_papers AS (
			SELECT p.id, 1 - (p.embedding <=> $1::vector) as similarity
			FROM papers p
			ORDER BY p.embedding <=> $1::vector
			LIMIT $2
		)
		SELECT p.id, p.title, p.authors, p.abstract, p.venue, p.year, p.bibtex,
		       mp.similarity,
		       t.id as tag_id, t.name as tag_name, t.color as tag_color
		FROM matched_papers mp
		JOIN papers p ON p.id = mp.id
		LEFT JOIN paper_tags pt ON p.id = pt.paper_id
		LEFT JOIN tags t ON pt.tag_id = t.id
		ORDER BY mp.similarity DESC;
	`

	rows, err := c.db.QueryContext(ctx, sqlQuery, vecStr, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// 論文IDをキーにしたマップで結果を収集
	papersMap := make(map[string]*PaperWithSimilarity)
	var orderedIDs []string
	for rows.Next() {
		var pID, pTitle, pAbstract, pVenue string
		var authors []string
		var year sql.NullInt32
		var bibtex sql.NullString
		var similarity float64
		var tagID sql.NullInt32
		var tagName, tagColor sql.NullString

		err := rows.Scan(&pID, &pTitle, pq.Array(&authors), &pAbstract, &pVenue, &year, &bibtex, &similarity,
			&tagID, &tagName, &tagColor)
		if err != nil {
			return nil, err
		}

		// まだ論文がマップにない場合は追加（CTEの順序を保持）
		if _, exists := papersMap[pID]; !exists {
			orderedIDs = append(orderedIDs, pID)
			papersMap[pID] = &PaperWithSimilarity{
				ID:         pID,
				Title:      pTitle,
				Authors:    authors,
				Abstract:   pAbstract,
				Venue:      pVenue,
				Year:       intYear(year),
				BibTeX:     bibtex.String,
				Similarity: similarity,
				Tags:       []Tag{},
			}
		}

		// タグがある場合は追加（重複を避ける）
		if tagID.Valid {
			papersMap[pID].Tags = appendUniqueTag(papersMap[pID].Tags, Tag{
				ID:    int(tagID.Int32),
				Name:  tagName.String,
				Color: tagColor.String,
			})
		}
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	// CTEの順序を保持してスライスに変換
	var papers []PaperWithSimilarity
	for _, id := range orderedIDs {
		papers = append(papers, *papersMap[id])
	}

	return papers, nil
}

// DeletePaper は論文を削除する
func (c *dbClientImpl) DeletePaper(ctx context.Context, id string) error {
	query := `DELETE FROM papers WHERE id = $1;`

	result, err := c.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}

	// 削除された行数を確認
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("論文が見つかりません: %s", id)
	}

	return nil
}

func (c *dbClientImpl) Close() error {
	if c.db != nil {
		return c.db.Close()
	}
	return nil
}

// LogOperation はユーザー操作ログを記録する
func (c *dbClientImpl) LogOperation(ctx context.Context, info *AuthInfo, action, target string, details map[string]interface{}, ip, ua string) error {
	if c.db == nil {
		return nil
	}

	role := "guest"
	if info.IsAdmin {
		role = "admin"
	}

	var detailsJSON []byte
	if len(details) > 0 {
		var err error
		detailsJSON, err = json.Marshal(details)
		if err != nil {
			return err
		}
	}

	_, err := c.db.ExecContext(ctx, `
		INSERT INTO operation_logs (session_id, role, action, target, details, ip_address, user_agent)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, info.SessionID, role, action, target, detailsJSON, ip, ua)
	return err
}

// =============================================================================
// タグ関連メソッド
// =============================================================================

// GetPapersPaginated はページネーション付きで全論文を取得する
func (c *dbClientImpl) GetPapersPaginated(ctx context.Context, offset int, limit int) ([]Paper, error) {
	log.Printf("GetPapersPaginated: offset=%d, limit=%d", offset, limit)

	// LIMIT/OFFSET は論文IDに対して適用し、タグJOINはその後に行う
	sqlQuery := `
		WITH page_papers AS (
			SELECT id FROM papers
			ORDER BY created_at DESC, id DESC
			LIMIT $1 OFFSET $2
		)
		SELECT p.id, p.title, p.authors, p.abstract, p.venue, p.year, p.bibtex,
		       t.id as tag_id, t.name as tag_name, t.color as tag_color
		FROM page_papers pp
		JOIN papers p ON p.id = pp.id
		LEFT JOIN paper_tags pt ON p.id = pt.paper_id
		LEFT JOIN tags t ON pt.tag_id = t.id
		ORDER BY p.created_at DESC, p.id DESC;
	`

	rows, err := c.db.QueryContext(ctx, sqlQuery, limit, offset)
	if err != nil {
		log.Printf("GetPapersPaginated query error: %v", err)
		return nil, err
	}
	defer rows.Close()

	// 論文IDをキーにしたマップで結果を収集
	papersMap := make(map[string]*Paper)
	var orderedIDs []string
	for rows.Next() {
		var pID, pTitle, pAbstract, pVenue string
		var authors []string
		var year sql.NullInt32
		var bibtex sql.NullString
		var tagID sql.NullInt32
		var tagName, tagColor sql.NullString

		err := rows.Scan(&pID, &pTitle, pq.Array(&authors), &pAbstract, &pVenue, &year, &bibtex,
			&tagID, &tagName, &tagColor)
		if err != nil {
			return nil, err
		}

		// まだ論文がマップにない場合は追加
		if _, exists := papersMap[pID]; !exists {
			papersMap[pID] = &Paper{
				ID:       pID,
				Title:    pTitle,
				Authors:  authors,
				Abstract: pAbstract,
				Venue:    pVenue,
				Year:     intYear(year),
				BibTeX:   bibtex.String,
				Tags:     []Tag{},
			}
			orderedIDs = append(orderedIDs, pID)
		}

		// タグがある場合は追加
		if tagID.Valid {
			papersMap[pID].Tags = append(papersMap[pID].Tags, Tag{
				ID:    int(tagID.Int32),
				Name:  tagName.String,
				Color: tagColor.String,
			})
		}
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	// マップをスライスに変換（SQLのORDER順を保持）
	var papers []Paper
	for _, id := range orderedIDs {
		papers = append(papers, *papersMap[id])
	}

	return papers, nil
}

// DeletePapers は複数の論文を削除する
func (c *dbClientImpl) DeletePapers(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	query := `DELETE FROM papers WHERE id = ANY($1);`
	_, err := c.db.ExecContext(ctx, query, pq.Array(ids))
	return err
}

// GetAllTags は全タグを取得する
func (c *dbClientImpl) GetAllTags(ctx context.Context) ([]Tag, error) {
	query := `SELECT id, name, color FROM tags ORDER BY name;`

	rows, err := c.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []Tag
	for rows.Next() {
		var t Tag
		err := rows.Scan(&t.ID, &t.Name, &t.Color)
		if err != nil {
			return nil, err
		}
		tags = append(tags, t)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return tags, nil
}

// CreateTag は新しいタグを作成する
func (c *dbClientImpl) CreateTag(ctx context.Context, name string, color string) (*Tag, error) {
	query := `INSERT INTO tags (name, color) VALUES ($1, $2) RETURNING id, name, color;`

	var t Tag
	err := c.db.QueryRowContext(ctx, query, name, color).Scan(&t.ID, &t.Name, &t.Color)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			return nil, ErrDuplicateTag
		}
		return nil, err
	}

	return &t, nil
}

// UpdateTag はタグを更新する
func (c *dbClientImpl) UpdateTag(ctx context.Context, id int, name string, color string) error {
	query := `UPDATE tags SET name = $1, color = $2 WHERE id = $3;`

	result, err := c.db.ExecContext(ctx, query, name, color, id)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			return ErrDuplicateTag
		}
		return err
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("タグが見つかりません: %d", id)
	}

	return nil
}

// DeleteTag はタグを削除する
func (c *dbClientImpl) DeleteTag(ctx context.Context, id int) error {
	query := `DELETE FROM tags WHERE id = $1;`

	result, err := c.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("タグが見つかりません: %d", id)
	}

	return nil
}

// GetPaperTags は論文のタグを取得する
func (c *dbClientImpl) GetPaperTags(ctx context.Context, paperID string) ([]Tag, error) {
	query := `
		SELECT t.id, t.name, t.color
		FROM tags t
		INNER JOIN paper_tags pt ON t.id = pt.tag_id
		WHERE pt.paper_id = $1
		ORDER BY t.name;
	`

	rows, err := c.db.QueryContext(ctx, query, paperID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []Tag
	for rows.Next() {
		var t Tag
		err := rows.Scan(&t.ID, &t.Name, &t.Color)
		if err != nil {
			return nil, err
		}
		tags = append(tags, t)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return tags, nil
}

// SetPaperTags は論文のタグを設定する（既存のタグを全て削除してから追加）
func (c *dbClientImpl) SetPaperTags(ctx context.Context, paperID string, tagIDs []int) error {
	// まず既存のタグを削除
	deleteQuery := `DELETE FROM paper_tags WHERE paper_id = $1;`
	_, err := c.db.ExecContext(ctx, deleteQuery, paperID)
	if err != nil {
		return err
	}

	// 新しいタグを追加
	if len(tagIDs) == 0 {
		return nil
	}
	tagIDs = uniquePositiveInts(tagIDs)
	if len(tagIDs) == 0 {
		return nil
	}

	insertQuery := `INSERT INTO paper_tags (paper_id, tag_id) VALUES `
	var values []string
	var params []interface{}
	params = append(params, paperID)

	for i, tagID := range tagIDs {
		values = append(values, fmt.Sprintf("($1, $%d)", i+2))
		params = append(params, tagID)
	}

	insertQuery += strings.Join(values, ", ") + ";"
	_, err = c.db.ExecContext(ctx, insertQuery, params...)
	return err
}

func uniquePositiveInts(values []int) []int {
	seen := make(map[int]struct{}, len(values))
	out := make([]int, 0, len(values))
	for _, value := range values {
		if value <= 0 {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

// GetPapersByTag はタグで絞り込んで論文を取得する
func (c *dbClientImpl) GetPapersByTag(ctx context.Context, tagID int, offset int, limit int) ([]Paper, error) {
	log.Printf("GetPapersByTag: tagID=%d, offset=%d, limit=%d", tagID, offset, limit)

	sqlQuery := `
		WITH page_papers AS (
			SELECT DISTINCT p.id
			FROM papers p
			INNER JOIN paper_tags pt ON p.id = pt.paper_id
			WHERE pt.tag_id = $1
			ORDER BY p.id DESC
			LIMIT $2 OFFSET $3
		)
		SELECT p.id, p.title, p.authors, p.abstract, p.venue, p.year, p.bibtex,
		       t.id as tag_id, t.name as tag_name, t.color as tag_color
		FROM page_papers pp
		JOIN papers p ON p.id = pp.id
		LEFT JOIN paper_tags pt2 ON p.id = pt2.paper_id
		LEFT JOIN tags t ON pt2.tag_id = t.id
		ORDER BY p.id DESC;
	`

	rows, err := c.db.QueryContext(ctx, sqlQuery, tagID, limit, offset)
	if err != nil {
		log.Printf("GetPapersByTag query error: %v", err)
		return nil, err
	}
	defer rows.Close()

	// 論文IDをキーにしたマップで結果を収集
	papersMap := make(map[string]*Paper)
	var orderedIDs []string
	for rows.Next() {
		var pID, pTitle, pAbstract, pVenue string
		var authors []string
		var year sql.NullInt32
		var bibtex sql.NullString
		var tagID sql.NullInt32
		var tagName, tagColor sql.NullString

		err := rows.Scan(&pID, &pTitle, pq.Array(&authors), &pAbstract, &pVenue, &year, &bibtex,
			&tagID, &tagName, &tagColor)
		if err != nil {
			return nil, err
		}

		// まだ論文がマップにない場合は追加
		if _, exists := papersMap[pID]; !exists {
			papersMap[pID] = &Paper{
				ID:       pID,
				Title:    pTitle,
				Authors:  authors,
				Abstract: pAbstract,
				Venue:    pVenue,
				Year:     intYear(year),
				BibTeX:   bibtex.String,
				Tags:     []Tag{},
			}
			orderedIDs = append(orderedIDs, pID)
		}

		if tagID.Valid {
			papersMap[pID].Tags = appendUniqueTag(papersMap[pID].Tags, Tag{
				ID:    int(tagID.Int32),
				Name:  tagName.String,
				Color: tagColor.String,
			})
		}
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	// マップをスライスに変換（SQLのORDER順を保持）
	var papers []Paper
	for _, id := range orderedIDs {
		papers = append(papers, *papersMap[id])
	}

	return papers, nil
}

// =============================================================================
// Guest papers (separate table from admin papers)
// =============================================================================

// StoreGuestPaper はゲストセッションに論文を保存する
func (c *dbClientImpl) StoreGuestPaper(ctx context.Context, sessionID string, paper *Paper) error {
	vecStr := formatVector(paper.Embedding)

	query := `
		INSERT INTO guest_papers (id, session_id, title, authors, abstract, venue, year, bibtex, embedding)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9);
	`

	_, err := c.db.ExecContext(ctx, query,
		paper.ID, sessionID, paper.Title, pq.Array(paper.Authors),
		paper.Abstract, paper.Venue, paper.Year, paper.BibTeX, vecStr)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			return ErrDuplicatePaper
		}
		return err
	}

	return nil
}

// GetGuestPapers はゲストセッションの論文をページネーション付きで取得する
func (c *dbClientImpl) GetGuestPapers(ctx context.Context, sessionID string, offset int, limit int) ([]Paper, error) {
	sqlQuery := `
		SELECT id, title, authors, abstract, venue, year, bibtex
		FROM guest_papers
		WHERE session_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3;
	`

	rows, err := c.db.QueryContext(ctx, sqlQuery, sessionID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var papers []Paper
	for rows.Next() {
		var p Paper
		var authors []string
		var year sql.NullInt32
		var bibtex sql.NullString
		err := rows.Scan(&p.ID, &p.Title, pq.Array(&authors), &p.Abstract, &p.Venue, &year, &bibtex)
		if err != nil {
			return nil, err
		}
		p.Authors = authors
		p.Year = intYear(year)
		p.BibTeX = bibtex.String
		p.IsOwnedByMe = true
		papers = append(papers, p)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return papers, nil
}

// GetGuestPaperCount はゲストセッションの論文数を取得する
func (c *dbClientImpl) GetGuestPaperCount(ctx context.Context, sessionID string) (int, error) {
	query := `SELECT COUNT(*) FROM guest_papers WHERE session_id = $1;`

	var count int
	err := c.db.QueryRowContext(ctx, query, sessionID).Scan(&count)
	if err != nil {
		return 0, err
	}

	return count, nil
}

// DeleteGuestPaper はゲストセッションから論文を削除する
func (c *dbClientImpl) DeleteGuestPaper(ctx context.Context, sessionID string, paperID string) error {
	query := `DELETE FROM guest_papers WHERE session_id = $1 AND id = $2;`

	result, err := c.db.ExecContext(ctx, query, sessionID, paperID)
	if err != nil {
		return err
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("論文が見つかりません: %s", paperID)
	}

	return nil
}

// GuestPaperExists はゲストセッションに論文が存在するか確認する
func (c *dbClientImpl) GuestPaperExists(ctx context.Context, sessionID string, paperID string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM guest_papers WHERE session_id = $1 AND id = $2);`

	var exists bool
	err := c.db.QueryRowContext(ctx, query, sessionID, paperID).Scan(&exists)
	if err != nil {
		return false, err
	}

	return exists, nil
}

// SearchGuestPapers はゲストセッション内でキーワード検索を行う
func (c *dbClientImpl) SearchGuestPapers(ctx context.Context, sessionID string, query string) ([]Paper, error) {
	sqlQuery := `
		SELECT id, title, authors, abstract, venue, year, bibtex
		FROM guest_papers
		WHERE session_id = $1 AND (title ILIKE $2 OR abstract ILIKE $2)
		ORDER BY id
		LIMIT 50;
	`

	searchPattern := "%" + query + "%"

	rows, err := c.db.QueryContext(ctx, sqlQuery, sessionID, searchPattern)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var papers []Paper
	for rows.Next() {
		var p Paper
		var authors []string
		var year sql.NullInt32
		var bibtex sql.NullString
		err := rows.Scan(&p.ID, &p.Title, pq.Array(&authors), &p.Abstract, &p.Venue, &year, &bibtex)
		if err != nil {
			return nil, err
		}
		p.Authors = authors
		p.Year = intYear(year)
		p.BibTeX = bibtex.String
		p.IsOwnedByMe = true
		papers = append(papers, p)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return papers, nil
}

// CleanupOldGuestPapers は古いゲスト論文を削除する
func (c *dbClientImpl) CleanupOldGuestPapers(ctx context.Context, olderThan time.Duration) (int, error) {
	query := `DELETE FROM guest_papers WHERE created_at < NOW() - INTERVAL '1 millisecond' * $1;`

	millis := int64(olderThan.Milliseconds())
	result, err := c.db.ExecContext(ctx, query, millis)
	if err != nil {
		return 0, err
	}

	rows, _ := result.RowsAffected()
	return int(rows), nil
}
