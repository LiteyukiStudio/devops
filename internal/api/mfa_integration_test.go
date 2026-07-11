package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/LiteyukiStudio/devops/internal/model"
	"github.com/LiteyukiStudio/devops/internal/secret"
	"github.com/gin-gonic/gin"
	"github.com/pquerna/otp/totp"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestMFAEnrollmentVerificationRecoveryAndDisableFlow(t *testing.T) {
	db := newMFAIntegrationDB(t)
	t.Setenv("APP_ENV", "development")
	t.Setenv("SECRET_ENCRYPTION_KEY", "mfa-integration-test-key")

	testSuffix := randomHex(4)
	user := model.User{ID: "usr_mfa_" + testSuffix, Email: "mfa-" + testSuffix + "@example.com", Name: "MFA User", Role: "platform_admin", Language: "zh-CN"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	sessionToken := "sess_mfa_integration_" + testSuffix
	session := model.UserSession{ID: "ses_mfa_" + testSuffix, UserID: user.ID, TokenHash: hashToken(sessionToken), ExpiresAt: time.Now().Add(time.Hour)}
	if err := db.Create(&session).Error; err != nil {
		t.Fatal(err)
	}
	handlers := &Handlers{db: db, configs: newConfigCache(db), mode: "development", rateLimiter: newRateLimiter()}
	handlers.secrets = secret.NewStore(db, handlers.audit)

	policyBlockedRecorder, policyBlockedContext := newMFAIntegrationContext(http.MethodPut, "/api/v1/configs", map[string]any{"values": map[string]any{"security.stepUpMfa.enabled": true}}, sessionToken)
	handlers.UpdateConfigs(policyBlockedContext)
	if policyBlockedRecorder.Code != http.StatusConflict {
		t.Fatalf("policy enabled without an MFA admin = %d %s", policyBlockedRecorder.Code, policyBlockedRecorder.Body.String())
	}

	statusRecorder, statusContext := newMFAIntegrationContext(http.MethodGet, "/api/v1/auth/mfa/status", nil, sessionToken)
	handlers.GetMFAStatus(statusContext)
	if statusRecorder.Code != http.StatusOK || jsonBool(t, statusRecorder.Body.Bytes(), "enabled") {
		t.Fatalf("initial status = %d %s", statusRecorder.Code, statusRecorder.Body.String())
	}

	enrollRecorder, enrollContext := newMFAIntegrationContext(http.MethodPost, "/api/v1/auth/mfa/totp/enroll", nil, sessionToken)
	handlers.EnrollMFA(enrollContext)
	if enrollRecorder.Code != http.StatusCreated {
		t.Fatalf("enroll = %d %s", enrollRecorder.Code, enrollRecorder.Body.String())
	}
	var enrollment struct {
		Secret        string `json:"secret"`
		OTPAuthURL    string `json:"otpauthUrl"`
		QRCodeDataURL string `json:"qrCodeDataUrl"`
	}
	if err := json.Unmarshal(enrollRecorder.Body.Bytes(), &enrollment); err != nil {
		t.Fatal(err)
	}
	if enrollment.Secret == "" || enrollment.OTPAuthURL == "" || enrollment.QRCodeDataURL == "" {
		t.Fatalf("incomplete enrollment response: %#v", enrollment)
	}
	var storedSecret model.SecretValue
	if err := db.First(&storedSecret, "resource = ?", mfaSecretResource(user.ID)).Error; err != nil {
		t.Fatal(err)
	}
	if storedSecret.CipherRef == "" || bytes.Contains([]byte(storedSecret.CipherRef), []byte(enrollment.Secret)) {
		t.Fatal("TOTP secret was not encrypted at rest")
	}

	confirmationCode, err := totp.GenerateCode(enrollment.Secret, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	confirmRecorder, confirmContext := newMFAIntegrationContext(http.MethodPost, "/api/v1/auth/mfa/totp/confirm", map[string]string{"code": confirmationCode}, sessionToken)
	handlers.ConfirmMFA(confirmContext)
	if confirmRecorder.Code != http.StatusOK {
		t.Fatalf("confirm = %d %s", confirmRecorder.Code, confirmRecorder.Body.String())
	}
	var initialRecovery struct {
		RecoveryCodes []string `json:"recoveryCodes"`
	}
	if err := json.Unmarshal(confirmRecorder.Body.Bytes(), &initialRecovery); err != nil {
		t.Fatal(err)
	}
	if len(initialRecovery.RecoveryCodes) != mfaRecoveryCodeCount {
		t.Fatalf("recovery code count = %d", len(initialRecovery.RecoveryCodes))
	}
	if !handlers.hasMFAEnabledPlatformAdmin() {
		t.Fatal("confirmed platform administrator was not recognized as MFA-enabled")
	}

	verificationCode, err := totp.GenerateCode(enrollment.Secret, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	securityVerifyRecorder, securityVerifyContext := newMFAIntegrationContext(http.MethodPost, "/api/v1/auth/mfa/verify", map[string]string{"purpose": stepUpPurposeSecuritySettingsUpdate, "code": verificationCode}, sessionToken)
	handlers.VerifyMFA(securityVerifyContext)
	if securityVerifyRecorder.Code != http.StatusOK {
		t.Fatalf("security settings verify = %d %s", securityVerifyRecorder.Code, securityVerifyRecorder.Body.String())
	}
	policyEnableRecorder, policyEnableContext := newMFAIntegrationContext(http.MethodPut, "/api/v1/configs", map[string]any{"values": map[string]any{"security.stepUpMfa.enabled": true}}, sessionToken)
	handlers.UpdateConfigs(policyEnableContext)
	if policyEnableRecorder.Code != http.StatusOK || !handlers.stepUpMFAEnabled() {
		t.Fatalf("policy enable = %d %s", policyEnableRecorder.Code, policyEnableRecorder.Body.String())
	}

	verifyRecorder, verifyContext := newMFAIntegrationContext(http.MethodPost, "/api/v1/auth/mfa/verify", map[string]string{"purpose": stepUpPurposeMFAManage, "code": verificationCode}, sessionToken)
	handlers.VerifyMFA(verifyContext)
	if verifyRecorder.Code != http.StatusOK {
		t.Fatalf("verify = %d %s", verifyRecorder.Code, verifyRecorder.Body.String())
	}

	regenerateRecorder, regenerateContext := newMFAIntegrationContext(http.MethodPost, "/api/v1/auth/mfa/recovery-codes", nil, sessionToken)
	handlers.RegenerateMFARecoveryCodes(regenerateContext)
	if regenerateRecorder.Code != http.StatusOK {
		t.Fatalf("regenerate = %d %s", regenerateRecorder.Code, regenerateRecorder.Body.String())
	}
	var regenerated struct {
		RecoveryCodes []string `json:"recoveryCodes"`
	}
	if err := json.Unmarshal(regenerateRecorder.Body.Bytes(), &regenerated); err != nil {
		t.Fatal(err)
	}
	if len(regenerated.RecoveryCodes) != mfaRecoveryCodeCount {
		t.Fatalf("regenerated recovery code count = %d", len(regenerated.RecoveryCodes))
	}

	recoveryCode := regenerated.RecoveryCodes[0]
	recoveryRecorder, recoveryContext := newMFAIntegrationContext(http.MethodPost, "/api/v1/auth/mfa/verify", map[string]string{"purpose": stepUpPurposeDataExport, "recoveryCode": recoveryCode}, sessionToken)
	handlers.VerifyMFA(recoveryContext)
	if recoveryRecorder.Code != http.StatusOK {
		t.Fatalf("recovery verify = %d %s", recoveryRecorder.Code, recoveryRecorder.Body.String())
	}
	assertionRecorder, assertionContext := newMFAIntegrationContext(http.MethodGet, "/sensitive", nil, sessionToken)
	if !handlers.requireMFAAssertion(assertionContext, user, stepUpPurposeDataExport) || assertionRecorder.Code != http.StatusOK {
		t.Fatalf("assertion refresh = %d %s", assertionRecorder.Code, assertionRecorder.Body.String())
	}

	reuseRecorder, reuseContext := newMFAIntegrationContext(http.MethodPost, "/api/v1/auth/mfa/verify", map[string]string{"purpose": stepUpPurposeRuntimeExec, "recoveryCode": recoveryCode}, sessionToken)
	handlers.VerifyMFA(reuseContext)
	if reuseRecorder.Code != http.StatusUnauthorized {
		t.Fatalf("reused recovery code = %d %s", reuseRecorder.Code, reuseRecorder.Body.String())
	}
	var usedCount int64
	if err := db.Model(&model.MFARecoveryCode{}).Where("user_id = ? and used_at is not null", user.ID).Count(&usedCount).Error; err != nil || usedCount != 1 {
		t.Fatalf("used recovery codes = %d, err=%v", usedCount, err)
	}
	blockedDisableRecorder, blockedDisableContext := newMFAIntegrationContext(http.MethodDelete, "/api/v1/auth/mfa", nil, sessionToken)
	handlers.DisableMFA(blockedDisableContext)
	if blockedDisableRecorder.Code != http.StatusConflict {
		t.Fatalf("last MFA admin disable while policy enabled = %d %s", blockedDisableRecorder.Code, blockedDisableRecorder.Body.String())
	}

	policyDisableRecorder, policyDisableContext := newMFAIntegrationContext(http.MethodPut, "/api/v1/configs", map[string]any{"values": map[string]any{"security.stepUpMfa.enabled": false}}, sessionToken)
	handlers.UpdateConfigs(policyDisableContext)
	if policyDisableRecorder.Code != http.StatusOK || handlers.stepUpMFAEnabled() {
		t.Fatalf("policy disable = %d %s", policyDisableRecorder.Code, policyDisableRecorder.Body.String())
	}

	disableRecorder, disableContext := newMFAIntegrationContext(http.MethodDelete, "/api/v1/auth/mfa", nil, sessionToken)
	handlers.DisableMFA(disableContext)
	disableContext.Writer.WriteHeaderNow()
	if disableRecorder.Code != http.StatusNoContent {
		t.Fatalf("disable = %d %s", disableRecorder.Code, disableRecorder.Body.String())
	}
	var configCount, assertionCount int64
	_ = db.Model(&model.UserMFAConfig{}).Where("user_id = ?", user.ID).Count(&configCount).Error
	_ = db.Model(&model.StepUpAssertion{}).Where("user_id = ?", user.ID).Count(&assertionCount).Error
	if configCount != 0 || assertionCount != 0 {
		t.Fatalf("MFA state remained after disable: configs=%d assertions=%d", configCount, assertionCount)
	}

	var auditLogs []model.AuditLog
	if err := db.Where("user_id = ?", user.ID).Find(&auditLogs).Error; err != nil {
		t.Fatal(err)
	}
	requiredAuditActions := map[string]bool{
		"mfa.enroll":                    false,
		"mfa.confirm":                   false,
		"mfa.verify":                    false,
		"mfa.recovery_codes_regenerate": false,
		"mfa.recovery_code_used":        false,
		"mfa.policy_update":             false,
		"mfa.disable":                   false,
	}
	for _, entry := range auditLogs {
		if _, required := requiredAuditActions[entry.Action]; required {
			requiredAuditActions[entry.Action] = true
		}
		for _, sensitiveValue := range []string{enrollment.Secret, confirmationCode, recoveryCode} {
			if sensitiveValue != "" && strings.Contains(entry.Message, sensitiveValue) {
				t.Fatalf("audit log %s leaked an MFA credential", entry.Action)
			}
		}
	}
	for action, found := range requiredAuditActions {
		if !found {
			t.Fatalf("missing MFA audit action %q; logs=%#v", action, auditLogs)
		}
	}
}

func newMFAIntegrationDB(t *testing.T) *gorm.DB {
	t.Helper()
	databaseURL := os.Getenv("AUTH_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("AUTH_TEST_DATABASE_URL is not configured")
	}
	adminDB, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{})
	if err != nil {
		t.Fatalf("open integration database: %v", err)
	}
	schema := fmt.Sprintf("mfa_api_test_%d", time.Now().UnixNano())
	if err := adminDB.Exec(`CREATE SCHEMA "` + schema + `"`).Error; err != nil {
		t.Fatalf("create integration schema: %v", err)
	}

	parsedURL, err := url.Parse(databaseURL)
	if err != nil {
		t.Fatalf("parse integration database URL: %v", err)
	}
	query := parsedURL.Query()
	query.Set("search_path", schema)
	parsedURL.RawQuery = query.Encode()
	db, err := gorm.Open(postgres.Open(parsedURL.String()), &gorm.Config{})
	if err != nil {
		t.Fatalf("open integration schema: %v", err)
	}
	if err := db.AutoMigrate(
		&model.User{},
		&model.UserSession{},
		&model.UserMFAConfig{},
		&model.MFARecoveryCode{},
		&model.StepUpAssertion{},
		&model.SecretValue{},
		&model.AuditLog{},
		&model.AppConfig{},
	); err != nil {
		t.Fatalf("migrate integration schema: %v", err)
	}
	t.Cleanup(func() {
		if sqlDB, dbErr := db.DB(); dbErr == nil {
			_ = sqlDB.Close()
		}
		_ = adminDB.Exec(`DROP SCHEMA IF EXISTS "` + schema + `" CASCADE`).Error
		if sqlDB, dbErr := adminDB.DB(); dbErr == nil {
			_ = sqlDB.Close()
		}
	})
	return db
}

func newMFAIntegrationContext(method, path string, body any, sessionToken string) (*httptest.ResponseRecorder, *gin.Context) {
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	var payload []byte
	if body != nil {
		payload, _ = json.Marshal(body)
	}
	ctx.Request = httptest.NewRequest(method, path, bytes.NewReader(payload))
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Request.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sessionToken})
	return recorder, ctx
}

func jsonBool(t *testing.T, data []byte, key string) bool {
	t.Helper()
	var body map[string]any
	if err := json.Unmarshal(data, &body); err != nil {
		t.Fatal(err)
	}
	value, _ := body[key].(bool)
	return value
}
