alter table build_runs add column if not exists build_variable_set_ids text;

create table if not exists build_variable_sets (
    id text primary key,
    name text not null,
    scope text not null default 'global',
    owner_ref text,
    variables text,
    enabled boolean not null default true,
    created_by text,
    created_at timestamptz,
    updated_at timestamptz,
    deleted_at timestamptz
);

create index if not exists idx_build_variable_sets_scope on build_variable_sets(scope);
create index if not exists idx_build_variable_sets_owner_ref on build_variable_sets(owner_ref);
create index if not exists idx_build_variable_sets_created_by on build_variable_sets(created_by);
create index if not exists idx_build_variable_sets_deleted_at on build_variable_sets(deleted_at);
