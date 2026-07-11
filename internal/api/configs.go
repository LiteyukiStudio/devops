package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/LiteyukiStudio/devops/internal/model"
	"github.com/LiteyukiStudio/devops/internal/security"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type configDefinition struct {
	Key         string   `json:"key"`
	Label       string   `json:"-"`
	Description string   `json:"-"`
	Type        string   `json:"type"`
	Public      bool     `json:"public"`
	Default     string   `json:"default"`
	Options     []string `json:"options,omitempty"`
}

type configDefinitionResponse struct {
	Key            string   `json:"key"`
	LabelKey       string   `json:"labelKey"`
	DescriptionKey string   `json:"descriptionKey"`
	Type           string   `json:"type"`
	Public         bool     `json:"public"`
	Default        string   `json:"default"`
	Options        []string `json:"options,omitempty"`
}

var configDefinitions = []configDefinition{
	{
		Key:         "site.title",
		Label:       "网站标题",
		Description: "浏览器标题和控制台品牌名称。",
		Type:        "string",
		Public:      true,
		Default:     "Luna DevOps",
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
	{
		Key:         "site.operationsDashboardUrl",
		Label:       "运营面板地址",
		Description: "用于平台管理员查看运营大盘的 Grafana dashboard 或 panel iframe 地址。留空时不展示运营面板内容。",
		Type:        "string",
		Public:      false,
		Default:     "",
	},
	{
		Key:         "security.egress.domainAllowList",
		Label:       "SSRF 域名特许白名单",
		Description: "每行一个域名或通配符域名。命中后直接允许该域名，适合本地 FakeIP、内网镜像站等明确可信目标。",
		Type:        "textarea",
		Public:      false,
		Default:     "",
	},
	{
		Key:         "security.egress.domainBlockList",
		Label:       "SSRF 域名黑名单",
		Description: "每行一个域名或通配符域名。命中后直接拒绝访问。",
		Type:        "textarea",
		Public:      false,
		Default:     "",
	},
	{
		Key:         "security.egress.ipAllowList",
		Label:       "SSRF IP 白名单",
		Description: "每行一个 IP 或 CIDR。用于允许直连或解析结果命中的私网/保留地址。",
		Type:        "textarea",
		Public:      false,
		Default:     "",
	},
	{
		Key:         "security.egress.ipBlockList",
		Label:       "SSRF IP 黑名单",
		Description: "每行一个 IP 或 CIDR。用于拦截直连 IP 或非白名单域名的解析结果；域名白名单命中时不再二次检查 IP 黑名单。",
		Type:        "textarea",
		Public:      false,
		Default:     security.ReservedIPBlockListText(),
	},
	{
		Key:         "security.egress.allowedPorts",
		Label:       "SSRF 允许端口",
		Description: "可选。留空表示不限制端口；填写后每行一个端口，只允许这些端口。",
		Type:        "textarea",
		Public:      false,
		Default:     "",
	},
	{
		Key:         "security.stepUpMfa.enabled",
		Label:       "敏感操作二次验证",
		Description: "开启后，Web Console、运行命令、数据导出、密钥、镜像凭据、kubeconfig、身份源和用户管理等敏感操作需要当前会话完成短时二次验证。",
		Type:        "boolean",
		Public:      false,
		Default:     "false",
	},
	{
		Key:         "security.stepUpMfa.idleTimeoutMinutes",
		Label:       "二次验证空闲超时",
		Description: "完成二次验证后没有执行敏感操作的最长分钟数，超时后需要重新验证。",
		Type:        "number",
		Public:      false,
		Default:     "10",
	},
	{
		Key:         "security.stepUpMfa.absoluteTimeoutMinutes",
		Label:       "二次验证最长有效期",
		Description: "一次二次验证可以持续生效的最长分钟数，即使持续操作也不能超过该时间。",
		Type:        "number",
		Public:      false,
		Default:     "60",
	},
	{
		Key:         "billing.creditsDisplayName",
		Label:       "Credits 展示名称",
		Description: "控制台展示平台内部 credits 时使用的名称。底层仍统一按 credits 存储和结算。",
		Type:        "string",
		Public:      true,
		Default:     "Credits",
	},
	{
		Key:         "billing.fiatCurrencyUnit",
		Label:       "现实货币单位",
		Description: "平台管理员在账单概览中查看 credits 折算金额时使用的现实货币单位，例如 CNY、USD 或 元。",
		Type:        "string",
		Public:      true,
		Default:     "CNY",
	},
	{
		Key:         "billing.creditsPerFiatUnit",
		Label:       "每 1 现实货币对应 Credits",
		Description: "用于管理员账单概览展示换算金额。例：1000 表示 1 个现实货币单位可兑换 1000 credits。",
		Type:        "string",
		Public:      true,
		Default:     "1000",
	},
	{
		Key:         "billing.freeQuotaCredits",
		Label:       "默认免费额度",
		Description: "新用户钱包可获得的默认 credits 额度。当前用于后续充值与额度策略，已创建用户不会自动补发。",
		Type:        "string",
		Public:      false,
		Default:     "0",
	},
	{
		Key:         "billing.lowBalanceThresholdCredits",
		Label:       "低余额提醒阈值",
		Description: "计费归属人余额低于该 credits 数值时，后续可用于展示提醒或触发通知。",
		Type:        "string",
		Public:      false,
		Default:     "100",
	},
	{
		Key:         "billing.overdueGracePeriodHours",
		Label:       "欠费宽限期",
		Description: "计费归属人余额不足后允许继续运行的小时数。限制策略启用后会使用该值。",
		Type:        "string",
		Public:      false,
		Default:     "72",
	},
	{
		Key:         "billing.allowNegativeBalance",
		Label:       "允许欠费余额",
		Description: "是否允许账本扣到负余额。关闭后，后续限制策略会阻止新的付费操作。",
		Type:        "select",
		Public:      false,
		Default:     "true",
		Options:     []string{"true", "false"},
	},
	{
		Key:         "billing.blockNewBuildsWhenInsufficient",
		Label:       "余额不足阻止新构建",
		Description: "开启后，计费归属人余额不足时不再接受新的构建任务。已经开始的任务仍会完成结算。",
		Type:        "select",
		Public:      false,
		Default:     "false",
		Options:     []string{"true", "false"},
	},
	{
		Key:         "billing.blockDeployChangesWhenInsufficient",
		Label:       "余额不足阻止部署变更",
		Description: "开启后，计费归属人余额不足时会阻止新发布、扩容和新增数据卷等付费变更。",
		Type:        "select",
		Public:      false,
		Default:     "false",
		Options:     []string{"true", "false"},
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

func (h *Handlers) GetConfigs(ctx *gin.Context) {
	user, ok := h.currentUser(ctx)
	if !ok {
		return
	}
	if user.Role != "platform_admin" {
		writeErrorKey(ctx, http.StatusForbidden, user.Language, "config.admin.required")
		return
	}

	ctx.JSON(http.StatusOK, h.configs.get(knownConfigKeys()))
}

func (h *Handlers) ListConfigDefinitions(ctx *gin.Context) {
	if _, ok := h.currentUser(ctx); !ok {
		return
	}

	definitions := make([]configDefinitionResponse, 0, len(configDefinitions))
	for _, definition := range configDefinitions {
		definitions = append(definitions, configDefinitionResponse{
			Key:            definition.Key,
			LabelKey:       "settings.configDefinitions." + definition.Key + ".label",
			DescriptionKey: "settings.configDefinitions." + definition.Key + ".description",
			Type:           definition.Type,
			Public:         definition.Public,
			Default:        definition.Default,
			Options:        definition.Options,
		})
	}

	ctx.JSON(http.StatusOK, definitions)
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
	stepUpConfigChanged := containsStepUpConfig(input.Values)
	if stepUpConfigChanged {
		targetEnabled, _, _, err := h.validateStepUpConfigUpdate(input.Values)
		if err != nil {
			writeErrorCode(ctx, http.StatusBadRequest, "mfa.invalid_policy", err.Error())
			return
		}
		if targetEnabled && !h.hasMFAEnabledPlatformAdmin() {
			writeErrorCode(ctx, http.StatusConflict, "mfa.admin_enrollment_required", "至少一名可用平台管理员绑定 MFA 后才能开启全局二次验证")
			return
		}
		if (h.stepUpMFAEnabled() || targetEnabled) && !h.requireMFAAssertion(ctx, user, stepUpPurposeSecuritySettingsUpdate) {
			return
		}
	}

	for key, rawValue := range input.Values {
		if !isKnownConfigKey(key) {
			writeError(ctx, http.StatusBadRequest, fmt.Sprintf("unknown config key: %s", key))
			return
		}
		value, err := configValueToString(rawValue)
		if err != nil {
			writeError(ctx, http.StatusBadRequest, err.Error())
			return
		}
		if definition := configDefinitionByKey(key); definition != nil && len(definition.Options) > 0 && !configOptionAllowed(value, definition.Options) {
			writeError(ctx, http.StatusBadRequest, fmt.Sprintf("invalid config value for %s", key))
			return
		}
		row := model.AppConfig{Key: key, Value: value, UpdatedAt: time.Now()}
		if err := h.db.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "key"}},
			DoUpdates: clause.AssignmentColumns([]string{"value", "updated_at"}),
		}).Create(&row).Error; err != nil {
			writeError(ctx, http.StatusBadRequest, err.Error())
			return
		}
	}
	h.configs.reload(h.db)
	if stepUpConfigChanged {
		h.audit(user.ID, "mfa.policy_update", "security.stepUpMfa", true, "step-up MFA policy updated")
	}

	ctx.JSON(http.StatusOK, h.configs.get(knownConfigKeys()))
}

