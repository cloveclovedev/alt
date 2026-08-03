enum "plan_kind" {
  schema = schema.public
  values = ["daily", "weekly"]
}

enum "daily_planning_session_status" {
  schema = schema.public
  values = ["gathering", "context_blocked", "chatting", "reviewing", "finalized", "cancelled"]
}

enum "daily_planning_message_role" {
  schema = schema.public
  values = ["user", "assistant", "system"]
}

enum "calendar_source_role" {
  schema = schema.public
  values = ["commitment", "optional", "context"]
}

table "plans" {
  schema = schema.public

  column "id" {
    type    = uuid
    default = sql("gen_random_uuid()")
  }
  column "user_id" {
    type = uuid
    null = false
  }
  column "kind" {
    type = enum.plan_kind
    null = false
  }
  column "period_start" {
    type = date
    null = false
  }
  column "created_at" {
    type    = timestamptz
    default = sql("now()")
  }

  primary_key {
    columns = [column.id]
  }

  foreign_key "plans_user_id_fkey" {
    columns     = [column.user_id]
    ref_columns = [table.users.column.id]
    on_delete   = CASCADE
  }

  check "plans_weekly_period_start_monday" {
    expr = "kind <> 'weekly' OR EXTRACT(ISODOW FROM period_start) = 1"
  }

  index "plans_user_kind_period_start_key" {
    unique  = true
    columns = [column.user_id, column.kind, column.period_start]
  }
}

table "plan_revisions" {
  schema = schema.public

  column "id" {
    type    = uuid
    default = sql("gen_random_uuid()")
  }
  column "plan_id" {
    type = uuid
    null = false
  }
  column "revision" {
    type = integer
    null = false
  }
  column "summary_markdown" {
    type = text
    null = false
  }
  column "content_markdown" {
    type = text
    null = false
  }
  column "notes_markdown" {
    type = text
    null = true
  }
  column "created_at" {
    type    = timestamptz
    default = sql("now()")
  }

  primary_key {
    columns = [column.id]
  }

  foreign_key "plan_revisions_plan_id_fkey" {
    columns     = [column.plan_id]
    ref_columns = [table.plans.column.id]
    on_delete   = CASCADE
  }

  check "plan_revisions_revision_positive" {
    expr = "revision > 0"
  }
  check "plan_revisions_summary_markdown_not_blank" {
    expr = "btrim(summary_markdown) <> ''"
  }
  check "plan_revisions_content_markdown_not_blank" {
    expr = "btrim(content_markdown) <> ''"
  }

  index "plan_revisions_plan_revision_key" {
    unique  = true
    columns = [column.plan_id, column.revision]
  }
}

table "plan_revision_github_issues" {
  schema = schema.public

  column "id" {

    type = uuid

    default = sql("gen_random_uuid()")

  }
  column "plan_revision_id" {
    type = uuid
    null = false
  }
  column "repository_owner" {
    type = text
    null = false
  }
  column "repository_name" {
    type = text
    null = false
  }
  column "issue_number" {
    type = integer
    null = false
  }
  column "title" {
    type = text
    null = false
  }
  column "html_url" {
    type = text
    null = false
  }
  column "position" {
    type = integer
    null = false
  }

  primary_key {

    columns = [column.id]

  }
  foreign_key "plan_revision_github_issues_plan_revision_id_fkey" {
    columns     = [column.plan_revision_id]
    ref_columns = [table.plan_revisions.column.id]
    on_delete   = CASCADE
  }
  check "plan_revision_github_issues_owner_not_blank" {
    expr = "btrim(repository_owner) <> ''"
  }
  check "plan_revision_github_issues_name_not_blank" {
    expr = "btrim(repository_name) <> ''"
  }
  check "plan_revision_github_issues_number_positive" {
    expr = "issue_number > 0"
  }
  check "plan_revision_github_issues_title_not_blank" {
    expr = "btrim(title) <> ''"
  }
  check "plan_revision_github_issues_url_not_blank" {
    expr = "btrim(html_url) <> ''"
  }
  check "plan_revision_github_issues_position_non_negative" {
    expr = "position >= 0"
  }
  index "plan_revision_github_issues_plan_revision_position_key" {
    unique  = true
    columns = [column.plan_revision_id, column.position]
  }
}

