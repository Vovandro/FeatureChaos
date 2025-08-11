-- +goose Up
-- +goose StatementBegin
-- Latest global feature value (no key, no param)
CREATE INDEX IF NOT EXISTS idx_av_feature_latest ON activation_values (feature_id, v DESC)
WHERE
    activation_key_id IS NULL
    AND activation_param_id IS NULL
    AND deleted_at IS NULL;

-- Latest key value (no param)
CREATE INDEX IF NOT EXISTS idx_av_key_latest ON activation_values (activation_key_id, v DESC)
WHERE
    activation_param_id IS NULL
    AND deleted_at IS NULL;

-- Latest param value
CREATE INDEX IF NOT EXISTS idx_av_param_latest ON activation_values (activation_param_id, v DESC)
WHERE
    deleted_at IS NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_av_param_latest;

DROP INDEX IF EXISTS idx_av_key_latest;

DROP INDEX IF EXISTS idx_av_feature_latest;
-- +goose StatementEnd