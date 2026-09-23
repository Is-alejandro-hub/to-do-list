-- ============================================================================
-- PROYECTO: To-Do List Full Stack
-- ARCHIVO: 01_schema.sql
-- DESCRIPCIÓN: Esquema completo de la base de datos
-- ============================================================================

-- ============================================================================
-- 0. PREPARACIÓN
-- ============================================================================

-- gen_random_uuid() está disponible desde PostgreSQL 13 sin extensiones.

DO $$
BEGIN
    IF (current_setting('server_version_num')::int < 130000) THEN
        RAISE EXCEPTION 'Se requiere PostgreSQL 13 o superior.';
    END IF;
END $$;

-- ============================================================================
-- 1. TIPOS PERSONALIZADOS
-- ============================================================================

-- ENUM para la prioridad. Elegi ENUM nativo de PostgreSQL en lugar de
-- VARCHAR + CHECK porque el motor valida el valor automáticamente y el
-- tipo queda documentado en el esquema.
CREATE TYPE task_priority AS ENUM ('LOW', 'MEDIUM', 'HIGH');

-- ============================================================================
-- 2. TABLA: tasks
-- ============================================================================

CREATE TABLE tasks (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title       VARCHAR(255) NOT NULL,
    description TEXT,
    priority    task_priority NOT NULL DEFAULT 'MEDIUM',
    due_date    TIMESTAMPTZ,
    completed   BOOLEAN NOT NULL DEFAULT FALSE,
    deleted_at  TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_title_not_blank CHECK (LENGTH(TRIM(title)) > 0)
);

COMMENT ON TABLE  tasks             IS 'Tareas del sistema. Soporta soft delete.';
COMMENT ON COLUMN tasks.deleted_at  IS 'NULL = activa. No NULL = eliminada lógicamente.';
COMMENT ON COLUMN tasks.due_date    IS 'Fecha límite opcional.';
COMMENT ON COLUMN tasks.completed   IS 'Indica si la tarea fue finalizada.';

-- ============================================================================
-- 3. TABLA: tags
-- ============================================================================

CREATE TABLE tags (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name       VARCHAR(50) NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_tag_name_not_blank CHECK (LENGTH(TRIM(name)) > 0)
);

COMMENT ON TABLE tags IS 'Catálogo global de etiquetas reutilizables.';

-- ============================================================================
-- 4. TABLA INTERMEDIA: task_tags (Muchos-a-Muchos)
-- ============================================================================

CREATE TABLE task_tags (
    task_id UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    tag_id  UUID NOT NULL REFERENCES tags(id)  ON DELETE CASCADE,
    PRIMARY KEY (task_id, tag_id)
);

COMMENT ON TABLE task_tags IS 'Relación N:M entre tareas y etiquetas.';

-- ============================================================================
-- 5. TABLA: task_audit_log
-- ============================================================================
-- Sin FK a tasks, deliberadamente. Ver explicación en el documento.

CREATE TABLE task_audit_log (
    id         BIGSERIAL PRIMARY KEY,
    task_id    UUID NOT NULL,
    action     VARCHAR(20) NOT NULL,
    old_data   JSONB,
    new_data   JSONB,
    changed_by VARCHAR(100) NOT NULL DEFAULT current_user,
    changed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_audit_action CHECK (
        action IN ('INSERT', 'UPDATE', 'SOFT_DELETE', 'RESTORE', 'DELETE')
    )
);

COMMENT ON TABLE task_audit_log IS 'Bitácora inmutable de cambios sobre tareas.';

-- ============================================================================
-- 6. TABLA: idempotency_keys
-- ============================================================================

CREATE TABLE idempotency_keys (
    key             UUID PRIMARY KEY,
    request_hash    VARCHAR(64) NOT NULL,
    response_status INT NOT NULL,
    response_body   JSONB NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at      TIMESTAMPTZ NOT NULL
);

