-- +goose Up
-- +goose StatementBegin
-- Ensure uniqueness for non-deleted activation keys within a feature
create unique index if not exists ux_activation_keys_feature_key_not_deleted on activation_keys (feature_id, key)
where
    deleted_at is null;

-- Ensure uniqueness for non-deleted activation params within a key
create unique index if not exists ux_activation_params_activation_name_not_deleted on activation_params (activation_id, name)
where
    deleted_at is null;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
drop index if exists ux_activation_params_activation_name_not_deleted;

drop index if exists ux_activation_keys_feature_key_not_deleted;
-- +goose StatementEnd