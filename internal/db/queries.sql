-- name: GetQuote :one
SELECT * FROM quotes WHERE id = ? LIMIT 1;

-- name: GetQuoteByHash :one
SELECT * FROM quotes WHERE text_hash = ? LIMIT 1;

-- name: ListQuotes :many
SELECT * FROM quotes ORDER BY created_at DESC LIMIT ? OFFSET ?;

-- name: ListQuotesByBook :many
SELECT * FROM quotes WHERE source_book = ? ORDER BY created_at DESC;

-- name: ListQuotesWithEmbeddings :many
SELECT * FROM quotes WHERE embedding IS NOT NULL ORDER BY id;

-- name: ListQuotesWithoutEmbeddings :many
SELECT * FROM quotes WHERE embedding IS NULL ORDER BY id LIMIT ?;

-- name: CountQuotes :one
SELECT COUNT(*) FROM quotes;

-- name: CountQuotesWithEmbeddings :one
SELECT COUNT(*) FROM quotes WHERE embedding IS NOT NULL;

-- name: CountQuotesByBook :many
SELECT source_book, COUNT(*) as count FROM quotes GROUP BY source_book;

-- name: CreateQuote :one
INSERT INTO quotes (
    text, text_hash, source_book, chapter, character,
    themes, modern_relevance, char_count
) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: UpdateQuoteEmbedding :exec
UPDATE quotes SET embedding = ? WHERE id = ?;

-- name: UpdateQuotePosted :exec
UPDATE quotes
SET times_posted = times_posted + 1, last_posted_at = CURRENT_TIMESTAMP
WHERE id = ?;

-- name: GetPost :one
SELECT * FROM posts WHERE id = ? LIMIT 1;

-- name: ListPosts :many
SELECT * FROM posts ORDER BY posted_at DESC LIMIT ? OFFSET ?;

-- name: ListPostsByPlatform :many
SELECT * FROM posts WHERE platform = ? ORDER BY posted_at DESC LIMIT ?;

-- name: CountPostsToday :one
SELECT COUNT(*) FROM posts
WHERE platform = ? AND posted_at >= date('now');

-- name: GetPostByTrendHash :one
SELECT * FROM posts WHERE trend_hash = ? AND platform = ? LIMIT 1;

-- name: CreatePost :one
INSERT INTO posts (
    quote_id, platform, platform_post_id, post_url,
    trend_id, trend_title, trend_source, trend_hash,
    relevance_score, relevance_reasoning, vector_similarity
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: UpdatePostEngagement :exec
UPDATE posts SET likes = ?, reposts = ?, replies = ? WHERE id = ?;

-- name: GetTrend :one
SELECT * FROM trends WHERE id = ? LIMIT 1;

-- name: GetTrendBySourceAndExternalID :one
SELECT * FROM trends WHERE source = ? AND external_id = ? LIMIT 1;

-- name: ListUnmatchedTrends :many
SELECT * FROM trends
WHERE matched = FALSE AND skipped = FALSE
ORDER BY detected_at DESC LIMIT ?;

-- name: CreateTrend :one
INSERT INTO trends (source, external_id, title, url, description, score)
VALUES (?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: UpdateTrendMatched :exec
UPDATE trends SET matched = TRUE WHERE id = ?;

-- name: UpdateTrendSkipped :exec
UPDATE trends SET skipped = TRUE, skip_reason = ? WHERE id = ?;

-- name: UpdateTrendEmbedding :exec
UPDATE trends SET embedding = ? WHERE id = ?;

-- name: GetExtractionJob :one
SELECT * FROM extraction_jobs WHERE id = ? LIMIT 1;

-- name: GetExtractionJobByBook :one
SELECT * FROM extraction_jobs WHERE book_title = ? ORDER BY created_at DESC LIMIT 1;

-- name: ListExtractionJobs :many
SELECT * FROM extraction_jobs ORDER BY created_at DESC;

-- name: CreateExtractionJob :one
INSERT INTO extraction_jobs (book_title, file_path, status)
VALUES (?, ?, 'pending')
RETURNING *;

-- name: UpdateExtractionJobStarted :exec
UPDATE extraction_jobs
SET status = 'running', total_chunks = ?, started_at = CURRENT_TIMESTAMP
WHERE id = ?;

-- name: UpdateExtractionJobProgress :exec
UPDATE extraction_jobs
SET processed_chunks = ?, quotes_extracted = ?
WHERE id = ?;

-- name: UpdateExtractionJobCompleted :exec
UPDATE extraction_jobs
SET status = 'completed', completed_at = CURRENT_TIMESTAMP
WHERE id = ?;

-- name: UpdateExtractionJobFailed :exec
UPDATE extraction_jobs
SET status = 'failed', error_message = ?, completed_at = CURRENT_TIMESTAMP
WHERE id = ?;

-- name: GetConfig :one
SELECT value FROM config WHERE key = ?;

-- name: SetConfig :exec
INSERT INTO config (key, value, updated_at) VALUES (?, ?, CURRENT_TIMESTAMP)
ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = CURRENT_TIMESTAMP;

-- name: ListConfig :many
SELECT * FROM config ORDER BY key;
