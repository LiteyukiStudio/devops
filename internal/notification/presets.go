package notification

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
			Description: "Feishu custom bot webhook.",
			AdapterKind: AdapterKindWebhook,
			ConfigTemplate: `{
  "method": "POST",
  "url": "https://open.feishu.cn/open-apis/bot/v2/hook/{{.Secrets.WebhookToken}}",
  "headers": {
    "Content-Type": "application/json"
  }
}`,
			JSONBodyTemplate: `{
  "msg_type": "text",
  "content": {
    "text": "[{{.Event.Severity}}] {{.Event.Type}}\n{{.Event.Message}}\nProject: {{.Event.Project.Name}}\nApplication: {{.Event.Application.Name}}"
  }
}`,
			SecretFields: []string{"WebhookToken"},
		},
		{
			ID:          "wecom-bot",
			Name:        "WeCom Bot",
			Description: "WeCom group bot webhook.",
			AdapterKind: AdapterKindWebhook,
			ConfigTemplate: `{
  "method": "POST",
  "url": "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key={{.Secrets.WebhookKey}}",
  "headers": {
    "Content-Type": "application/json"
  }
}`,
			JSONBodyTemplate: `{
  "msgtype": "text",
  "text": {
    "content": "[{{.Event.Severity}}] {{.Event.Type}}\n{{.Event.Message}}\nProject: {{.Event.Project.Name}}\nApplication: {{.Event.Application.Name}}"
  }
}`,
			SecretFields: []string{"WebhookKey"},
		},
	}
}
