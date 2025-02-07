-- +goose Up
-- +goose StatementBegin
create table service_access
(
    id uuid primary key,
    feature_id uuid not null references features(id),
    service_id uuid not null references services(id)
);

create index idx_service_access_feature_id on service_access(feature_id);
create index idx_service_access_service_id on service_access(service_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
drop table service_access;
-- +goose StatementEnd