table "plan_revision_routines" {
  schema = schema.public

  column "id" {

    type = uuid

    default = sql("gen_random_uuid()")

  }
  column "plan_revision_id" {
    type = uuid
    null = false
  }
  column "routine_id" {
    type = uuid
    null = false
  }
  column "position" {
    type = integer
    null = false
  }

  primary_key {

    columns = [column.id]

  }
  foreign_key "plan_revision_routines_plan_revision_id_fkey" {
    columns     = [column.plan_revision_id]
    ref_columns = [table.plan_revisions.column.id]
    on_delete   = CASCADE
  }
  foreign_key "plan_revision_routines_routine_id_fkey" {
    columns     = [column.routine_id]
    ref_columns = [table.routines.column.id]
  }
  check "plan_revision_routines_position_non_negative" {
    expr = "position >= 0"
  }
  index "plan_revision_routines_plan_revision_position_key" {
    unique  = true
    columns = [column.plan_revision_id, column.position]
  }
  index "plan_revision_routines_routine_id_idx" {
    columns = [column.routine_id]
  }
}

table "plan_revision_action_items" {
  schema = schema.public

  column "id" {

    type = uuid

    default = sql("gen_random_uuid()")

  }
  column "plan_revision_id" {
    type = uuid
    null = false
  }
  column "title" {
    type = text
    null = false
  }
  column "note" {
    type    = text
    null    = false
    default = ""
  }
  column "position" {
    type = integer
    null = false
  }

  primary_key {

    columns = [column.id]

  }
  foreign_key "plan_revision_action_items_plan_revision_id_fkey" {
    columns     = [column.plan_revision_id]
    ref_columns = [table.plan_revisions.column.id]
    on_delete   = CASCADE
  }
  check "plan_revision_action_items_title_not_blank" {
    expr = "btrim(title) <> ''"
  }
  check "plan_revision_action_items_position_non_negative" {
    expr = "position >= 0"
  }
  index "plan_revision_action_items_plan_revision_position_key" {
    unique  = true
    columns = [column.plan_revision_id, column.position]
  }
}

table "google_calendar_connections" {
  schema = schema.public

  column "id" {

    type = uuid

    default = sql("gen_random_uuid()")

  }
  column "user_id" {
    type = uuid
    null = false
  }
  column "encrypted_refresh_token" {
    type = bytea
    null = false
  }
  column "granted_scopes" {
    type = sql("text[]")
    null = false
  }
  column "created_at" {
    type    = timestamptz
    default = sql("now()")
  }
  column "updated_at" {
    type    = timestamptz
    default = sql("now()")
  }

  primary_key {

    columns = [column.id]

  }
  foreign_key "google_calendar_connections_user_id_fkey" {
    columns     = [column.user_id]
    ref_columns = [table.users.column.id]
    on_delete   = CASCADE
  }
  index "google_calendar_connections_user_id_key" {
    unique  = true
    columns = [column.user_id]
  }
}

table "calendar_sources" {
  schema = schema.public

  column "id" {

    type = uuid

    default = sql("gen_random_uuid()")

  }
  column "connection_id" {
    type = uuid
    null = false
  }
  column "external_calendar_id" {
    type = text
    null = false
  }
  column "display_name" {
    type = text
    null = false
  }
  column "enabled" {
    type    = boolean
    default = false
  }
  column "role" {
    type    = enum.calendar_source_role
    default = "commitment"
  }
  column "planning_instructions" {
    type    = text
    null    = false
    default = ""
  }
  column "timezone" {
    type    = text
    null    = false
    default = "UTC"
  }
  column "created_at" {
    type    = timestamptz
    default = sql("now()")
  }
  column "updated_at" {
    type    = timestamptz
    default = sql("now()")
  }

  primary_key {

    columns = [column.id]

  }
  foreign_key "calendar_sources_connection_id_fkey" {
    columns     = [column.connection_id]
    ref_columns = [table.google_calendar_connections.column.id]
    on_delete   = CASCADE
  }
  check "calendar_sources_external_id_not_blank" {
    expr = "btrim(external_calendar_id) <> ''"
  }
  check "calendar_sources_display_name_not_blank" {
    expr = "btrim(display_name) <> ''"
  }
  check "calendar_sources_timezone_not_blank" {
    expr = "btrim(timezone) <> ''"
  }
  index "calendar_sources_connection_external_id_key" {
    unique  = true
    columns = [column.connection_id, column.external_calendar_id]
  }
  index "calendar_sources_connection_enabled_idx" {
    columns = [column.connection_id, column.enabled]
  }
}

