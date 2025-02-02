-- +goose Up
-- +goose StatementBegin
create table activation_params
(
    id uuid primary key,
    feature_id uuid not null references features(id),
    activation_id uuid not null references activation_keys(id),
    name varchar(255) not null,
    created_at timestamp not null default now()
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
drop table activation_params;
-- +goose StatementEnd
