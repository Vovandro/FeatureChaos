-- +goose Up
-- +goose StatementBegin
-- Ensure uniqueness for latest value per (feature,key,param) among non-deleted rows
create unique index if not exists ux_activation_values_triplet_not_deleted on activation_values (
    feature_id,
    activation_key_id,
    activation_param_id
)
where
    deleted_at is null;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
drop index if exists ux_activation_values_triplet_not_deleted;
-- +goose StatementEnd