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

create index idx_activation_keys_feature_id on activation_keys(feature_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
drop table activation_keys;
-- +goose StatementEnd
