-- +goose Up
-- +goose StatementBegin
create table features
(
    id uuid primary key,
    name varchar(255) not null unique,
    description text,
    created_at timestamp not null default now()
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
drop table features;
-- +goose StatementEnd
