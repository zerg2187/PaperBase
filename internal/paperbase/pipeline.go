package paperbase

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/lib/pq"
)

// =============================================================================
// データ構造体の定義
// =============================================================================

// 1. arXiv API用
type ArxivFeed struct {
	Entries []ArxivEntry `xml:"entry"`
}
type ArxivEntry struct {
	Title   string `xml:"title"`
	Summary string `xml:"summary"`
	Authors []ArxivAuthor `xml:"author"`
}
type ArxivAuthor struct {
	Name string `xml:"name"`
}
// 2. Semantic Scholar API用
type S2ExternalIds struct {
	DOI string `json:"DOI"`
}

type S2Venue struct {
	Id   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
	Url  string `json:"url"`
	Issn string `json:"issn"`
}

type S2Journal struct {
	Name   string `json:"name"`
	Volume string `json:"volume"`
	Issue  string `json:"issue"`
	Pages  string `json:"pages"`
}

type S2Response struct {
	ExternalIds      S2ExternalIds `json:"externalIds"`
	Venue            string         `json:"venue"`
	Year             int            `json:"year"`
	Journal          *S2Journal     `json:"journal"`          // ジャーナル情報（ポインタでnullを許容）
	PublicationVenue S2Venue       `json:"publicationVenue"` // 会議/ジャーナル詳細情報
	CitationStyles   struct {
		Bibtex string `json:"bibtex"`
	} `json:"citationStyles"`
}

// 3. Crossref API用
type CrossrefRequest struct {
	 CrossRefType string `json:"crossref_type"`
	 DOI          string `json:"doi"`
	 Component    string `json:"component"`
	 Format       string `json:"format"`
	 Style        string `json:"style"`
	 Locale       string `json:"locale"`
}

type CrossrefResponse struct {
	 Title       []string `json:"title"`
	 ContainerTitle []string `json:"container-title"`
	 Publisher   string   `json:"publisher"`
	 Volume      string   `json:"volume"`
	 Issue       string   `json:"issue"`
	 Page        string   `json:"page"`
	 PublishedPrint struct {
		  DateParts [][]int `json:"date-parts"`
	 } `json:"published-print"`
	 PublishedOnline struct {
		  DateParts [][]int `json:"date-parts"`
	 } `json:"published-online"`
}

// 4. DataCite API用
type DataCiteResponse struct {
	Data struct {
		Attributes struct {
			Titles          []struct { Title string `json:"title"` } `json:"titles"`
			ContainerTitle  string `json:"container-title"`
			Publisher       string `json:"publisher"`
			Volume          string `json:"volume"`
			Issue           string `json:"issue"`
			Page            string `json:"page"`
			Published       string `json:"published"`
		} `json:"attributes"`
	} `json:"data"`
}

// 5. Gemini API用
type GeminiRequest struct {
	Model   string `json:"model"`
	Content struct {
		Parts []struct {
			Text string `json:"text"`
		} `json:"parts"`
	} `json:"content"`
	OutputDimensionality int `json:"outputDimensionality,omitempty"`
}
type GeminiResponse struct {
	Embedding struct {
		Values []float32 `json:"values"`
	} `json:"embedding"`
}

// =============================================================================
// メイン処理
// =============================================================================