table "plan_revision_calendar_events" {
  schema = schema.public

  column "id" {

    type = uuid

    default = sql("gen_random_uuid()")

  }
  column "plan_revision_id" {
    type = uuid
    null = false
  }
  column "calendar_source_id" {
    type = uuid
    null = true
  }
  column "external_event_id" {
    type = text
    null = false
  }
  column "title" {
    type = text
    null = false
  }
  column "starts_at" {
    type = timestamptz
    null = true
  }
  column "ends_at" {
    type = timestamptz
    null = true
  }
  column "start_date" {
    type = date
    null = true
  }
  column "end_date" {
    type = date
    null = true
  }
  column "all_day" {
    type = boolean
    null = false
  }
  column "role" {
    type = enum.calendar_source_role
    null = false
  }
  column "html_url" {
    type    = text
    null    = false
    default = ""
  }
  column "position" {
    type = integer
    null = false
  }

  primary_key {

    columns = [column.id]

  }
  foreign_key "plan_revision_calendar_events_plan_revision_id_fkey" {
    columns     = [column.plan_revision_id]
    ref_columns = [table.plan_revisions.column.id]
    on_delete   = CASCADE
  }
  foreign_key "plan_revision_calendar_events_calendar_source_id_fkey" {
    columns     = [column.calendar_source_id]
    ref_columns = [table.calendar_sources.column.id]
    on_delete   = SET_NULL
  }
  check "plan_revision_calendar_events_external_id_not_blank" {
    expr = "btrim(external_event_id) <> ''"
  }
  check "plan_revision_calendar_events_title_not_blank" {
    expr = "btrim(title) <> ''"
  }
  check "plan_revision_calendar_events_position_non_negative" {
    expr = "position >= 0"
  }
  check "plan_revision_calendar_events_time_representation" {
    expr = "(all_day AND start_date IS NOT NULL AND end_date IS NOT NULL AND starts_at IS NULL AND ends_at IS NULL) OR (NOT all_day AND starts_at IS NOT NULL AND ends_at IS NOT NULL AND start_date IS NULL AND end_date IS NULL)"
  }
  index "plan_revision_calendar_events_plan_revision_position_key" {
    unique  = true
    columns = [column.plan_revision_id, column.position]
  }
  index "plan_revision_calendar_events_calendar_source_id_idx" {
    columns = [column.calendar_source_id]
  }
}

table "github_repositories" {
  schema = schema.public

  column "id" {

    type = uuid

    default = sql("gen_random_uuid()")

  }
  column "user_id" {
    type = uuid
    null = false
  }
  column "owner" {
    type = text
    null = false
  }
  column "name" {
    type = text
    null = false
  }
  column "enabled" {
    type    = boolean
    default = true
  }
  column "planning_instructions" {
    type    = text
    null    = false
    default = ""
  }
  column "created_at" {
    type    = timestamptz
    default = sql("now()")
  }
  column "updated_at" {
    type    = timestamptz
    default = sql("now()")
  }

  primary_key {

    columns = [column.id]

  }
  foreign_key "github_repositories_user_id_fkey" {
    columns     = [column.user_id]
    ref_columns = [table.users.column.id]
    on_delete   = CASCADE
  }
  check "github_repositories_owner_not_blank" {
    expr = "btrim(owner) <> ''"
  }
  check "github_repositories_name_not_blank" {
    expr = "btrim(name) <> ''"
  }
  index "github_repositories_user_owner_name_key" {
    unique  = true
    columns = [column.user_id, column.owner, column.name]
  }
  index "github_repositories_user_enabled_idx" {
    columns = [column.user_id, column.enabled]
  }
}

