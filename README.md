# Tell — Event-Driven Memo API (Go)

Tell adalah backend API sederhana untuk mencatat memo berbasis **event sourcing + projection**, dengan dukungan **tag dinamis**, **search**, dan **background reminder worker**.
Project ini dibuat sebagai latihan backend Go dengan pendekatan yang mendekati production (bukan CRUD biasa).

---

## ✨ Features

* 🔐 JWT Authentication
* 📝 Memo berbasis **event log** (CREATED, UPDATED, ARCHIVED, dll)
* 🧠 **Projection table** untuk query cepat
* #️⃣ **Dynamic hashtag parsing** (`#tag`) dari konten
* 🔎 Search memo (`q=`), filter tag & archived
* ⏰ Reminder dengan **background worker**
* 🧵 Job queue + retry (exponential backoff)
* 🌐 CORS-friendly API (siap FE)

---

## 🏗️ Architecture (High Level)

```
HTTP API (chi)
   │
   ▼
Domain Service (memo)
   │
   ├─ memo_events        ← event log (immutable)
   └─ memo_projections   ← current state (query fast)
           │
           ▼
       Background Worker
           │
           ▼
        jobs table
```

**Prinsip utama:**

* Write → event
* Read → projection
* Worker terpisah dari HTTP lifecycle

---

## 🧱 Tech Stack

* **Go**
* **chi** (HTTP router)
* **GORM**
* **PostgreSQL**
* **JWT**
* Background worker (goroutine)

---

## 📂 Project Structure

```
cmd/tell/                # main entry
internal/
  auth/                  # JWT + middleware
  memo/                  # domain (models, service, events)
  jobs/                  # job queue + worker
  http/
    handler/             # HTTP handlers
    router.go
  db/                    # db init & migration
```

---

## 🗄️ Database Schema (Simplified)

### memo_events

* id
* memo_id
* user_id
* type
* payload (jsonb)
* idempotency_key
* created_at

### memo_projections

* memo_id
* user_id
* content
* tags (text[])
* archived
* remind_at
* version
* updated_at

### jobs

* id
* user_id
* type
* payload (jsonb)
* run_at
* status
* attempts

---

## 🚀 Running Locally

### 1️⃣ Environment

```bash
export DATABASE_URL="postgres://user:pass@localhost:5432/tell?sslmode=disable"
export JWT_SECRET="supersecret"
```

### 2️⃣ Run

```bash
go run ./cmd/tell
```

Server output:

```
listening on :8080
```

Worker berjalan di proses yang sama.

---

## 🔐 Authentication

Semua endpoint `/memos/*` membutuhkan JWT.

Header:

```http
Authorization: Bearer <token>
```

---

## 📡 API Endpoints

### Create Memo

```http
POST /memos
```

```json
{
  "content": "cek PN mixer #pntrend #shift1"
}
```

---

### Update / Events

```http
POST /memos/{id}/events
```

```json
{
  "type": "UPDATED",
  "content": "cek ulang #finaltest"
}
```

Event types:

* CREATED
* UPDATED
* ARCHIVED / RESTORED
* REMINDER_SET
* REMINDER_CLEARED

---

### List Memos

```http
GET /memos?archived=false&tag=pntrend&q=mixer
```

Response:

```json
[
  {
    "memo_id": 2,
    "user_id": 1,
    "content": "...",
    "tags": ["pntrend","shift1"],
    "archived": false,
    "updated_at": "..."
  }
]
```

---

### Timeline (Event Log)

```http
GET /memos/{id}/timeline
```

```json
[
  {
    "type": "UPDATED",
    "payload": { "content": "..." },
    "created_at": "..."
  }
]
```

---

### Tags (Autocomplete)

```http
GET /memos/tags?q=pn&limit=10
```

```json
[
  { "tag": "pntrend", "count": 4 }
]
```

---

## 🏷️ Tag System

* Tag diambil otomatis dari konten (`#tag`)
* Lowercase, deduplicated, capped
* Disimpan di projection (`text[]`)
* Bisa difilter & autocomplete

---

## ⏰ Reminder System

* `REMINDER_SET` → enqueue job
* Worker polling `jobs` table
* Exponential backoff retry
* Dedupe reminder job per memo
* `REMINDER_CLEARED` → cancel pending job

---

## ⚡ Performance Notes

Indexes yang direkomendasikan:

```sql
CREATE INDEX ON memo_projections USING GIN (tags);
CREATE INDEX ON memo_projections (user_id, archived, updated_at DESC);
```

---

## 🎯 Why This Project?

Tell dibuat untuk latihan:

* event-driven backend design
* clean separation domain vs infra
* worker & async processing
* API yang siap dipakai frontend

Bukan sekadar CRUD.

---

## 📌 Next Ideas

* Full-text search (PostgreSQL FTS)
* Webhook / notification delivery
* Multi-user sharing
* Pagination

---

## 🧑‍💻 Author

Built by **Ajar**
Target: **Go Backend Intern / Junior Backend**
