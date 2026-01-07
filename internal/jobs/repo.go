package jobs

import (
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

type Repo struct {
	DB *gorm.DB
}

func (r *Repo) EnqueueReminder(userID uint64, memoID uint64, runAt time.Time) error {
	payload, _ := json.Marshal(map[string]any{
		"memo_id": memoID,
	})
	j := Job{
		UserID:  userID,
		Type:    "REMINDER_DISPATCH",
		Payload: payload,
		RunAt:   runAt,
		Status:  "PENDING",
	}
	return r.DB.Create(&j).Error
}

// Claim one due job atomically using SKIP LOCKED.
// Works on Postgres.
func (r *Repo) Claim(workerID string) (*Job, error) {
	var job Job
	err := r.DB.Transaction(func(tx *gorm.DB) error {
		// requeue stuck RUNNING jobs (optional safety)
		tx.Exec(`
update jobs
set status='PENDING', locked_by=null, locked_at=null, updated_at=now()
where status='RUNNING' and locked_at is not null and locked_at < now() - interval '5 minutes'
`)

		// claim
		// FOR UPDATE SKIP LOCKED ensures no double-claim
		q := tx.Raw(`
with cte as (
  select id
  from jobs
  where status='PENDING' and run_at <= now()
  order by run_at asc
  for update skip locked
  limit 1
)
update jobs
set status='RUNNING', locked_by=?, locked_at=now(), updated_at=now()
where id in (select id from cte)
returning *;
`, workerID)

		return q.Scan(&job).Error
	})
	if err != nil {
		return nil, err
	}
	if job.ID == 0 {
		return nil, nil
	}
	return &job, nil
}

func (r *Repo) MarkDone(id uint64) error {
	return r.DB.Exec(`update jobs set status='DONE', updated_at=now() where id=?`, id).Error
}

func (r *Repo) MarkFailed(id uint64, errMsg string) error {
	return r.DB.Exec(`update jobs set status='FAILED', last_error=?, updated_at=now() where id=?`, errMsg, id).Error
}

func (r *Repo) RetryLater(id uint64, attempts int, runAt time.Time, errMsg string) error {
	return r.DB.Exec(`
update jobs
set status='PENDING',
    attempts=?,
    run_at=?,
    locked_by=null,
    locked_at=null,
    last_error=?,
    updated_at=now()
where id=?`, attempts, runAt, errMsg, id).Error
}
