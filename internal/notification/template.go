package notification

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"text/template"
	"time"
)

const maxRenderedTemplateBytes = 64 * 1024

func renderTemplate(name string, source string, data any) (string, error) {
	source = strings.TrimSpace(source)
	if source == "" {
		return "", nil
	}
	tpl, err := template.New(name).Funcs(template.FuncMap{
		"default":  templateDefault,
		"json":     templateJSON,
		"time":     templateTime,
		"truncate": templateTruncate,
	}).Option("missingkey=error").Parse(source)
	if err != nil {
		return "", err
	}
	var out bytes.Buffer
	if err := tpl.Execute(&out, data); err != nil {
		return "", err
	}
	if out.Len() > maxRenderedTemplateBytes {
		return "", fmt.Errorf("rendered template exceeds %d bytes", maxRenderedTemplateBytes)
	}
	return out.String(), nil
}

func renderMessage(event Event, tpl Template, secrets map[string]string) (RenderedMessage, error) {
	data := templateData{Event: event, Secrets: secrets}
	subject, err := renderTemplate("subject", tpl.Subject, data)
	if err != nil {
		return RenderedMessage{}, fmt.Errorf("render subject: %w", err)
	}
	body, err := renderTemplate("body", tpl.Body, data)
	if err != nil {
		return RenderedMessage{}, fmt.Errorf("render body: %w", err)
	}
	jsonBody, err := renderTemplate("json", tpl.JSON, data)
	if err != nil {
		return RenderedMessage{}, fmt.Errorf("render json: %w", err)
	}
	message := RenderedMessage{Subject: subject, Body: body}
	if strings.TrimSpace(jsonBody) != "" {
		var normalized any
		if err := json.Unmarshal([]byte(jsonBody), &normalized); err != nil {
			return RenderedMessage{}, fmt.Errorf("rendered body is not valid json: %w", err)
		}
		data, err := json.Marshal(normalized)
		if err != nil {
			return RenderedMessage{}, err
		}
		message.JSON = data
	}
	return message, nil
}

type templateData struct {
	Event   Event
	Secrets map[string]string
}

func templateDefault(fallback string, value any) string {
	text := strings.TrimSpace(fmt.Sprint(value))
	if text == "" || text == "<nil>" {
		return fallback
	}
	return text
}

func templateJSON(value any) (string, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func templateTime(value any, layout string) string {
	if layout == "" {
		layout = time.RFC3339
	}
	switch item := value.(type) {
	case time.Time:
		if item.IsZero() {
			return ""
		}
		return item.Format(layout)
	case *time.Time:
		if item == nil || item.IsZero() {
			return ""
		}
		return item.Format(layout)
	default:
		return ""
	}
}

func templateTruncate(value any, max int) string {
	text := fmt.Sprint(value)
	if max <= 0 || len([]rune(text)) <= max {
		return text
	}
	runes := []rune(text)
	return string(runes[:max])
}
