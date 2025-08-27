-- +goose Up
-- +goose StatementBegin
-- Ensure uniqueness for service access pairs so ON CONFLICT works
create unique index if not exists ux_service_access_feature_service on service_access (feature_id, service_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
drop index if exists ux_service_access_feature_service;
-- +goose StatementEnd