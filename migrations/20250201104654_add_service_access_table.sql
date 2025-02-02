-- +goose Up
-- +goose StatementBegin
create table service_access
(
    id uuid primary key,
    feature_id uuid not null references features(id),
    service_id uuid not null references services(id)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
drop table service_access;
-- +goose StatementEnd
