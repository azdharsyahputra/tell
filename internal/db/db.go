package db

import (
	"fmt"

	"tell/internal/auth"
	"tell/internal/jobs"
	"tell/internal/memo"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func Connect(dsn string) (*gorm.DB, error) {
	gdb, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	return gdb, nil
}

func AutoMigrateAndIndexes(gdb *gorm.DB) error {
	// Tables
	if err := gdb.AutoMigrate(
		&memo.Memo{},
		&memo.MemoEvent{},
		&memo.MemoProjection{},
		&memo.Tag{},
		&memo.MemoTag{},
		&jobs.Job{},
		&auth.User{},
	); err != nil {
		return err
	}

	// Constraints / unique (user_id, name) for tags
	if err := gdb.Exec(`create unique index if not exists uq_tags_user_name on tags(user_id, name);`).Error; err != nil {
		return err
	}

	// Join table helper index
	if err := gdb.Exec(`create index if not exists idx_memo_tags_user_tag on memo_tags(user_id, tag_id);`).Error; err != nil {
		return err
	}

	// Projection tag filter (GIN for text[])
	if err := gdb.Exec(`create index if not exists idx_proj_tags on memo_projections using gin (tags);`).Error; err != nil {
		return err
	}

	// Full-text search on projection.content
	if err := gdb.Exec(`create index if not exists idx_proj_fts on memo_projections using gin (to_tsvector('simple', content));`).Error; err != nil {
		return err
	}

	// Event idempotency: unique per user + idempotency_key where not null
	// Note: table/column names depend on GORM naming. Default is snake_case plural.
	if err := gdb.Exec(`
create unique index if not exists uq_events_user_idem
on memo_events(user_id, idempotency_key)
where idempotency_key is not null;
`).Error; err != nil {
		return err
	}

	// Helpful indexes
	stmts := []string{
		`create index if not exists idx_events_memo on memo_events(memo_id, id);`,
		`create index if not exists idx_events_user_created on memo_events(user_id, created_at desc);`,
		`create index if not exists idx_proj_user_updated on memo_projections(user_id, updated_at desc);`,
		`create index if not exists idx_jobs_due on jobs(status, run_at);`,
		`create index if not exists idx_jobs_lock on jobs(status, locked_at);`,
	}
	for _, s := range stmts {
		if err := gdb.Exec(s).Error; err != nil {
			return fmt.Errorf("index exec failed: %w (sql=%s)", err, s)
		}
	}

	return nil
}
