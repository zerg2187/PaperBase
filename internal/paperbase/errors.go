package paperbase

import "errors"

// ErrDuplicatePaper は論文IDが既に存在する場合のエラー
var ErrDuplicatePaper = errors.New("論文IDが既に存在します")

// ErrDuplicateTag はタグ名が既に存在する場合のエラー
var ErrDuplicateTag = errors.New("タグ名が既に存在します")
