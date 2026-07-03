create extension if not exists pgcrypto;

do $$
begin
  if not exists (select 1 from pg_type where typname = 'content_source_status') then
    create type content_source_status as enum ('active', 'paused', 'error');
  end if;

  if not exists (select 1 from pg_type where typname = 'trend_status') then
    create type trend_status as enum ('new', 'candidate', 'selected', 'rejected', 'archived');
  end if;

  if not exists (select 1 from pg_type where typname = 'article_status') then
    create type article_status as enum (
      'draft',
      'researching',
      'topic_review',
      'outline_review',
      'drafting',
      'final_review',
      'ready_to_publish',
      'published',
      'failed',
      'archived'
    );
  end if;

  if not exists (select 1 from pg_type where typname = 'artifact_kind') then
    create type artifact_kind as enum (
      'research_brief',
      'topic_card',
      'outline',
      'draft',
      'translation',
      'title_candidates',
      'image_plan',
      'quality_report',
      'publish_package'
    );
  end if;

  if not exists (select 1 from pg_type where typname = 'job_status') then
    create type job_status as enum ('queued', 'running', 'succeeded', 'failed', 'cancelled');
  end if;
end $$;

create table if not exists skill_repositories (
  id uuid primary key default gen_random_uuid(),
  name text not null unique,
  repo_url text not null,
  branch text not null default 'main',
  status content_source_status not null default 'active',
  last_synced_commit text,
  last_synced_at timestamptz,
  config jsonb not null default '{}'::jsonb,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

create table if not exists skill_snapshots (
  id uuid primary key default gen_random_uuid(),
  repository_id uuid not null references skill_repositories(id) on delete cascade,
  skill_name text not null,
  display_name text,
  description text,
  commit_sha text not null,
  content_hash text not null,
  skill_md text not null,
  reference_files jsonb not null default '{}'::jsonb,
  manifest jsonb not null default '{}'::jsonb,
  created_at timestamptz not null default now(),
  unique (repository_id, skill_name, commit_sha)
);

create table if not exists content_sources (
  id uuid primary key default gen_random_uuid(),
  provider text not null,
  name text not null,
  source_url text not null,
  auth_secret_ref text,
  poll_interval_minutes integer not null default 60 check (poll_interval_minutes > 0),
  status content_source_status not null default 'active',
  config jsonb not null default '{}'::jsonb,
  last_polled_at timestamptz,
  last_cursor text,
  last_error text,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  unique (provider, source_url)
);

create table if not exists trend_items (
  id uuid primary key default gen_random_uuid(),
  source_id uuid not null references content_sources(id) on delete cascade,
  external_id text,
  canonical_url text not null,
  title text not null,
  summary text,
  channel text,
  language text not null default 'en',
  source_published_at timestamptz,
  highlighted_at timestamptz,
  significance text,
  tags text[] not null default '{}'::text[],
  metrics jsonb not null default '{}'::jsonb,
  raw_payload jsonb not null default '{}'::jsonb,
  dedupe_hash text not null unique,
  status trend_status not null default 'new',
  selected_at timestamptz,
  rejected_reason text,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  unique (source_id, external_id)
);

create table if not exists trend_evaluations (
  id uuid primary key default gen_random_uuid(),
  trend_id uuid not null references trend_items(id) on delete cascade,
  skill_snapshot_id uuid references skill_snapshots(id) on delete set null,
  model_provider text not null,
  model text not null,
  prompt_version text not null,
  h_score integer not null check (h_score between 0 and 5),
  k_score integer not null check (k_score between 0 and 5),
  r_score integer not null check (r_score between 0 and 5),
  brand_fit_score integer not null default 0 check (brand_fit_score between 0 and 5),
  risk_score integer not null default 0 check (risk_score between 0 and 5),
  total_score integer generated always as (
    h_score + k_score + r_score + brand_fit_score - risk_score
  ) stored,
  confidence numeric(4,3) check (confidence >= 0 and confidence <= 1),
  recommended boolean not null default false,
  target_audience text,
  recommended_angle text,
  risk_notes text,
  output jsonb not null default '{}'::jsonb,
  created_at timestamptz not null default now()
);

create table if not exists article_projects (
  id uuid primary key default gen_random_uuid(),
  trend_id uuid references trend_items(id) on delete set null,
  status article_status not null default 'draft',
  canonical_language text not null default 'zh-CN',
  target_languages text[] not null default array['zh-CN', 'en-US'],
  working_title text,
  audience text,
  core_claim text,
  owner_user_id text,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

create table if not exists article_runs (
  id uuid primary key default gen_random_uuid(),
  project_id uuid not null references article_projects(id) on delete cascade,
  status job_status not null default 'queued',
  current_stage text not null default 'research_brief',
  workflow_version text not null default 'v1',
  requested_by text,
  started_at timestamptz,
  completed_at timestamptz,
  error_message text,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

create table if not exists article_artifacts (
  id uuid primary key default gen_random_uuid(),
  project_id uuid not null references article_projects(id) on delete cascade,
  run_id uuid references article_runs(id) on delete set null,
  kind artifact_kind not null,
  language text,
  title text,
  body_md text not null,
  metadata jsonb not null default '{}'::jsonb,
  skill_snapshot_ids uuid[] not null default '{}'::uuid[],
  model_provider text,
  model text,
  source_citations jsonb not null default '[]'::jsonb,
  quality_score numeric(5,2),
  superseded_by uuid references article_artifacts(id) on delete set null,
  created_at timestamptz not null default now()
);

create table if not exists article_reviews (
  id uuid primary key default gen_random_uuid(),
  project_id uuid not null references article_projects(id) on delete cascade,
  artifact_id uuid references article_artifacts(id) on delete cascade,
  reviewer_user_id text,
  status text not null check (status in ('requested', 'changes_requested', 'approved')),
  notes text,
  created_at timestamptz not null default now()
);

create table if not exists media_assets (
  id uuid primary key default gen_random_uuid(),
  project_id uuid not null references article_projects(id) on delete cascade,
  artifact_id uuid references article_artifacts(id) on delete set null,
  role text not null,
  prompt text,
  provider text,
  asset_url text,
  local_path text,
  status job_status not null default 'queued',
  metadata jsonb not null default '{}'::jsonb,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

create table if not exists automation_jobs (
  id uuid primary key default gen_random_uuid(),
  job_type text not null,
  status job_status not null default 'queued',
  scheduled_for timestamptz not null default now(),
  locked_at timestamptz,
  attempts integer not null default 0,
  max_attempts integer not null default 3,
  payload jsonb not null default '{}'::jsonb,
  last_error text,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

create table if not exists publication_packages (
  id uuid primary key default gen_random_uuid(),
  project_id uuid not null references article_projects(id) on delete cascade,
  language text not null,
  title text not null,
  slug text,
  body_artifact_id uuid references article_artifacts(id) on delete set null,
  cover_asset_id uuid references media_assets(id) on delete set null,
  target text not null default 'wechat',
  status article_status not null default 'ready_to_publish',
  metadata jsonb not null default '{}'::jsonb,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

create index if not exists idx_skill_snapshots_skill_commit
  on skill_snapshots (skill_name, commit_sha);

create index if not exists idx_content_sources_status
  on content_sources (status, provider);

create index if not exists idx_trend_items_status_highlighted
  on trend_items (status, highlighted_at desc);

create index if not exists idx_trend_items_channel_highlighted
  on trend_items (channel, highlighted_at desc);

create index if not exists idx_trend_items_tags
  on trend_items using gin (tags);

create index if not exists idx_trend_evaluations_recommended_score
  on trend_evaluations (recommended, total_score desc, created_at desc);

create index if not exists idx_article_projects_status
  on article_projects (status, updated_at desc);

create index if not exists idx_article_artifacts_project_kind
  on article_artifacts (project_id, kind, created_at desc);

create index if not exists idx_automation_jobs_claim
  on automation_jobs (status, scheduled_for, locked_at);

insert into skill_repositories (name, repo_url, branch, status, config)
values (
  'agi01-skills',
  'https://github.com/happycto/agi01-skills',
  'main',
  'active',
  '{"purpose":"AGI01 article generation workflow"}'::jsonb
)
on conflict (name) do nothing;

insert into content_sources (provider, name, source_url, poll_interval_minutes, status, config)
values (
  'daily_dev_highlights',
  'daily.dev Highlights',
  'https://daily.dev/highlights',
  10,
  'active',
  '{"adapter":"next_data_highlights","preferred_future_adapter":"daily_dev_public_api"}'::jsonb
)
on conflict (provider, source_url) do nothing;