COMMENT ON TABLE idempotency_keys IS 'Almacena respuestas de POST para evitar duplicados por reintentos.';
COMMENT ON COLUMN idempotency_keys.request_hash IS 'SHA-256 del body. Detecta reutilización de key con payload distinto.';

-- ============================================================================
-- 7. ÍNDICES
-- ============================================================================

-- Índices parciales: solo indexan tareas activas. Mucho más rápidos que un
-- índice completo cuando el 99% de las consultas filtran deleted_at IS NULL.
CREATE INDEX idx_tasks_active_due_date ON tasks (due_date)        WHERE deleted_at IS NULL;
CREATE INDEX idx_tasks_active_priority ON tasks (priority)        WHERE deleted_at IS NULL;
CREATE INDEX idx_tasks_active_created  ON tasks (created_at DESC) WHERE deleted_at IS NULL;

-- Índice sobre el nombre de tag para búsquedas por nombre (case-insensitive).
CREATE INDEX idx_tags_name_lower ON tags (LOWER(name));

-- Índices para el audit log.
CREATE INDEX idx_audit_task_id    ON task_audit_log (task_id);
CREATE INDEX idx_audit_changed_at ON task_audit_log (changed_at DESC);

-- Índice para la limpieza de claves expiradas.
CREATE INDEX idx_idempotency_expires_at ON idempotency_keys (expires_at);

-- ============================================================================
-- 8. FUNCIONES DE TRIGGER
-- ============================================================================

-- 8.1. Actualiza updated_at en cada UPDATE de tasks.
CREATE OR REPLACE FUNCTION fn_set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at := NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- 8.2. Registra cambios en task_audit_log automáticamente.
CREATE OR REPLACE FUNCTION fn_tasks_audit()
RETURNS TRIGGER AS $$
DECLARE
    v_action     VARCHAR(20);
    v_old        JSONB;
    v_new        JSONB;
    v_task_id    UUID;
    v_changed_by VARCHAR(100);
BEGIN
    -- Permite a la app Go setear el usuario vía SET LOCAL.
    -- Si no se setea, cae al usuario de la conexión PostgreSQL.
    v_changed_by := COALESCE(
        NULLIF(current_setting('app.current_user', TRUE), ''),
        current_user
    );

    IF TG_OP = 'INSERT' THEN
        v_action  := 'INSERT';
        v_old     := NULL;
        v_new     := to_jsonb(NEW);
        v_task_id := NEW.id;

    ELSIF TG_OP = 'UPDATE' THEN
        -- Distinguimos operaciones especiales sobre UPDATE.
        IF OLD.deleted_at IS NULL AND NEW.deleted_at IS NOT NULL THEN
            v_action := 'SOFT_DELETE';
        ELSIF OLD.deleted_at IS NOT NULL AND NEW.deleted_at IS NULL THEN
            v_action := 'RESTORE';
        ELSE
            v_action := 'UPDATE';
        END IF;
        v_old     := to_jsonb(OLD);
        v_new     := to_jsonb(NEW);
        v_task_id := NEW.id;

    ELSIF TG_OP = 'DELETE' THEN
        v_action  := 'DELETE';
        v_old     := to_jsonb(OLD);
        v_new     := NULL;
        v_task_id := OLD.id;
    END IF;

    INSERT INTO task_audit_log (task_id, action, old_data, new_data, changed_by)
    VALUES (v_task_id, v_action, v_old, v_new, v_changed_by);

    RETURN COALESCE(NEW, OLD);
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- 9. TRIGGERS
-- ============================================================================

-- BEFORE UPDATE: primero actualiza updated_at.
CREATE TRIGGER trg_tasks_set_updated_at
BEFORE UPDATE ON tasks
FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();

-- AFTER INSERT/UPDATE/DELETE: luego registra en la bitácora.
-- El orden importa: el audit log capturará el updated_at ya actualizado.
CREATE TRIGGER trg_tasks_audit
AFTER INSERT OR UPDATE OR DELETE ON tasks
FOR EACH ROW EXECUTE FUNCTION fn_tasks_audit();