```mermaid
erDiagram
    TASKS {
        uuid id PK
        varchar title
        text description
        task_priority priority
        timestamptz due_date
        boolean completed
        timestamptz deleted_at
        timestamptz created_at
        timestamptz updated_at
    }
    TAGS {
        uuid id PK
        varchar name UK
        timestamptz created_at
    }
    TASK_TAGS {
        uuid task_id PK,FK
        uuid tag_id PK,FK
    }
    TASK_AUDIT_LOG {
        bigserial id PK
        uuid task_id
        varchar action
        jsonb old_data
        jsonb new_data
        varchar changed_by
        timestamptz changed_at
    }
    IDEMPOTENCY_KEYS {
        uuid key PK
        varchar request_hash
        int response_status
        jsonb response_body
        timestamptz created_at
        timestamptz expires_at
    }

    TASKS ||--o{ TASK_TAGS : "tiene"
    TAGS ||--o{ TASK_TAGS : "aplicado a"
    TASKS ||--o{ TASK_AUDIT_LOG : "es auditada"