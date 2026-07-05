package notification

import "encoding/json"

type WebhookPreset struct {
	ID               string   `json:"id"`
	Name             string   `json:"name"`
	Description      string   `json:"description"`
	AdapterKind      string   `json:"adapterKind"`
	ConfigTemplate   string   `json:"configTemplate"`
	JSONBodyTemplate string   `json:"jsonBodyTemplate"`
	SecretFields     []string `json:"secretFields"`
}

func WebhookPresets() []WebhookPreset {
	return []WebhookPreset{
		{
			ID:          "feishu-bot",
			Name:        "Feishu Bot",
			Description: "Feishu custom bot webhook with rich post content.",
			AdapterKind: AdapterKindWebhook,
			ConfigTemplate: webhookPresetConfig("POST", "https://open.feishu.cn/open-apis/bot/v2/hook/{{.Secrets.WebhookToken}}", map[string]string{
				"Content-Type": "application/json",
			}, feishuTextTestTemplate),
			JSONBodyTemplate: feishuPostTemplate("zh_cn", "项目空间", "应用", "部署配置", "时间"),
			SecretFields:     []string{"WebhookToken"},
		},
		{
			ID:          "lark-bot",
			Name:        "Lark Bot",
			Description: "Lark custom bot webhook with rich post content.",
			AdapterKind: AdapterKindWebhook,
			ConfigTemplate: webhookPresetConfig("POST", "https://open.larksuite.com/open-apis/bot/v2/hook/{{.Secrets.WebhookToken}}", map[string]string{
				"Content-Type": "application/json",
			}, feishuTextTestTemplate),
			JSONBodyTemplate: feishuPostTemplate("en_us", "Project", "Application", "Deployment", "Time"),
			SecretFields:     []string{"WebhookToken"},
		},
		{
			ID:          "wecom-bot",
			Name:        "WeCom Bot",
			Description: "WeCom group robot webhook with markdown content.",
			AdapterKind: AdapterKindWebhook,
			ConfigTemplate: webhookPresetConfig("POST", "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key={{.Secrets.WebhookKey}}", map[string]string{
				"Content-Type": "application/json",
			}, wecomMarkdownTestTemplate),
			JSONBodyTemplate: wecomMarkdownTemplate,
			SecretFields:     []string{"WebhookKey"},
		},
		{
			ID:          "gotify",
			Name:        "Gotify",
			Description: "Gotify application message API with markdown display extras.",
			AdapterKind: AdapterKindWebhook,
			ConfigTemplate: webhookPresetConfig("POST", "https://{{.Secrets.GotifyHost}}/message", map[string]string{
				"Content-Type": "application/json",
				"X-Gotify-Key": "{{.Secrets.AppToken}}",
			}, gotifyTestTemplate),
			JSONBodyTemplate: gotifyMarkdownTemplate,
			SecretFields:     []string{"GotifyHost", "AppToken"},
		},
		{
			ID:          "dingtalk-bot",
			Name:        "DingTalk Bot",
			Description: "DingTalk custom robot webhook with markdown content. Signed robots are not supported by this preset.",
			AdapterKind: AdapterKindWebhook,
			ConfigTemplate: webhookPresetConfig("POST", "https://oapi.dingtalk.com/robot/send?access_token={{.Secrets.AccessToken}}", map[string]string{
				"Content-Type": "application/json",
			}, dingtalkTextTestTemplate),
			JSONBodyTemplate: dingtalkMarkdownTemplate,
			SecretFields:     []string{"AccessToken"},
		},
		{
			ID:          "slack-webhook",
			Name:        "Slack Incoming Webhook",
			Description: "Slack incoming webhook with mrkdwn blocks.",
			AdapterKind: AdapterKindWebhook,
			ConfigTemplate: webhookPresetConfig("POST", "https://hooks.slack.com/services/{{.Secrets.WebhookPath}}", map[string]string{
				"Content-Type": "application/json",
			}, slackTextTestTemplate),
			JSONBodyTemplate: slackBlocksTemplate,
			SecretFields:     []string{"WebhookPath"},
		},
		{
			ID:          "discord-webhook",
			Name:        "Discord Webhook",
			Description: "Discord webhook with embeds.",
			AdapterKind: AdapterKindWebhook,
			ConfigTemplate: webhookPresetConfig("POST", "https://discord.com/api/webhooks/{{.Secrets.WebhookID}}/{{.Secrets.WebhookToken}}", map[string]string{
				"Content-Type": "application/json",
			}, discordTextTestTemplate),
			JSONBodyTemplate: discordEmbedTemplate,
			SecretFields:     []string{"WebhookID", "WebhookToken"},
		},
	}
}

func webhookPresetConfig(method string, url string, headers map[string]string, testJSONBodyTemplate string) string {
	data, _ := json.MarshalIndent(WebhookConfig{
		Method:               method,
		URL:                  url,
		Headers:              headers,
		Timeout:              15,
		TestJSONBodyTemplate: testJSONBodyTemplate,
	}, "", "  ")
	return string(data)
}

