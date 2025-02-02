-- +goose Up
-- +goose StatementBegin
create table activation_keys
(
    id uuid primary key,
    feature_id uuid not null references features(id),
    key varchar(255) not null,
    description text,
    created_at timestamp not null default now()
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
drop table activation_keys;
-- +goose StatementEnd
