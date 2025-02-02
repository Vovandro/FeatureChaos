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
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
drop table activation_values;
-- +goose StatementEnd
