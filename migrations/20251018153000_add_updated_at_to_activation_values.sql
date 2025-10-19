-- +goose Up
-- +goose StatementBegin
alter table activation_values
add column if not exists updated_at timestamp not null default now();

create or replace function set_activation_values_updated_at()
returns trigger as $$
begin
    new.updated_at := now();
    return new;
end;
$$ language plpgsql;

drop trigger if exists trg_activation_values_set_updated_at on activation_values;

create trigger trg_activation_values_set_updated_at
before update on activation_values
for each row
execute function set_activation_values_updated_at();
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
drop trigger if exists trg_activation_values_set_updated_at on activation_values;

drop function if exists set_activation_values_updated_at ();

alter table activation_values drop column if exists updated_at;
-- +goose StatementEnd