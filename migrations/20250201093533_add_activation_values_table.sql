-- +goose Up
-- +goose StatementBegin
create table activation_values
(
    id uuid primary key,
    feature_id uuid not null references features(id),
    activation_key_id uuid references activation_keys(id),
    activation_param_id uuid references activation_params(id),
    value smallint not null default 0,
    deleted_at timestamp default null,
    v bigint not null default 0
);

create index idx_activation_values_feature_id on activation_values(feature_id, deleted_at, v);
create index idx_activation_values_activation_id on activation_values(activation_key_id, deleted_at, v);
create index idx_activation_values_activation_param_id on activation_values(activation_param_id, deleted_at, v);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
drop table activation_values;
-- +goose StatementEnd
