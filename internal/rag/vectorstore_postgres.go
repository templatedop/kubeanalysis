package rag

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

// VectorStoreType defines the type of vector store backend.
type VectorStoreType string

const (
	VectorStoreTypeMemory   VectorStoreType = "memory"
	VectorStoreTypePostgres VectorStoreType = "postgres"
)

// VectorStoreConfig holds configuration for the vector store.
type VectorStoreConfig struct {
	Type          VectorStoreType `yaml:"type" json:"type"`
	PostgresURL   string          `yaml:"postgres_url" json:"postgres_url"`
	TableName     string          `yaml:"table_name" json:"table_name"`
	Dimension     int             `yaml:"dimension" json:"dimension"`
	CreateTable   bool            `yaml:"create_table" json:"create_table"`
	MaxConns      int             `yaml:"max_conns" json:"max_conns"`
}

// DefaultVectorStoreConfig returns a default configuration.
func DefaultVectorStoreConfig() VectorStoreConfig {
	return VectorStoreConfig{
		Type:        VectorStoreTypeMemory,
		TableName:   "vector_chunks",
		Dimension:   384,
		CreateTable: true,
		MaxConns:    10,
	}
}

// NewVectorStoreFromConfig creates a vector store from configuration.
func NewVectorStoreFromConfig(cfg VectorStoreConfig) (VectorStore, error) {
	switch cfg.Type {
	case VectorStoreTypeMemory:
		return NewInMemoryVectorStore(), nil
	case VectorStoreTypePostgres:
		return NewPostgresVectorStore(cfg)
	default:
		return nil, fmt.Errorf("unsupported vector store type: %s", cfg.Type)
	}
}

// PostgresVectorStore implements VectorStore with PostgreSQL and pgvector.
type PostgresVectorStore struct {
	db        *sql.DB
	tableName string
	dimension int
}

// NewPostgresVectorStore creates a new PostgreSQL vector store.
func NewPostgresVectorStore(cfg VectorStoreConfig) (*PostgresVectorStore, error) {
	db, err := sql.Open("postgres", cfg.PostgresURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to postgres: %w", err)
	}

	db.SetMaxOpenConns(cfg.MaxConns)
	db.SetMaxIdleConns(cfg.MaxConns / 2)
	db.SetConnMaxLifetime(time.Hour)

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping postgres: %w", err)
	}

	store := &PostgresVectorStore{
		db:        db,
		tableName: cfg.TableName,
		dimension: cfg.Dimension,
	}

	if cfg.CreateTable {
		if err := store.createTable(); err != nil {
			return nil, fmt.Errorf("failed to create table: %w", err)
		}
	}

	return store, nil
}