func feishuPostTemplate(locale string, projectLabel string, applicationLabel string, deploymentLabel string, timeLabel string) string {
	return `{
  "msg_type": "post",
  "content": {
    "post": {
      "` + locale + `": {
        "title": {{json (printf "[%s] %s" .Event.Severity .Event.Type)}},
        "content": [
          [
            {"tag": "text", "text": {{json .Event.Message}}}
          ],
          [
            {"tag": "text", "text": "` + projectLabel + `: "},
            {"tag": "text", "text": {{json (default "-" .Event.Project.Name)}}}
          ],
          [
            {"tag": "text", "text": "` + applicationLabel + `: "},
            {"tag": "text", "text": {{json (default "-" .Event.Application.Name)}}}
          ],
          [
            {"tag": "text", "text": "` + deploymentLabel + `: "},
            {"tag": "text", "text": {{json (default "-" .Event.DeploymentTarget.Name)}}}
          ],
          [
            {"tag": "text", "text": "` + timeLabel + `: "},
            {"tag": "text", "text": {{json (time .Event.OccurredAt "2006-01-02 15:04:05 MST")}}}
          ]
        ]
      }
    }
  }
}`
}

const feishuTextTestTemplate = `{
  "msg_type": "text",
  "content": {
    "text": {{json .Event.Message}}
  }
}`

const wecomMarkdownTestTemplate = `{
  "msgtype": "markdown",
  "markdown": {
    "content": {{json (printf "**%s**\n%s" .Event.Type .Event.Message)}}
  }
}`

const wecomMarkdownTemplate = `{
  "msgtype": "markdown",
  "markdown": {
    "content": {{json (printf "**[%s] %s**\n> %s\n> 项目空间：%s\n> 应用：%s\n> 部署配置：%s\n> 时间：%s" .Event.Severity .Event.Type .Event.Message (default "-" .Event.Project.Name) (default "-" .Event.Application.Name) (default "-" .Event.DeploymentTarget.Name) (time .Event.OccurredAt "2006-01-02 15:04:05 MST"))}}
  }
}`

const gotifyTestTemplate = `{
  "title": {{json .Event.Type}},
  "message": {{json .Event.Message}},
  "priority": 5,
  "extras": {
    "client::display": {
      "contentType": "text/markdown"
    }
  }
}`

const gotifyMarkdownTemplate = `{
  "title": {{json (printf "[%s] %s" .Event.Severity .Event.Type)}},
  "message": {{json (printf "**%s**\n\n- Project: %s\n- Application: %s\n- Deployment: %s\n- Time: %s" .Event.Message (default "-" .Event.Project.Name) (default "-" .Event.Application.Name) (default "-" .Event.DeploymentTarget.Name) (time .Event.OccurredAt "2006-01-02 15:04:05 MST"))}},
  "priority": 5,
  "extras": {
    "client::display": {
      "contentType": "text/markdown"
    }
  }
}`

const dingtalkTextTestTemplate = `{
  "msgtype": "text",
  "text": {
    "content": {{json .Event.Message}}
  }
}`

const dingtalkMarkdownTemplate = `{
  "msgtype": "markdown",
  "markdown": {
    "title": {{json (printf "[%s] %s" .Event.Severity .Event.Type)}},
    "text": {{json (printf "### [%s] %s\n\n%s\n\n- Project: %s\n- Application: %s\n- Deployment: %s\n- Time: %s" .Event.Severity .Event.Type .Event.Message (default "-" .Event.Project.Name) (default "-" .Event.Application.Name) (default "-" .Event.DeploymentTarget.Name) (time .Event.OccurredAt "2006-01-02 15:04:05 MST"))}}
  }
}`

const slackTextTestTemplate = `{
  "text": {{json .Event.Message}}
}`

const slackBlocksTemplate = `{
  "text": {{json (printf "[%s] %s: %s" .Event.Severity .Event.Type .Event.Message)}},
  "blocks": [
    {
      "type": "header",
      "text": {
        "type": "plain_text",
        "text": {{json (printf "[%s] %s" .Event.Severity .Event.Type)}}
      }
    },
    {
      "type": "section",
      "text": {
        "type": "mrkdwn",
        "text": {{json (printf "%s\n\n*Project:* %s\n*Application:* %s\n*Deployment:* %s\n*Time:* %s" .Event.Message (default "-" .Event.Project.Name) (default "-" .Event.Application.Name) (default "-" .Event.DeploymentTarget.Name) (time .Event.OccurredAt "2006-01-02 15:04:05 MST"))}}
      }
    }
  ]
}`

const discordTextTestTemplate = `{
  "content": {{json .Event.Message}}
}`

const discordEmbedTemplate = `{
  "embeds": [
    {
      "title": {{json (printf "[%s] %s" .Event.Severity .Event.Type)}},
      "description": {{json .Event.Message}},
      "color": 15158332,
      "fields": [
        {"name": "Project", "value": {{json (default "-" .Event.Project.Name)}}, "inline": true},
        {"name": "Application", "value": {{json (default "-" .Event.Application.Name)}}, "inline": true},
        {"name": "Deployment", "value": {{json (default "-" .Event.DeploymentTarget.Name)}}, "inline": true},
        {"name": "Time", "value": {{json (time .Event.OccurredAt "2006-01-02 15:04:05 MST")}}, "inline": false}
      ]
    }
  ]
}`
