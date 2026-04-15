package session

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"

	"github.com/nice-code/codego/internal/types"
)

// Store persists sessions and messages in SQLite.
type Store struct {
	db *sql.DB
}

// SessionInfo is a session summary for listing.
type SessionInfo struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	MsgCount  int       `json:"msg_count"`
}

// Open opens (or creates) a session store at the given path.
func Open(path string) (*Store, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create dir: %w", err)
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	// Enable WAL mode for better concurrency
	_, _ = db.Exec("PRAGMA journal_mode=WAL")

	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}

	return s, nil
}

// OpenDefault opens the default session store.
func OpenDefault() (*Store, error) {
	home, _ := os.UserHomeDir()
	return Open(filepath.Join(home, ".codego", "sessions.db"))
}

// Close closes the database.
func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) migrate() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS sessions (
			id         TEXT PRIMARY KEY,
			title      TEXT NOT NULL DEFAULT '',
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL
		);

		CREATE TABLE IF NOT EXISTS messages (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			session_id TEXT NOT NULL,
			role       TEXT NOT NULL,
			content    TEXT NOT NULL,
			created_at DATETIME NOT NULL,
			FOREIGN KEY (session_id) REFERENCES sessions(id) ON DELETE CASCADE
		);

		CREATE INDEX IF NOT EXISTS idx_messages_session ON messages(session_id);
		CREATE INDEX IF NOT EXISTS idx_sessions_updated ON sessions(updated_at);
	`)
	return err
}

// CreateSession creates a new session.
func (s *Store) CreateSession(id, title string) error {
	now := time.Now()
	_, err := s.db.Exec(
		"INSERT INTO sessions (id, title, created_at, updated_at) VALUES (?, ?, ?, ?)",
		id, title, now, now,
	)
	return err
}

// UpdateTitle updates the session title.
func (s *Store) UpdateTitle(id, title string) error {
	_, err := s.db.Exec(
		"UPDATE sessions SET title = ?, updated_at = ? WHERE id = ?",
		title, time.Now(), id,
	)
	return err
}

// DeleteSession deletes a session and its messages.
func (s *Store) DeleteSession(id string) error {
	_, err := s.db.Exec("DELETE FROM messages WHERE session_id = ?", id)
	if err != nil {
		return err
	}
	_, err = s.db.Exec("DELETE FROM sessions WHERE id = ?", id)
	return err
}

// ListSessions returns all sessions ordered by most recent.
func (s *Store) ListSessions(limit int) ([]SessionInfo, error) {
	if limit <= 0 {
		limit = 50
	}

	rows, err := s.db.Query(`
		SELECT s.id, s.title, s.created_at, s.updated_at, COUNT(m.id) as msg_count
		FROM sessions s
		LEFT JOIN messages m ON m.session_id = s.id
		GROUP BY s.id
		ORDER BY s.updated_at DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []SessionInfo
	for rows.Next() {
		var si SessionInfo
		if err := rows.Scan(&si.ID, &si.Title, &si.CreatedAt, &si.UpdatedAt, &si.MsgCount); err != nil {
			return nil, err
		}
		sessions = append(sessions, si)
	}
	return sessions, rows.Err()
}

// GetSession returns a single session's info.
func (s *Store) GetSession(id string) (*SessionInfo, error) {
	var si SessionInfo
	err := s.db.QueryRow(`
		SELECT s.id, s.title, s.created_at, s.updated_at, COUNT(m.id)
		FROM sessions s
		LEFT JOIN messages m ON m.session_id = s.id
		WHERE s.id = ?
		GROUP BY s.id
	`, id).Scan(&si.ID, &si.Title, &si.CreatedAt, &si.UpdatedAt, &si.MsgCount)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("session not found: %s", id)
	}
	return &si, err
}

// AppendMessage adds a message to a session.
func (s *Store) AppendMessage(sessionID string, msg types.Message) error {
	content, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal message: %w", err)
	}

	_, err = s.db.Exec(
		"INSERT INTO messages (session_id, role, content, created_at) VALUES (?, ?, ?, ?)",
		sessionID, string(msg.Role), string(content), time.Now(),
	)
	if err != nil {
		return err
	}

	// Touch session updated_at
	_, err = s.db.Exec("UPDATE sessions SET updated_at = ? WHERE id = ?", time.Now(), sessionID)
	return err
}

// GetMessages returns all messages for a session, ordered by creation.
func (s *Store) GetMessages(sessionID string) ([]types.Message, error) {
	rows, err := s.db.Query(
		"SELECT content FROM messages WHERE session_id = ? ORDER BY id ASC",
		sessionID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var msgs []types.Message
	for rows.Next() {
		var content string
		if err := rows.Scan(&content); err != nil {
			return nil, err
		}
		var msg types.Message
		if err := json.Unmarshal([]byte(content), &msg); err != nil {
			return nil, fmt.Errorf("unmarshal message: %w", err)
		}
		msgs = append(msgs, msg)
	}
	return msgs, rows.Err()
}

// MessageCount returns the number of messages in a session.
func (s *Store) MessageCount(sessionID string) (int, error) {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM messages WHERE session_id = ?", sessionID).Scan(&count)
	return count, err
}

// ExportJSONL exports a session's messages as JSONL.
func (s *Store) ExportJSONL(sessionID string) (string, error) {
	msgs, err := s.GetMessages(sessionID)
	if err != nil {
		return "", err
	}

	var output string
	for _, msg := range msgs {
		data, err := json.Marshal(msg)
		if err != nil {
			return "", err
		}
		output += string(data) + "\n"
	}
	return output, nil
}

// SearchSessions searches sessions by title or message content.
func (s *Store) SearchSessions(query string, limit int) ([]SessionInfo, error) {
	if limit <= 0 {
		limit = 20
	}

	likeQuery := "%" + query + "%"
	rows, err := s.db.Query(`
		SELECT DISTINCT s.id, s.title, s.created_at, s.updated_at, COUNT(m2.id) as msg_count
		FROM sessions s
		LEFT JOIN messages m ON m.session_id = s.id
		LEFT JOIN messages m2 ON m2.session_id = s.id
		WHERE s.title LIKE ? OR m.content LIKE ?
		GROUP BY s.id
		ORDER BY s.updated_at DESC
		LIMIT ?
	`, likeQuery, likeQuery, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []SessionInfo
	for rows.Next() {
		var si SessionInfo
		if err := rows.Scan(&si.ID, &si.Title, &si.CreatedAt, &si.UpdatedAt, &si.MsgCount); err != nil {
			return nil, err
		}
		sessions = append(sessions, si)
	}
	return sessions, rows.Err()
}