table "daily_planning_sessions" {
  schema = schema.public

  column "id" {

    type = uuid

    default = sql("gen_random_uuid()")

  }
  column "user_id" {
    type = uuid
    null = false
  }
  column "plan_date" {
    type = date
    null = false
  }
  column "status" {
    type = enum.daily_planning_session_status
    null = false
  }
  column "preview_json" {
    type = jsonb
    null = true
  }
  column "confirmed_plan_revision_id" {
    type = uuid
    null = true
  }
  column "model_id" {
    type    = text
    null    = false
    default = ""
  }
  column "prompt_version" {
    type    = text
    null    = false
    default = ""
  }
  column "created_at" {
    type    = timestamptz
    default = sql("now()")
  }
  column "updated_at" {
    type    = timestamptz
    default = sql("now()")
  }
  column "finalized_at" {
    type = timestamptz
    null = true
  }

  primary_key {

    columns = [column.id]

  }
  foreign_key "daily_planning_sessions_user_id_fkey" {
    columns     = [column.user_id]
    ref_columns = [table.users.column.id]
    on_delete   = CASCADE
  }
  foreign_key "daily_planning_sessions_confirmed_plan_revision_id_fkey" {
    columns     = [column.confirmed_plan_revision_id]
    ref_columns = [table.plan_revisions.column.id]
  }
  index "daily_planning_sessions_user_plan_date_active_key" {
    unique  = true
    columns = [column.user_id, column.plan_date]
    where   = "status IN ('gathering', 'context_blocked', 'chatting', 'reviewing')"
  }
  index "daily_planning_sessions_user_plan_date_idx" {
    columns = [column.user_id, column.plan_date]
  }
}

table "daily_planning_messages" {
  schema = schema.public

  column "id" {

    type = uuid

    default = sql("gen_random_uuid()")

  }
  column "session_id" {
    type = uuid
    null = false
  }
  column "sequence" {
    type = integer
    null = false
  }
  column "role" {
    type = enum.daily_planning_message_role
    null = false
  }
  column "content" {
    type = text
    null = false
  }
  column "created_at" {
    type    = timestamptz
    default = sql("now()")
  }

  primary_key {

    columns = [column.id]

  }
  foreign_key "daily_planning_messages_session_id_fkey" {
    columns     = [column.session_id]
    ref_columns = [table.daily_planning_sessions.column.id]
    on_delete   = CASCADE
  }
  check "daily_planning_messages_sequence_positive" {
    expr = "sequence > 0"
  }
  check "daily_planning_messages_content_not_blank" {
    expr = "btrim(content) <> ''"
  }
  index "daily_planning_messages_session_sequence_key" {
    unique  = true
    columns = [column.session_id, column.sequence]
  }
}

table "daily_planning_contexts" {
  schema = schema.public

  column "session_id" {

    type = uuid

    null = false

  }
  column "calendar_context" {
    type    = jsonb
    null    = false
    default = sql("'{}'::jsonb")
  }
  column "github_context" {
    type    = jsonb
    null    = false
    default = sql("'{}'::jsonb")
  }
  column "routine_context" {
    type    = jsonb
    null    = false
    default = sql("'{}'::jsonb")
  }
  column "nutrition_context" {
    type    = jsonb
    null    = false
    default = sql("'{}'::jsonb")
  }
  column "calendar_status" {
    type = text
    null = false
  }
  column "github_status" {
    type = text
    null = false
  }
  column "routine_status" {
    type = text
    null = false
  }
  column "nutrition_status" {
    type    = text
    null    = false
    default = "unavailable"
  }
  column "gathered_at" {
    type = timestamptz
    null = false
  }

  primary_key {

    columns = [column.session_id]

  }
  foreign_key "daily_planning_contexts_session_id_fkey" {
    columns     = [column.session_id]
    ref_columns = [table.daily_planning_sessions.column.id]
    on_delete   = CASCADE
  }
}

table "google_oauth_states" {
  schema = schema.public

  column "state_hash" {

    type = bytea

    null = false

  }
  column "user_id" {
    type = uuid
    null = false
  }
  column "encrypted_code_verifier" {
    type = bytea
    null = false
  }
  column "expires_at" {
    type = timestamptz
    null = false
  }
  column "created_at" {
    type    = timestamptz
    default = sql("now()")
  }

  primary_key {

    columns = [column.state_hash]

  }
  foreign_key "google_oauth_states_user_id_fkey" {
    columns     = [column.user_id]
    ref_columns = [table.users.column.id]
    on_delete   = CASCADE
  }
  index "google_oauth_states_expires_at_idx" {
    columns = [column.expires_at]
  }
}