func containsStepUpConfig(values map[string]any) bool {
	for key := range values {
		if strings.HasPrefix(key, "security.stepUpMfa.") {
			return true
		}
	}
	return false
}

func (h *Handlers) validateStepUpConfigUpdate(values map[string]any) (bool, int, int, error) {
	enabledText, err := pendingConfigValue(values, "security.stepUpMfa.enabled", h.configValue("security.stepUpMfa.enabled"))
	if err != nil {
		return false, 0, 0, err
	}
	if !isBooleanConfigValue(enabledText) {
		return false, 0, 0, fmt.Errorf("security.stepUpMfa.enabled must be a boolean")
	}
	idleText, err := pendingConfigValue(values, "security.stepUpMfa.idleTimeoutMinutes", h.configValue("security.stepUpMfa.idleTimeoutMinutes"))
	if err != nil {
		return false, 0, 0, err
	}
	absoluteText, err := pendingConfigValue(values, "security.stepUpMfa.absoluteTimeoutMinutes", h.configValue("security.stepUpMfa.absoluteTimeoutMinutes"))
	if err != nil {
		return false, 0, 0, err
	}
	idleMinutes, err := configMinuteValue(idleText, int(defaultStepUpIdleTimeout/time.Minute), 1, 120)
	if err != nil {
		return false, 0, 0, fmt.Errorf("invalid idle timeout: %w", err)
	}
	absoluteMinutes, err := configMinuteValue(absoluteText, int(defaultStepUpAbsoluteTimeout/time.Minute), 5, 1440)
	if err != nil {
		return false, 0, 0, fmt.Errorf("invalid absolute timeout: %w", err)
	}
	if idleMinutes > absoluteMinutes {
		return false, 0, 0, fmt.Errorf("idle timeout cannot exceed absolute timeout")
	}
	return configBool(enabledText), idleMinutes, absoluteMinutes, nil
}

