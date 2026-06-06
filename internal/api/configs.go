package api

import (
	"net/http"
	"sync"
	"time"

	"github.com/LiteyukiStudio/devops/internal/model"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type configDefinition struct {
	Key         string `json:"key"`
	Label       string `json:"label"`
	Description string `json:"description"`
	Type        string `json:"type"`
	Public      bool   `json:"public"`
	Default     string `json:"default"`
}

var configDefinitions = []configDefinition{
	{
		Key:         "site.title",
		Label:       "网站标题",
		Description: "浏览器标题和控制台品牌名称。",
		Type:        "string",
		Public:      true,
		Default:     "Liteyuki DevOps",
	},
	{
		Key:         "site.logoUrl",
		Label:       "Logo 地址",
		Description: "控制台左上角 Logo 图片地址，留空时使用默认图标。",
		Type:        "string",
		Public:      true,
		Default:     "",
	},
	{
		Key:         "site.faviconUrl",
		Label:       "Favicon 地址",
		Description: "浏览器标签页图标地址，留空时使用默认 favicon。",
		Type:        "string",
		Public:      true,
		Default:     "",
	},
	{
		Key:         "site.loginSubtitle",
		Label:       "登录页副标题",
		Description: "登录页品牌下方的短说明。",
		Type:        "string",
		Public:      true,
		Default:     "使用本地账号登录控制台",
	},
}

type configCache struct {
	mu     sync.RWMutex
	values map[string]string
}

func newConfigCache(db *gorm.DB) *configCache {
	cache := &configCache{values: map[string]string{}}
	cache.reload(db)
	return cache
}

func (c *configCache) reload(db *gorm.DB) {
	values := map[string]string{}
	for _, definition := range configDefinitions {
		values[definition.Key] = definition.Default
	}

	var rows []model.AppConfig
	if err := db.Find(&rows).Error; err == nil {
		for _, row := range rows {
			values[row.Key] = row.Value
		}
	}

	c.mu.Lock()
	c.values = values
	c.mu.Unlock()
}

func (c *configCache) get(keys []string) map[string]string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := map[string]string{}
	for _, key := range keys {
		if value, ok := c.values[key]; ok {
			result[key] = value
		}
	}
	return result
}

func (c *configCache) set(key, value string) {
	c.mu.Lock()
	c.values[key] = value
	c.mu.Unlock()
}

func (h *Handlers) GetPublicConfigs(ctx *gin.Context) {
	var input configKeysInput
	if !bindJSON(ctx, &input) {
		return
	}
	ctx.JSON(http.StatusOK, h.configs.get(publicConfigKeys(input.Keys)))
}

func (h *Handlers) ListConfigDefinitions(ctx *gin.Context) {
	if _, ok := h.currentUser(ctx); !ok {
		return
	}

	ctx.JSON(http.StatusOK, configDefinitions)
}

func (h *Handlers) UpdateConfigs(ctx *gin.Context) {
	user, ok := h.currentUser(ctx)
	if !ok {
		return
	}
	if user.Role != "platform_admin" {
		writeErrorKey(ctx, http.StatusForbidden, user.Language, "config.admin.required")
		return
	}

	var input updateConfigsInput
	if !bindJSON(ctx, &input) {
		return
	}

	for key, value := range input.Values {
		if !isKnownConfigKey(key) {
			continue
		}
		row := model.AppConfig{Key: key, Value: value, UpdatedAt: time.Now()}
		if err := h.db.Save(&row).Error; err != nil {
			writeError(ctx, http.StatusBadRequest, err.Error())
			return
		}
		h.configs.set(key, value)
	}

	ctx.JSON(http.StatusOK, h.configs.get(knownConfigKeys()))
}

func publicConfigKeys(keys []string) []string {
	allowed := map[string]bool{}
	for _, definition := range configDefinitions {
		if definition.Public {
			allowed[definition.Key] = true
		}
	}

	result := make([]string, 0, len(keys))
	for _, key := range keys {
		if allowed[key] {
			result = append(result, key)
		}
	}
	return result
}

func knownConfigKeys() []string {
	keys := make([]string, 0, len(configDefinitions))
	for _, definition := range configDefinitions {
		keys = append(keys, definition.Key)
	}
	return keys
}

func isKnownConfigKey(key string) bool {
	for _, definition := range configDefinitions {
		if definition.Key == key {
			return true
		}
	}
	return false
}

type configKeysInput struct {
	Keys []string `json:"keys"`
}

type updateConfigsInput struct {
	Values map[string]string `json:"values"`
}
