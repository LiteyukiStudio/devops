UPDATE registry_credentials
SET repository_template = REPLACE(
      REPLACE(
        REPLACE(
          REPLACE(
            REPLACE(
              REPLACE(
                REPLACE(
                  REPLACE(repository_template,
                    '{projectSlug}', '{projectIdentifier}'),
                  '${projectSlug}', '${projectIdentifier}'),
                '{appSlug}', '{appIdentifier}'),
              '${appSlug}', '${appIdentifier}'),
            '{applicationSlug}', '{applicationIdentifier}'),
          '${applicationSlug}', '${applicationIdentifier}'),
        '{targetSlug}', '{target}'),
      '${targetSlug}', '${target}'),
    tag_template = REPLACE(
      REPLACE(
        REPLACE(
          REPLACE(
            REPLACE(
              REPLACE(
                REPLACE(
                  REPLACE(tag_template,
                    '{projectSlug}', '{projectIdentifier}'),
                  '${projectSlug}', '${projectIdentifier}'),
                '{appSlug}', '{appIdentifier}'),
              '${appSlug}', '${appIdentifier}'),
            '{applicationSlug}', '{applicationIdentifier}'),
          '${applicationSlug}', '${applicationIdentifier}'),
        '{targetSlug}', '{target}'),
      '${targetSlug}', '${target}');

UPDATE notification_templates
SET subject_template = REPLACE(
      REPLACE(
        REPLACE(subject_template,
          '.Event.Project.Slug', '.Event.Project.Identifier'),
        '.Event.Application.Slug', '.Event.Application.Identifier'),
      '.Event.DeploymentTarget.Slug', '.Event.DeploymentTarget.Identifier'),
    body_template = REPLACE(
      REPLACE(
        REPLACE(body_template,
          '.Event.Project.Slug', '.Event.Project.Identifier'),
        '.Event.Application.Slug', '.Event.Application.Identifier'),
      '.Event.DeploymentTarget.Slug', '.Event.DeploymentTarget.Identifier'),
    json_body_template = REPLACE(
      REPLACE(
        REPLACE(json_body_template,
          '.Event.Project.Slug', '.Event.Project.Identifier'),
        '.Event.Application.Slug', '.Event.Application.Identifier'),
      '.Event.DeploymentTarget.Slug', '.Event.DeploymentTarget.Identifier');

UPDATE notification_channels
SET config_json = REPLACE(
      REPLACE(
        REPLACE(config_json::text,
          '.Event.Project.Slug', '.Event.Project.Identifier'),
        '.Event.Application.Slug', '.Event.Application.Identifier'),
      '.Event.DeploymentTarget.Slug', '.Event.DeploymentTarget.Identifier')::jsonb;

UPDATE notification_deliveries
SET event_json = jsonb_set(
  event_json,
  '{project}',
  ((event_json->'project') - 'slug'::text) ||
    jsonb_build_object('identifier', COALESCE(event_json #>> '{project,identifier}', event_json #>> '{project,slug}', '')),
  true
)
WHERE jsonb_typeof(event_json->'project') = 'object'
  AND (event_json->'project') ? 'slug';

UPDATE notification_deliveries
SET event_json = jsonb_set(
  event_json,
  '{application}',
  ((event_json->'application') - 'slug'::text) ||
    jsonb_build_object('identifier', COALESCE(event_json #>> '{application,identifier}', event_json #>> '{application,slug}', '')),
  true
)
WHERE jsonb_typeof(event_json->'application') = 'object'
  AND (event_json->'application') ? 'slug';

UPDATE notification_deliveries
SET event_json = jsonb_set(
  event_json,
  '{deploymentTarget}',
  ((event_json->'deploymentTarget') - 'slug'::text) ||
    jsonb_build_object('identifier', COALESCE(event_json #>> '{deploymentTarget,identifier}', event_json #>> '{deploymentTarget,slug}', '')),
  true
)
WHERE jsonb_typeof(event_json->'deploymentTarget') = 'object'
  AND (event_json->'deploymentTarget') ? 'slug';

UPDATE platform_events
SET detail_json = jsonb_set(
  detail_json,
  '{project}',
  ((detail_json->'project') - 'slug'::text) ||
    jsonb_build_object('identifier', COALESCE(detail_json #>> '{project,identifier}', detail_json #>> '{project,slug}', '')),
  true
)
WHERE jsonb_typeof(detail_json->'project') = 'object'
  AND (detail_json->'project') ? 'slug';

UPDATE platform_events
SET detail_json = jsonb_set(
  detail_json,
  '{application}',
  ((detail_json->'application') - 'slug'::text) ||
    jsonb_build_object('identifier', COALESCE(detail_json #>> '{application,identifier}', detail_json #>> '{application,slug}', '')),
  true
)
WHERE jsonb_typeof(detail_json->'application') = 'object'
  AND (detail_json->'application') ? 'slug';

UPDATE platform_events
SET detail_json = jsonb_set(
  detail_json,
  '{deploymentTarget}',
  ((detail_json->'deploymentTarget') - 'slug'::text) ||
    jsonb_build_object('identifier', COALESCE(detail_json #>> '{deploymentTarget,identifier}', detail_json #>> '{deploymentTarget,slug}', '')),
  true
)
WHERE jsonb_typeof(detail_json->'deploymentTarget') = 'object'
  AND (detail_json->'deploymentTarget') ? 'slug';
