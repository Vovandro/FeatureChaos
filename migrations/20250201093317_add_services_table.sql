-- +goose Up
-- +goose StatementBegin
create table services
(
    id uuid primary key,
    name varchar(255) not null unique,
    created_at timestamp not null default now()
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
drop table services;
-- +goose StatementEnd
