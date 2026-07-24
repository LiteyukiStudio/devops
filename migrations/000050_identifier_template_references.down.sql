UPDATE registry_credentials
SET repository_template = REPLACE(
      REPLACE(
        REPLACE(
          REPLACE(
            REPLACE(
              REPLACE(repository_template,
                '{projectIdentifier}', '{projectSlug}'),
              '${projectIdentifier}', '${projectSlug}'),
            '{appIdentifier}', '{appSlug}'),
          '${appIdentifier}', '${appSlug}'),
        '{applicationIdentifier}', '{applicationSlug}'),
      '${applicationIdentifier}', '${applicationSlug}'),
    tag_template = REPLACE(
      REPLACE(
        REPLACE(
          REPLACE(
            REPLACE(
              REPLACE(tag_template,
                '{projectIdentifier}', '{projectSlug}'),
              '${projectIdentifier}', '${projectSlug}'),
            '{appIdentifier}', '{appSlug}'),
          '${appIdentifier}', '${appSlug}'),
        '{applicationIdentifier}', '{applicationSlug}'),
      '${applicationIdentifier}', '${applicationSlug}');

UPDATE notification_templates
SET subject_template = REPLACE(
      REPLACE(
        REPLACE(subject_template,
          '.Event.Project.Identifier', '.Event.Project.Slug'),
        '.Event.Application.Identifier', '.Event.Application.Slug'),
      '.Event.DeploymentTarget.Identifier', '.Event.DeploymentTarget.Slug'),
    body_template = REPLACE(
      REPLACE(
        REPLACE(body_template,
          '.Event.Project.Identifier', '.Event.Project.Slug'),
        '.Event.Application.Identifier', '.Event.Application.Slug'),
      '.Event.DeploymentTarget.Identifier', '.Event.DeploymentTarget.Slug'),
    json_body_template = REPLACE(
      REPLACE(
        REPLACE(json_body_template,
          '.Event.Project.Identifier', '.Event.Project.Slug'),
        '.Event.Application.Identifier', '.Event.Application.Slug'),
      '.Event.DeploymentTarget.Identifier', '.Event.DeploymentTarget.Slug');

UPDATE notification_channels
SET config_json = REPLACE(
      REPLACE(
        REPLACE(config_json::text,
          '.Event.Project.Identifier', '.Event.Project.Slug'),
        '.Event.Application.Identifier', '.Event.Application.Slug'),
      '.Event.DeploymentTarget.Identifier', '.Event.DeploymentTarget.Slug')::jsonb;

UPDATE notification_deliveries
SET event_json = jsonb_set(
  event_json,
  '{project}',
  ((event_json->'project') - 'identifier'::text) ||
    jsonb_build_object('slug', COALESCE(event_json #>> '{project,slug}', event_json #>> '{project,identifier}', '')),
  true
)
WHERE jsonb_typeof(event_json->'project') = 'object'
  AND (event_json->'project') ? 'identifier';

UPDATE notification_deliveries
SET event_json = jsonb_set(
  event_json,
  '{application}',
  ((event_json->'application') - 'identifier'::text) ||
    jsonb_build_object('slug', COALESCE(event_json #>> '{application,slug}', event_json #>> '{application,identifier}', '')),
  true
)
WHERE jsonb_typeof(event_json->'application') = 'object'
  AND (event_json->'application') ? 'identifier';

UPDATE notification_deliveries
SET event_json = jsonb_set(
  event_json,
  '{deploymentTarget}',
  ((event_json->'deploymentTarget') - 'identifier'::text) ||
    jsonb_build_object('slug', COALESCE(event_json #>> '{deploymentTarget,slug}', event_json #>> '{deploymentTarget,identifier}', '')),
  true
)
WHERE jsonb_typeof(event_json->'deploymentTarget') = 'object'
  AND (event_json->'deploymentTarget') ? 'identifier';

UPDATE platform_events
SET detail_json = jsonb_set(
  detail_json,
  '{project}',
  ((detail_json->'project') - 'identifier'::text) ||
    jsonb_build_object('slug', COALESCE(detail_json #>> '{project,slug}', detail_json #>> '{project,identifier}', '')),
  true
)
WHERE jsonb_typeof(detail_json->'project') = 'object'
  AND (detail_json->'project') ? 'identifier';

UPDATE platform_events
SET detail_json = jsonb_set(
  detail_json,
  '{application}',
  ((detail_json->'application') - 'identifier'::text) ||
    jsonb_build_object('slug', COALESCE(detail_json #>> '{application,slug}', detail_json #>> '{application,identifier}', '')),
  true
)
WHERE jsonb_typeof(detail_json->'application') = 'object'
  AND (detail_json->'application') ? 'identifier';

UPDATE platform_events
SET detail_json = jsonb_set(
  detail_json,
  '{deploymentTarget}',
  ((detail_json->'deploymentTarget') - 'identifier'::text) ||
    jsonb_build_object('slug', COALESCE(detail_json #>> '{deploymentTarget,slug}', detail_json #>> '{deploymentTarget,identifier}', '')),
  true
)
WHERE jsonb_typeof(detail_json->'deploymentTarget') = 'object'
  AND (detail_json->'deploymentTarget') ? 'identifier';
