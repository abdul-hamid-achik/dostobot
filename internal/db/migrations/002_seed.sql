-- +migrate Up
-- Seed default configuration values
INSERT INTO config (key, value) VALUES
    ('monitor_interval', '30m'),
    ('post_interval', '4h'),
    ('max_posts_per_day', '6'),
    ('min_relevance_score', '0.6'),
    ('min_vector_similarity', '0.5');

-- +migrate Down
DELETE FROM config WHERE key IN (
    'monitor_interval',
    'post_interval',
    'max_posts_per_day',
    'min_relevance_score',
    'min_vector_similarity'
);