func (vs *PostgresVectorStore) createTable() error {
	// Enable pgvector extension
	_, err := vs.db.Exec("CREATE EXTENSION IF NOT EXISTS vector")
	if err != nil {
		return fmt.Errorf("failed to create vector extension: %w", err)
	}

	// Create table
	query := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			id TEXT PRIMARY KEY,
			document_id TEXT NOT NULL,
			content TEXT NOT NULL,
			embedding vector(%d),
			metadata JSONB,
			source TEXT,
			namespace TEXT DEFAULT 'default',
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			chunk_index INTEGER DEFAULT 0
		)
	`, vs.tableName, vs.dimension)

	_, err = vs.db.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to create table: %w", err)
	}

	// Create indexes
	indexes := []string{
		fmt.Sprintf(`CREATE INDEX IF NOT EXISTS %s_namespace_idx ON %s (namespace)`, vs.tableName, vs.tableName),
		fmt.Sprintf(`CREATE INDEX IF NOT EXISTS %s_metadata_idx ON %s USING gin (metadata)`, vs.tableName, vs.tableName),
	}

	for _, idx := range indexes {
		if _, err = vs.db.Exec(idx); err != nil {
			// Non-fatal, continue
		}
	}

	return nil
}

// Upsert adds or updates chunks in the store.
func (vs *PostgresVectorStore) Upsert(ctx context.Context, chunks []Chunk) error {
	if len(chunks) == 0 {
		return nil
	}

	tx, err := vs.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	query := fmt.Sprintf(`
		INSERT INTO %s (id, document_id, content, embedding, metadata, source, namespace, chunk_index)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (id) DO UPDATE SET
			content = EXCLUDED.content,
			embedding = EXCLUDED.embedding,
			metadata = EXCLUDED.metadata
	`, vs.tableName)

	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, chunk := range chunks {
		if chunk.ID == "" {
			chunk.ID = uuid.New().String()
		}

		metadataJSON, err := json.Marshal(chunk.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}

		embeddingStr := vectorToString(chunk.Embedding)
		namespace := chunk.Metadata["namespace"]
		if namespace == "" {
			namespace = "default"
		}

		source := chunk.Metadata["source"]
		_, err = stmt.ExecContext(ctx,
			chunk.ID,
			chunk.DocumentID,
			chunk.Content,
			embeddingStr,
			metadataJSON,
			source,
			namespace,
			chunk.StartIndex,
		)
		if err != nil {
			return fmt.Errorf("failed to upsert chunk %s: %w", chunk.ID, err)
		}
	}

	return tx.Commit()
}

// Search finds similar chunks using pgvector.
func (vs *PostgresVectorStore) Search(ctx context.Context, embedding []float64, query SearchQuery) ([]SearchResult, error) {
	embeddingStr := vectorToString(embedding)

	// Build WHERE clause
	whereParts := []string{"1=1"}
	args := []interface{}{embeddingStr}
	argNum := 2

	if query.Namespace != "" {
		whereParts = append(whereParts, fmt.Sprintf("namespace = $%d", argNum))
		args = append(args, query.Namespace)
		argNum++
	}

	for key, value := range query.Filters {
		whereParts = append(whereParts, fmt.Sprintf("metadata->>'%s' = $%d", key, argNum))
		args = append(args, value)
		argNum++
	}

	topK := query.TopK
	if topK <= 0 {
		topK = 10
	}

	sqlQuery := fmt.Sprintf(`
		SELECT id, document_id, content, embedding, metadata, source, chunk_index,
			   1 - (embedding <=> $1) as similarity
		FROM %s
		WHERE %s
		ORDER BY embedding <=> $1
		LIMIT $%d
	`, vs.tableName, strings.Join(whereParts, " AND "), argNum)

	args = append(args, topK)

	rows, err := vs.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to search: %w", err)
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var chunk Chunk
		var embeddingStr string
		var metadataJSON []byte
		var similarity float64
		var source string
		var chunkIndex int

		err := rows.Scan(
			&chunk.ID,
			&chunk.DocumentID,
			&chunk.Content,
			&embeddingStr,
			&metadataJSON,
			&source,
			&chunkIndex,
			&similarity,
		)
		_ = source     // stored in metadata
		_ = chunkIndex // stored as StartIndex
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		chunk.Embedding = stringToVector(embeddingStr)
		if err := json.Unmarshal(metadataJSON, &chunk.Metadata); err != nil {
			chunk.Metadata = make(map[string]string)
		}

		if similarity >= query.MinScore {
			results = append(results, SearchResult{
				Chunk:    chunk,
				Score:    similarity,
				Distance: 1 - similarity,
			})
		}
	}

	return results, rows.Err()
}

// Delete removes chunks by their IDs.
func (vs *PostgresVectorStore) Delete(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	placeholders := make([]string, len(ids))
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}

	query := fmt.Sprintf("DELETE FROM %s WHERE id IN (%s)", vs.tableName, strings.Join(placeholders, ","))
	_, err := vs.db.ExecContext(ctx, query, args...)
	return err
}

// Clear removes all chunks from the specified namespace.
func (vs *PostgresVectorStore) Clear(ctx context.Context, namespace string) error {
	if namespace == "" {
		namespace = "default"
	}
	query := fmt.Sprintf("DELETE FROM %s WHERE namespace = $1", vs.tableName)
	_, err := vs.db.ExecContext(ctx, query, namespace)
	return err
}

// Count returns the number of chunks in the store.
func (vs *PostgresVectorStore) Count(ctx context.Context, namespace string) (int64, error) {
	var count int64
	var query string
	var args []interface{}

	if namespace == "" {
		query = fmt.Sprintf("SELECT COUNT(*) FROM %s", vs.tableName)
	} else {
		query = fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE namespace = $1", vs.tableName)
		args = append(args, namespace)
	}

	err := vs.db.QueryRowContext(ctx, query, args...).Scan(&count)
	return count, err
}

// GetChunk retrieves a chunk by ID.
func (vs *PostgresVectorStore) GetChunk(ctx context.Context, id string) (*Chunk, error) {
	query := fmt.Sprintf(`
		SELECT id, document_id, content, embedding, metadata, source, chunk_index
		FROM %s WHERE id = $1
	`, vs.tableName)

	var chunk Chunk
	var embeddingStr string
	var metadataJSON []byte
	var source string
	var chunkIndex int

	err := vs.db.QueryRowContext(ctx, query, id).Scan(
		&chunk.ID,
		&chunk.DocumentID,
		&chunk.Content,
		&embeddingStr,
		&metadataJSON,
		&source,
		&chunkIndex,
	)
	_ = source     // stored in metadata
	_ = chunkIndex // not used

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	chunk.Embedding = stringToVector(embeddingStr)
	if err := json.Unmarshal(metadataJSON, &chunk.Metadata); err != nil {
		chunk.Metadata = make(map[string]string)
	}

	return &chunk, nil
}

// Close closes the database connection.
func (vs *PostgresVectorStore) Close() error {
	return vs.db.Close()
}

// Helper functions

func vectorToString(v []float64) string {
	if len(v) == 0 {
		return "[]"
	}
	parts := make([]string, len(v))
	for i, val := range v {
		parts[i] = fmt.Sprintf("%f", val)
	}
	return "[" + strings.Join(parts, ",") + "]"
}

func stringToVector(s string) []float64 {
	s = strings.Trim(s, "[]")
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	result := make([]float64, len(parts))
	for i, p := range parts {
		fmt.Sscanf(strings.TrimSpace(p), "%f", &result[i])
	}
	return result
}