func main() {
	// プログラム全体の実行時間計測スタート
	totalStart := time.Now()
	fmt.Println("=== 論文処理パイプライン デバッグ実行開始 ===")

	// 1. 環境変数の読み込み
	if err := godotenv.Load(); err != nil {
		fmt.Println("警告: .envファイルが見つかりません。システム環境変数を使用します。")
	}

	geminiKey := os.Getenv("GEMINI_API_KEY")
	s2Key := os.Getenv("SEMANTIC_API_KEY")

	if geminiKey == "" || s2Key == "" {
		fmt.Println("エラー: 必要なAPIキーが設定されていません (.envを確認してください)")
		return
	}

	// テスト用のarXiv ID (LLMのアライメントに関する代表的論文 InstructGPT)
	paperID := "2406.11717"
	fmt.Printf("対象論文ID: %s\n\n", paperID)

	// -------------------------------------------------------------------------
	// Step 1: arXiv API からメタデータとAbstractを取得
	// -------------------------------------------------------------------------
	fmt.Println("[Step 1] arXiv API 通信中...")
	step1Start := time.Now()

	arxivURL := fmt.Sprintf("http://export.arxiv.org/api/query?id_list=%s", paperID)
	arxivResp, err := http.Get(arxivURL)
	if err != nil {
		fmt.Println("arXivリクエストエラー:", err)
		return
	}
	defer arxivResp.Body.Close()

	arxivBody, _ := io.ReadAll(arxivResp.Body)
	var feed ArxivFeed
	xml.Unmarshal(arxivBody, &feed)

	if len(feed.Entries) == 0 {
		fmt.Println("エラー: 論文が見つかりませんでした")
		return
	}

	paper := feed.Entries[0]
	title := strings.TrimSpace(strings.ReplaceAll(paper.Title, "\n", " "))
	abstract := strings.TrimSpace(strings.ReplaceAll(paper.Summary, "\n", " "))

	fmt.Printf("  -> 完了 (所要時間: %v)\n", time.Since(step1Start))

	// -------------------------------------------------------------------------
	// Step 2: Semantic Scholar API から学会情報とBibTeXを取得
	// -------------------------------------------------------------------------
	fmt.Println("[Step 2] Semantic Scholar API 通信中...")
	step2Start := time.Now()

	s2URL := fmt.Sprintf("https://api.semanticscholar.org/graph/v1/paper/ARXIV:%s?fields=externalIds,venue,citationStyles,year,journal,publicationVenue", paperID)

	var s2Data S2Response
	maxRetries := 5
	retryDelay := 1 * time.Second

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			fmt.Printf("  -> リトライ %d/%d (待機: %v)...\n", attempt, maxRetries-1, retryDelay)
			time.Sleep(retryDelay)
			retryDelay *= 2 // 指数バックオフ
		}

		s2Req, _ := http.NewRequest("GET", s2URL, nil)
		s2Req.Header.Add("x-api-key", s2Key)

		s2Client := &http.Client{}
		s2Resp, err := s2Client.Do(s2Req)
		if err != nil {
			fmt.Printf("  -> S2リクエストエラー: %v\n", err)
			continue
		}

		// レスポンスを先に読み取る
		s2Body, _ := io.ReadAll(s2Resp.Body)
		s2Resp.Body.Close()

		if s2Resp.StatusCode == http.StatusOK {
			json.Unmarshal(s2Body, &s2Data)
			// デバッグ出力：S2 APIからのデータを確認
			break // 成功時はループを抜ける
		}

		// 429 (Too Many Requests) または 5xx エラーの場合はリトライ
		if s2Resp.StatusCode == 429 || s2Resp.StatusCode >= 500 {
			fmt.Printf("  -> S2 API エラー (ステータス: %d) - リトライします\n", s2Resp.StatusCode)
			continue
		}

		// その他のエラー（4xx等）はリトライせず終了
		fmt.Printf("  -> S2 API エラー (ステータス: %d)\n", s2Resp.StatusCode)
		fmt.Printf("  -> エラー詳細: %s\n", string(s2Body))
		return
	}

	fmt.Printf("  -> 完了 (所要時間: %v)\n", time.Since(step2Start))

	// -------------------------------------------------------------------------
	// Step 2.2: タイトルで会議版の論文を検索（arXiv版が会議に採録されている場合）
	// -------------------------------------------------------------------------
	var conferencePaperId string
	if s2Data.Venue != "" && s2Data.Journal != nil && s2Data.Journal.Name == "ArXiv" {
		fmt.Println("[Step 2.2] タイトルで会議版の論文を検索中...")
		step22Start := time.Now()

		// タイトルでS2 APIを検索
		searchQuery := fmt.Sprintf("%s %s", title, s2Data.Venue)
		searchURL := fmt.Sprintf("https://api.semanticscholar.org/graph/v1/paper/search?query=%s&fields=paperId,title,externalIds,year,journal&limit=5", url.QueryEscape(searchQuery))

		searchReq, _ := http.NewRequest("GET", searchURL, nil)
		searchReq.Header.Add("x-api-key", s2Key)

		searchClient := &http.Client{}
		searchResp, err := searchClient.Do(searchReq)
		if err != nil {
			fmt.Printf("  -> 検索リクエストエラー: %v\n", err)
		} else {
			defer searchResp.Body.Close()

			if searchResp.StatusCode == http.StatusOK {
				searchBody, _ := io.ReadAll(searchResp.Body)
				var searchResult struct {
					Data []struct {
						PaperId     string         `json:"paperId"`
						Title       string         `json:"title"`
						ExternalIds S2ExternalIds `json:"externalIds"`
						Year        int            `json:"year"`
						Journal     *S2Journal     `json:"journal"`
					} `json:"data"`
				}
				json.Unmarshal(searchBody, &searchResult)
				// 会議版の論文を探す（arXiv版ではない、同じタイトルの論文）
				for _, paper := range searchResult.Data {
					if paper.Journal == nil || paper.Journal.Name != "ArXiv" {
						// 同じタイトルの会議版が見つかった
						conferencePaperId = paper.PaperId
						fmt.Printf("  -> 会議版論文を見つけました: %s (DOI: %s)\n", paper.PaperId, paper.ExternalIds.DOI)
						break
					}
				}
				if conferencePaperId == "" {
					fmt.Println("  -> 会議版論文が見つかりませんでした")
				}
			} else {
				fmt.Printf("  -> 検索API エラー (ステータス: %d)\n", searchResp.StatusCode)
			}
		}
		fmt.Printf("  -> 完了 (所要時間: %v)\n", time.Since(step22Start))
	}

	// -------------------------------------------------------------------------
	// Step 2.3: Crossrefタイトル検索で会議版を探す
	// -------------------------------------------------------------------------
	var crossrefDOI string
	if conferencePaperId == "" && s2Data.Venue != "" {
		fmt.Println("[Step 2.3] Crossrefタイトル検索で会議版を探す...")
		step23Start := time.Now()

		// タイトルでCrossref APIを検索
		crossrefSearchURL := fmt.Sprintf("https://api.crossref.org/works?query=%s&rows=5", url.QueryEscape(title))
		crossrefSearchReq, _ := http.NewRequest("GET", crossrefSearchURL, nil)
		crossrefSearchReq.Header.Set("User-Agent", "Paperbase/1.0 (mailto:your-email@example.com)")

		crossrefSearchClient := &http.Client{}
		crossrefSearchResp, err := crossrefSearchClient.Do(crossrefSearchReq)
		if err != nil {
			fmt.Printf("  -> Crossref検索エラー: %v\n", err)
		} else {
			defer crossrefSearchResp.Body.Close()

			if crossrefSearchResp.StatusCode == http.StatusOK {
				crossrefSearchBody, _ := io.ReadAll(crossrefSearchResp.Body)
				var crossrefSearchResult struct {
					Message struct {
						Items []struct {
							DOI   string   `json:"DOI"`
							Title  []string `json:"title"`
							Type   string   `json:"type"`
							ContainerTitle []string `json:"container-title"`
							Volume string   `json:"volume"`
							Issue  string   `json:"issue"`
							Page   string   `json:"page"`
						} `json:"items"`
					} `json:"message"`
				}
				json.Unmarshal(crossrefSearchBody, &crossrefSearchResult)

				// proceedings-article（会議論文）を探す
				for _, item := range crossrefSearchResult.Message.Items {
					if item.Type == "proceedings-article" && len(item.Title) > 0 && item.Title[0] == title {
						crossrefDOI = item.DOI
						fmt.Printf("  -> 会議版論文を見つけました: %s\n", item.DOI)
						if len(item.ContainerTitle) > 0 {
							fmt.Printf("      会議: %s, Pages: %s\n", item.ContainerTitle[0], item.Page)
						}
						break
					}
				}
				if crossrefDOI == "" {
					fmt.Println("  -> 会議版論文が見つかりませんでした")
				}
			} else {
				fmt.Printf("  -> Crossref検索エラー (ステータス: %d)\n", crossrefSearchResp.StatusCode)
			}
		}
		fmt.Printf("  -> 完了 (所要時間: %v)\n", time.Since(step23Start))
	}

		// DOIが取得できた場合のみ Crossref/DataCite API を呼び出す
		var crossrefData CrossrefResponse
		var dataciteData DataCiteResponse
		var officialBibtex string // オフィシャルBibTeX

		// crossrefDOI（会議版）またはarXiv版DOIを優先
		targetDOI := crossrefDOI
		if targetDOI == "" {
			targetDOI = s2Data.ExternalIds.DOI
		}

		if targetDOI != "" {
			// -------------------------------------------------------------------------
			// Step 2.5: Crossref API から正確な論文情報を取得
			// -------------------------------------------------------------------------
			fmt.Println("[Step 2.5] Crossref API 通信中...")
			step25Start := time.Now()

			// Crossref transform APIでBibTeX形式を直接取得
			doi := strings.TrimPrefix(targetDOI, "doi:")
			crossrefURL := fmt.Sprintf("https://api.crossref.org/works/%s/transform", doi)
			crossrefReq, _ := http.NewRequest("GET", crossrefURL, nil)
			crossrefReq.Header.Set("User-Agent", "Paperbase/1.0 (mailto:your-email@example.com)")
			crossrefReq.Header.Set("Accept", "application/x-bibtex")

			crossrefClient := &http.Client{}
			crossrefResp, err := crossrefClient.Do(crossrefReq)
			if err != nil {
				fmt.Printf("  -> Crossrefリクエストエラー: %v\n", err)
			} else {
				defer crossrefResp.Body.Close()

				if crossrefResp.StatusCode == http.StatusOK {
					bibtexBody, _ := io.ReadAll(crossrefResp.Body)
					officialBibtex = string(bibtexBody)
					fmt.Printf("  -> 完了 (所要時間: %v)\n", time.Since(step25Start))
				} else {
					fmt.Printf("  -> Crossref API エラー (ステータス: %d) - DataCiteを試します\n", crossrefResp.StatusCode)

					// -------------------------------------------------------------------------
					// Step 2.6: DataCite API から論文情報を取得 (arXiv DOI用)
					// -------------------------------------------------------------------------
					fmt.Println("[Step 2.6] DataCite API 通信中...")
					step26Start := time.Now()

					dataciteURL := fmt.Sprintf("https://api.datacite.org/dois/%s", strings.TrimPrefix(s2Data.ExternalIds.DOI, "doi:"))
					dataciteReq, _ := http.NewRequest("GET", dataciteURL, nil)
					dataciteReq.Header.Set("User-Agent", "Paperbase/1.0 (mailto:your-email@example.com)")
						dataciteReq.Header.Set("Accept", "application/x-bibtex")

						dataciteClient := &http.Client{}
						dataciteResp, err := dataciteClient.Do(dataciteReq)
						if err != nil {
							fmt.Printf("  -> DataCiteリクエストエラー: %v\n", err)
						} else {
							defer dataciteResp.Body.Close()

							if dataciteResp.StatusCode == http.StatusOK {
								dataciteBody, _ := io.ReadAll(dataciteResp.Body)
								officialBibtex = string(dataciteBody)
								// JSON形式も取得して表示用データを取得
								jsonURL := fmt.Sprintf("https://api.datacite.org/dois/%s", strings.TrimPrefix(s2Data.ExternalIds.DOI, "doi:"))
								jsonReq, _ := http.NewRequest("GET", jsonURL, nil)
								jsonReq.Header.Set("User-Agent", "Paperbase/1.0")
								jsonClient := &http.Client{}
								jsonResp, _ := jsonClient.Do(jsonReq)
								if jsonResp != nil && jsonResp.StatusCode == http.StatusOK {
									jsonBody, _ := io.ReadAll(jsonResp.Body)
									json.Unmarshal(jsonBody, &dataciteData)
									jsonResp.Body.Close()
								}
								fmt.Printf("  -> 完了 (所要時間: %v)\n", time.Since(step26Start))
							} else {
								fmt.Printf("  -> DataCite API エラー (ステータス: %d)\n", dataciteResp.StatusCode)
							}
						}
				}
			}
		} else {
			fmt.Println("[Step 2.5] DOIが取得できませんでした - スキップ")
		}

	// -------------------------------------------------------------------------
	// Step 3: Gemini API でAbstractをベクトル化
	// -------------------------------------------------------------------------
	fmt.Println("[Step 3] Gemini API でベクトル化中...")
	step3Start := time.Now()

	// 埋め込み用のテキストを整形（タイトル + Abstract）
	embedText := fmt.Sprintf("Title: %s\nAbstract: %s", title, abstract)

	geminiReqData := GeminiRequest{Model: "models/gemini-embedding-2", OutputDimensionality: 768}
	geminiReqData.Content.Parts = append(geminiReqData.Content.Parts, struct {
		Text string `json:"text"`
	}{Text: embedText})

	geminiJSON, _ := json.Marshal(geminiReqData)
	geminiURL := fmt.Sprintf("https://generativelanguage.googleapis.com/v1/models/gemini-embedding-2:embedContent?key=%s", geminiKey)

	geminiReq, _ := http.NewRequest("POST", geminiURL, bytes.NewBuffer(geminiJSON))
	geminiReq.Header.Set("Content-Type", "application/json")

	geminiClient := &http.Client{}
	geminiResp, err := geminiClient.Do(geminiReq)
	if err != nil {
		fmt.Println("Geminiリクエスト(通信)エラー:", err)
		return
	}
	defer geminiResp.Body.Close()

	// ---------------------------------------------------------
	// 【追加】レスポンスの中身を読み取り、ステータスコードをチェック
	// ---------------------------------------------------------
	geminiBody, _ := io.ReadAll(geminiResp.Body)

	if geminiResp.StatusCode != http.StatusOK {
		fmt.Printf("  -> [APIエラー] HTTPステータス: %d\n", geminiResp.StatusCode)
		fmt.Printf("  -> [エラー詳細]: %s\n", string(geminiBody)) // Googleからの警告文を表示
		return
	}

	var geminiData GeminiResponse
	json.Unmarshal(geminiBody, &geminiData)

	vector := geminiData.Embedding.Values

	fmt.Printf("  -> 完了 (所要時間: %v)\n", time.Since(step3Start))

	// -------------------------------------------------------------------------
	// 結果のサマリー出力
	// -------------------------------------------------------------------------
	fmt.Println("\n==================================================")
	fmt.Println(" 論文データ取得結果")
	fmt.Println("==================================================")
	fmt.Printf("【タイトル】%s\n", title)
	fmt.Printf("【DOI】%s\n", s2Data.ExternalIds.DOI)

		// Crossrefから取得した情報を優先表示
		if len(crossrefData.ContainerTitle) > 0 {
			fmt.Printf("【ジャーナル/会議(Crossref)】%s\n", crossrefData.ContainerTitle[0])
			if crossrefData.Publisher != "" {
				fmt.Printf("【出版社】%s\n", crossrefData.Publisher)
			}
			if crossrefData.Volume != "" {
				fmt.Printf("【巻(Volume)】%s\n", crossrefData.Volume)
			}
			if crossrefData.Issue != "" {
				fmt.Printf("【号(Issue)】%s\n", crossrefData.Issue)
			}
			if crossrefData.Page != "" {
				fmt.Printf("【ページ】%s\n", crossrefData.Page)
			}
			// 出版日を表示
			if len(crossrefData.PublishedOnline.DateParts) > 0 && len(crossrefData.PublishedOnline.DateParts[0]) >= 3 {
				dateParts := crossrefData.PublishedOnline.DateParts[0]
				fmt.Printf("【出版日】%d年%d月\n", dateParts[0], dateParts[1])
			} else if len(crossrefData.PublishedPrint.DateParts) > 0 && len(crossrefData.PublishedPrint.DateParts[0]) >= 3 {
				dateParts := crossrefData.PublishedPrint.DateParts[0]
				fmt.Printf("【出版日】%d年%d月\n", dateParts[0], dateParts[1])
			}
		} else if len(dataciteData.Data.Attributes.Titles) > 0 {
			// DataCiteから取得した情報を表示
			fmt.Printf("【ジャーナル/会議(DataCite)】%s\n", dataciteData.Data.Attributes.ContainerTitle)
			if dataciteData.Data.Attributes.Publisher != "" {
				fmt.Printf("【出版社】%s\n", dataciteData.Data.Attributes.Publisher)
			}
			if dataciteData.Data.Attributes.Volume != "" {
				fmt.Printf("【巻(Volume)】%s\n", dataciteData.Data.Attributes.Volume)
			}
			if dataciteData.Data.Attributes.Issue != "" {
				fmt.Printf("【号(Issue)】%s\n", dataciteData.Data.Attributes.Issue)
			}
			if dataciteData.Data.Attributes.Page != "" {
				fmt.Printf("【ページ】%s\n", dataciteData.Data.Attributes.Page)
			}
			if dataciteData.Data.Attributes.Published != "" {
				fmt.Printf("【出版日】%s\n", dataciteData.Data.Attributes.Published)
			}
		} else {
			// S2フォールバック
			venue := s2Data.Venue
			if venue == "" {
				venue = "不明 (プレプリント等)"
			}
			fmt.Printf("【学会情報(S2)】%s\n", venue)
		}

	// Abstractを安全に表示（100文字未満の場合は全体を表示）
	abstractPreview := abstract
	if len(abstract) > 100 {
		abstractPreview = abstract[:100] + "..."
	}
	fmt.Printf("【Abstract】%s\n", abstractPreview)

	// BibTeXを表示（オフィシャルBibTeXを優先、なければS2フォールバック）
	bibtex := officialBibtex
	if bibtex == "" {
		bibtex = s2Data.CitationStyles.Bibtex
	}
	if bibtex != "" {
		if officialBibtex != "" {
			fmt.Printf("【BibTeX(Official)】\n%s\n", strings.TrimSpace(bibtex))
		} else {
			// 見やすくするために改行を調整
			bibtex = strings.ReplaceAll(bibtex, "},", "},\n  ")
			fmt.Printf("【BibTeX(S2)】\n%s\n", bibtex)
		}
	}

	if len(vector) > 0 {
		fmt.Printf("\n【ベクトル】次元数: %d\n", len(vector))
		fmt.Printf("            値(先頭5件): %v\n", vector[:5])
	} else {
		fmt.Println("\n【ベクトル】取得失敗")
	}

	fmt.Println("==================================================")
	fmt.Printf("パイプライン全体の総実行時間: %v\n", time.Since(totalStart))
