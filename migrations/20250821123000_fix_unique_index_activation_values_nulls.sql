-- +goose Up
-- +goose StatementBegin
-- 1) Remove older duplicates among non-deleted rows, keep the latest by v per scope
WITH
    ranked AS (
        SELECT id, ROW_NUMBER() OVER (
                PARTITION BY
                    feature_id, COALESCE(
                        activation_key_id, '00000000-0000-0000-0000-000000000000'::uuid
                    ), COALESCE(
                        activation_param_id, '00000000-0000-0000-0000-000000000000'::uuid
                    )
                ORDER BY v DESC
            ) AS rn
        FROM activation_values
        WHERE
            deleted_at IS NULL
    )
DELETE FROM activation_values av USING ranked r
WHERE
    av.id = r.id
    AND r.rn > 1;

-- 2) Replace previous uniqueness with a scope-aware unique index that treats NULLs as equal
DROP INDEX IF EXISTS ux_activation_values_triplet_not_deleted;

CREATE UNIQUE INDEX IF NOT EXISTS ux_av_scope_not_deleted ON activation_values (
    feature_id,
    COALESCE(
        activation_key_id,
        '00000000-0000-0000-0000-000000000000'::uuid
    ),
    COALESCE(
        activation_param_id,
        '00000000-0000-0000-0000-000000000000'::uuid
    )
)
WHERE
    deleted_at IS NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS ux_av_scope_not_deleted;

-- Recreate the previous index (without NULLS handling) for rollback compatibility
CREATE UNIQUE INDEX IF NOT EXISTS ux_activation_values_triplet_not_deleted ON activation_values (
    feature_id,
    activation_key_id,
    activation_param_id
)
WHERE
    deleted_at IS NULL;
-- +goose StatementEnd