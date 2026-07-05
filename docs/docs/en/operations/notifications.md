# Notifications

Notifications send platform events to external collaboration tools or email. This version starts with failure events so administrators are not flooded by every successful build, release, and route sync.

## How it works

The notification flow has four layers:

1. The platform emits a structured event when a build, release, hook, or access route fails.
2. Notification rules filter events by event type, severity, project space, application, and deployment target.
3. A matching rule creates delivery records, then worker sends them asynchronously.
4. A notification adapter renders the event into the request or email required by the target platform.

Business modules only emit events. They do not know about Feishu, WeCom, SMTP, or custom Webhook details.

## Channels

Supported channels:

- Webhook: custom `method`, `url`, `headers`, and JSON body templates for Feishu, WeCom, Slack-like bots, or internal alert endpoints.
- SMTP: standard SMTP, STARTTLS, or TLS email delivery.

Webhook channels restrict methods to `POST`, `PUT`, or `PATCH`, and validate target URLs with the platform public egress policy to avoid sending notifications to sensitive internal addresses. Channel secrets are stored in Secret Store. Business tables only keep secret references, and API responses only expose `secretSet`.

## Preset snapshots

The platform includes Feishu Bot and WeCom Bot presets. Creating a channel from a preset turns the preset into an ordinary Webhook channel and default template snapshot:

- You only provide the token or key required by the preset.
- Existing channels do not automatically follow future preset changes.
- To change the message format, edit the generated template or create another template.

## Template variables

Templates use Go template syntax with missing-field errors enabled, so typoed variables fail instead of silently sending empty content. Common variables:

| Variable | Description |
| --- | --- |
| `.Event.Type` | Event type, such as `build.failed`. |
| `.Event.Severity` | Severity, such as `error`. |
| `.Event.Message` | Failure summary. |
| `.Event.Project.Name` / `.Event.Project.Slug` | Project space name and slug. |
| `.Event.Application.Name` / `.Event.Application.Slug` | Application name and slug. |
| `.Event.DeploymentTarget.Name` / `.Event.DeploymentTarget.Slug` | Deployment target name and stage. |
| `.Event.Build.ID` / `.Event.Build.Image` / `.Event.Build.GitRef` | Build context. |
| `.Event.Release.ID` / `.Event.Release.ImageRef` / `.Event.Release.Revision` | Release context. |
| `.Event.Hook.Name` / `.Event.Hook.Phase` | Hook context. |
| `.Event.Gateway.Domain` / `.Event.Gateway.Path` | Access route context. |
| `.Secrets.<Name>` | Channel secret value injected only while rendering. It is not echoed by APIs. |

Available functions:

- `json`: encode a value as a JSON string.
- `time`: format a timestamp.
- `default`: use a fallback for empty values.
- `truncate`: shorten long text.

## Rules

Rules must select at least one channel. Supported failure events:

- `build.failed`
- `release.failed`
- `hook.failed`
- `gateway.apply_failed`

Filter JSON example:

```json
{
  "severities": ["error"],
  "projectIds": ["prj_xxx"],
  "applicationIds": [],
  "deploymentTargetIds": []
}
```

An empty array means that dimension is not filtered. The current UI manages global administrator rules first; project-space rules can be added later.

## SMTP example

Channel config JSON:

```json
{
  "host": "smtp.example.com",
  "port": 587,
  "security": "starttls",
  "username": "notice@example.com",
  "from": "Liteyuki DevOps <notice@example.com>",
  "to": ["ops@example.com"],
  "timeoutSeconds": 15
}
```

Secret key-values:

```text
password=email or SMTP password
```

The password is stored in Secret Store. You do not need to fill it again when editing; enter `password=...` only when replacing it.

## Verification

Suggested verification flow:

1. Create a Webhook or SMTP channel.
2. Click the test action and confirm that the target platform receives the test message.
3. Create or confirm a notification template.
4. Create a rule, select failure events and a channel.
5. Trigger a build failure or hook failure intentionally.
6. Check delivery records for status, attempts, errors, and the redacted request snapshot.