// -------------------------------------------------------------------------

		// venue変数をDB保存用に設定（CrossrefまたはS2フォールバック）
		var venue string
		if len(crossrefData.ContainerTitle) > 0 {
			venue = crossrefData.ContainerTitle[0]
		} else {
			venue = s2Data.Venue
			if venue == "" {
				venue = "不明 (プレプリント等)"
			}
		}
	// 【デモ用】Step 4: Supabase (PostgreSQL) への INSERT
	// ※本番では papers テーブルと chunks テーブルを分けて実装します
	// -------------------------------------------------------------------------
	fmt.Println("[Step 4] Supabaseへのデータ保存中...")
	step4Start := time.Now()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		fmt.Println("  -> スキップ: DATABASE_URL が設定されていません")
		return
	}

	// データベースに接続
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		fmt.Println("DB接続エラー:", err)
		return
	}
	defer db.Close()

	// pgvectorに保存するため、[]float32 の配列を "[0.1, 0.2, ...]" という文字列形式に変換
	var vecStrBuilder strings.Builder
	vecStrBuilder.WriteString("[")
	for i, v := range vector {
		vecStrBuilder.WriteString(fmt.Sprintf("%f", v))
		if i < len(vector)-1 {
			vecStrBuilder.WriteString(",")
		}
	}
	vecStrBuilder.WriteString("]")
	vecStr := vecStrBuilder.String()

	// 著者リストのカンマ区切り文字列を作成（簡易版）
	var authorNames []string
	for _, author := range feed.Entries[0].Authors {
		authorNames = append(authorNames, author.Name)
	}

	// SQLの実行（同じIDが来たらタイトル等を上書きする簡易的なUpsert）
	query := `
		INSERT INTO papers (id, title, authors, abstract, venue, bibtex, embedding)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (id) DO UPDATE 
		SET title = EXCLUDED.title, abstract = EXCLUDED.abstract;
	`

	_, err = db.Exec(query, paperID, title, pq.Array(authorNames), abstract, venue, bibtex, vecStr)
	if err != nil {
		fmt.Println("DB保存エラー:", err)
		// ※ もし "relation papers does not exist" と出た場合は、Supabase側でテーブルを作成してください
		return
	}

	fmt.Printf("  -> 完了 (所要時間: %v)\n", time.Since(step4Start))
}
