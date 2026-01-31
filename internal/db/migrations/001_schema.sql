-- +migrate Up
-- quotes: Core quote storage
CREATE TABLE quotes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    text TEXT NOT NULL,
    text_hash TEXT NOT NULL UNIQUE,
    source_book TEXT NOT NULL,
    chapter TEXT,
    character TEXT,
    themes TEXT NOT NULL,              -- JSON array
    modern_relevance TEXT,
    embedding BLOB,                    -- 768 float32s
    char_count INTEGER NOT NULL,
    times_posted INTEGER DEFAULT 0,
    last_posted_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_quotes_book ON quotes(source_book);
CREATE INDEX idx_quotes_char_count ON quotes(char_count);
CREATE INDEX idx_quotes_last_posted ON quotes(last_posted_at);

-- posts: Track all posted content
CREATE TABLE posts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    quote_id INTEGER NOT NULL REFERENCES quotes(id),
    platform TEXT NOT NULL,
    platform_post_id TEXT,
    post_url TEXT,
    trend_id INTEGER REFERENCES trends(id),
    trend_title TEXT NOT NULL,
    trend_source TEXT NOT NULL,
    trend_hash TEXT NOT NULL,          -- Prevent duplicate trend posts
    relevance_score REAL NOT NULL,
    relevance_reasoning TEXT,
    vector_similarity REAL NOT NULL,
    likes INTEGER DEFAULT 0,
    reposts INTEGER DEFAULT 0,
    replies INTEGER DEFAULT 0,
    posted_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_posts_platform ON posts(platform, posted_at DESC);
CREATE INDEX idx_posts_trend_hash ON posts(trend_hash);
CREATE UNIQUE INDEX idx_posts_trend_platform ON posts(trend_hash, platform);

-- trends: Detected trends
CREATE TABLE trends (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    source TEXT NOT NULL,
    external_id TEXT,
    title TEXT NOT NULL,
    url TEXT,
    description TEXT,
    score INTEGER,
    embedding BLOB,
    matched BOOLEAN DEFAULT FALSE,
    skipped BOOLEAN DEFAULT FALSE,
    skip_reason TEXT,
    detected_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX idx_trends_source_external ON trends(source, external_id);
CREATE INDEX idx_trends_unmatched ON trends(matched, skipped, detected_at DESC);

-- extraction_jobs: Track book processing
CREATE TABLE extraction_jobs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    book_title TEXT NOT NULL,
    file_path TEXT NOT NULL,
    total_chunks INTEGER,
    processed_chunks INTEGER DEFAULT 0,
    quotes_extracted INTEGER DEFAULT 0,
    status TEXT DEFAULT 'pending',
    error_message TEXT,
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- config: Runtime configuration
CREATE TABLE config (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- +migrate Down
DROP TABLE IF EXISTS config;
DROP TABLE IF EXISTS extraction_jobs;
DROP TABLE IF EXISTS trends;
DROP TABLE IF EXISTS posts;
DROP TABLE IF EXISTS quotes;
