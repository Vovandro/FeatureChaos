-- +goose Up
-- +goose StatementBegin
-- Add deleted_at to features
alter table features
add column if not exists deleted_at timestamp null;

create index if not exists idx_features_deleted_at on features (deleted_at);

-- Add deleted_at to activation_keys
alter table activation_keys
add column if not exists deleted_at timestamp null;

create index if not exists idx_activation_keys_deleted_at on activation_keys (deleted_at);

-- Add deleted_at to activation_params
alter table activation_params
add column if not exists deleted_at timestamp null;

create index if not exists idx_activation_params_deleted_at on activation_params (deleted_at);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- Drop indexes first, then columns
drop index if exists idx_activation_params_deleted_at;

alter table activation_params drop column if exists deleted_at;

drop index if exists idx_activation_keys_deleted_at;

alter table activation_keys drop column if exists deleted_at;

drop index if exists idx_features_deleted_at;

alter table features drop column if exists deleted_at;
-- +goose StatementEnd