func pendingConfigValue(values map[string]any, key, current string) (string, error) {
	value, exists := values[key]
	if !exists {
		return current, nil
	}
	return configValueToString(value)
}

func configMinuteValue(value string, fallback, minimum, maximum int) (int, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback, nil
	}
	minutes, err := strconv.Atoi(value)
	if err != nil || minutes < minimum || minutes > maximum {
		return 0, fmt.Errorf("must be an integer from %d to %d", minimum, maximum)
	}
	return minutes, nil
}

func isBooleanConfigValue(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "true", "false", "1", "0", "yes", "no", "on", "off", "enabled", "disabled":
		return true
	default:
		return false
	}
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
	return configDefinitionByKey(key) != nil
}

func configDefinitionByKey(key string) *configDefinition {
	for _, definition := range configDefinitions {
		if definition.Key == key {
			return &definition
		}
	}
	return nil
}

func configOptionAllowed(value string, options []string) bool {
	for _, option := range options {
		if value == option {
			return true
		}
	}
	return false
}

func configValueToString(value any) (string, error) {
	switch typed := value.(type) {
	case nil:
		return "", nil
	case string:
		return typed, nil
	case bool:
		return fmt.Sprintf("%t", typed), nil
	case float64:
		return fmt.Sprintf("%v", typed), nil
	default:
		data, err := json.Marshal(typed)
		if err != nil {
			return "", err
		}
		return string(data), nil
	}
}

type configKeysInput struct {
	Keys []string `json:"keys"`
}

type updateConfigsInput struct {
	Values map[string]any `json:"values"`
}
