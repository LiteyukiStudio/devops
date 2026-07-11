package api

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/LiteyukiStudio/devops/internal/id"
	"github.com/LiteyukiStudio/devops/internal/model"
	"github.com/LiteyukiStudio/devops/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const developmentRateLimit = 10000

const (
	sessionDuration  = 24 * time.Hour
	rememberDuration = 30 * 24 * time.Hour
)

var (
	errRememberTokenInvalid = errors.New("remember token is invalid or expired")
	errRememberUserDisabled = errors.New("remember token user is unavailable")
)

func (h *Handlers) currentUser(ctx *gin.Context) (model.User, bool) {
	if strings.HasPrefix(strings.ToLower(ctx.GetHeader("Authorization")), "bearer ") {
		return h.currentUserFromAccessToken(ctx)
	}

	plainToken, err := ctx.Cookie(sessionCookieName)
	if err != nil {
		writeErrorKey(ctx, http.StatusUnauthorized, requestLanguage(ctx), "auth.session.missing")
		return model.User{}, false
	}

	var session model.UserSession
	err = h.db.First(&session, "token_hash = ? and expires_at > ?", hashToken(plainToken), time.Now()).Error
	if err != nil {
		clearSessionCookie(ctx)
		writeErrorKey(ctx, http.StatusUnauthorized, requestLanguage(ctx), "auth.session.expired")
		return model.User{}, false
	}

	var user model.User
	if err := h.db.First(&user, "id = ? and disabled = ?", session.UserID, false).Error; err != nil {
		clearSessionCookie(ctx)
		writeErrorKey(ctx, http.StatusUnauthorized, requestLanguage(ctx), "auth.account.disabled")
		return model.User{}, false
	}

	return user, true
}

func (h *Handlers) currentSessionFromCookie(ctx *gin.Context) (model.UserSession, bool) {
	plainToken, err := ctx.Cookie(sessionCookieName)
	if err != nil {
		return model.UserSession{}, false
	}

	var session model.UserSession
	err = h.db.First(&session, "token_hash = ? and expires_at > ?", hashToken(plainToken), time.Now()).Error
	if err != nil {
		return model.UserSession{}, false
	}

	return session, true
}

func (h *Handlers) currentUserFromAccessToken(ctx *gin.Context) (model.User, bool) {
	header := ctx.GetHeader("Authorization")
	if !strings.HasPrefix(strings.ToLower(header), "bearer ") {
		return model.User{}, false
	}

	plainToken := strings.TrimSpace(header[len("Bearer "):])
	var token model.AccessToken
	err := h.db.First(
		&token,
		"token_hash = ? and revoked_at is null and (expires_at is null or expires_at > ?)",
		hashToken(plainToken),
		time.Now(),
	).Error
	if err != nil || !accessTokenAllows(token.Scope, requiredScopeForRequest(ctx)) {
		writeError(ctx, http.StatusForbidden, "Access Token scope 不足或已失效")
		return model.User{}, false
	}

	var user model.User
	if err := h.db.First(&user, "id = ? and disabled = ?", token.UserID, false).Error; err != nil {
		writeErrorKey(ctx, http.StatusUnauthorized, requestLanguage(ctx), "auth.account.disabled")
		return model.User{}, false
	}

	return user, true
}

func requiredScopeForRequest(ctx *gin.Context) string {
	return service.RequiredAccessTokenScope(ctx.FullPath(), ctx.Request.Method)
}

func accessTokenAllows(scopeText, required string) bool {
	return service.AccessTokenAllows(scopeText, required)
}

func (h *Handlers) hasPlatformAdmin() bool {
	exists, err := platformAdminExists(h.db)
	return err == nil && exists
}

func platformAdminExists(db *gorm.DB) (bool, error) {
	var count int64
	if err := db.Model(&model.User{}).Where("role = ? and disabled = ?", "platform_admin", false).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (h *Handlers) requirePlatformAdmin(ctx *gin.Context) bool {
	user, ok := h.currentUser(ctx)
	if !ok {
		return false
	}
	if user.Role != "platform_admin" {
		writeErrorKey(ctx, http.StatusForbidden, user.Language, "config.admin.required")
		return false
	}
	return true
}

func (h *Handlers) createSession(ctx *gin.Context, userID string) bool {
	return h.createSessionWithImpersonation(ctx, userID, "")
}

func (h *Handlers) createSessionWithImpersonation(ctx *gin.Context, userID string, impersonatorID string) bool {
	session, plainToken := newUserSession(userID, impersonatorID, time.Now())
	if err := h.db.Create(&session).Error; err != nil {
		writeError(ctx, http.StatusInternalServerError, err.Error())
		return false
	}

	setSessionCookie(ctx, plainToken, h.mode == "production")
	return true
}

func newUserSession(userID, impersonatorID string, now time.Time) (model.UserSession, string) {
	plainToken := "sess_" + randomHex(32)
	return model.UserSession{
		ID:             id.New("ses"),
		UserID:         userID,
		ImpersonatorID: impersonatorID,
		TokenHash:      hashToken(plainToken),
		ExpiresAt:      now.Add(sessionDuration),
	}, plainToken
}

func newUserRememberToken(userID string, now time.Time) (model.UserRememberToken, string) {
	plainToken := "rem_" + randomHex(32)
	return model.UserRememberToken{
		ID:        id.New("rem"),
		UserID:    userID,
		TokenHash: hashToken(plainToken),
		ExpiresAt: now.Add(rememberDuration),
	}, plainToken
}

// Calls that omit requested default to no persistent login. This keeps older
// authentication flows secure until they expose an explicit remember choice.
func (h *Handlers) createRememberToken(ctx *gin.Context, userID string, requested ...bool) bool {
	if len(requested) == 0 || !requested[0] {
		return true
	}
	_ = h.db.Where("expires_at <= ?", time.Now()).Delete(&model.UserRememberToken{}).Error

	token, plainToken := newUserRememberToken(userID, time.Now())
	if err := h.db.Create(&token).Error; err != nil {
		writeError(ctx, http.StatusInternalServerError, err.Error())
		return false
	}

	setRememberCookie(ctx, userID, plainToken, h.mode == "production")
	return true
}

func (h *Handlers) rotateRememberLogin(userID, plainToken string) (model.User, string, string, error) {
	now := time.Now()
	newSession, newSessionToken := newUserSession(userID, "", now)
	newRemember, newRememberToken := newUserRememberToken(userID, now)
	var user model.User

	err := h.db.Transaction(func(tx *gorm.DB) error {
		var current model.UserRememberToken
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(
			&current,
			"token_hash = ? and user_id = ? and expires_at > ?",
			hashToken(plainToken),
			userID,
			now,
		).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errRememberTokenInvalid
			}
			return err
		}
		if err := tx.First(&user, "id = ? and disabled = ?", current.UserID, false).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errRememberUserDisabled
			}
			return err
		}
		result := tx.Where("id = ? and token_hash = ?", current.ID, current.TokenHash).Delete(&model.UserRememberToken{})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return errRememberTokenInvalid
		}
		if err := tx.Create(&newRemember).Error; err != nil {
			return err
		}
		return tx.Create(&newSession).Error
	})
	if err != nil {
		return model.User{}, "", "", err
	}
	return user, newSessionToken, newRememberToken, nil
}

func revokeUserAuthentication(tx *gorm.DB, userID string) error {
	if err := tx.Where("user_id = ?", userID).Delete(&model.StepUpAssertion{}).Error; err != nil {
		return err
	}
	if err := tx.Where("user_id = ?", userID).Delete(&model.UserRememberToken{}).Error; err != nil {
		return err
	}
	return tx.Where("user_id = ?", userID).Delete(&model.UserSession{}).Error
}

func (h *Handlers) revokeCurrentSessionAndRememberTokens(plainToken string) (string, error) {
	userID := ""
	err := h.db.Transaction(func(tx *gorm.DB) error {
		var session model.UserSession
		if err := tx.First(&session, "token_hash = ?", hashToken(plainToken)).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil
			}
			return err
		}
		userID = session.UserID
		if err := tx.Where("session_id = ?", session.ID).Delete(&model.StepUpAssertion{}).Error; err != nil {
			return err
		}
		if err := tx.Where("user_id = ?", session.UserID).Delete(&model.UserRememberToken{}).Error; err != nil {
			return err
		}
		return tx.Where("id = ?", session.ID).Delete(&model.UserSession{}).Error
	})
	return userID, err
}

func setSessionCookie(ctx *gin.Context, token string, secure bool) {
	ctx.SetSameSite(http.SameSiteLaxMode)
	ctx.SetCookie(sessionCookieName, token, 86400, "/", "", secure, true)
}

func setRememberCookie(ctx *gin.Context, userID string, token string, secure bool) {
	ctx.SetSameSite(http.SameSiteLaxMode)
	ctx.SetCookie(rememberCookieNameForUser(userID), token, 30*86400, "/", "", secure, true)
}

func clearSessionCookie(ctx *gin.Context) {
	ctx.SetSameSite(http.SameSiteLaxMode)
	ctx.SetCookie(sessionCookieName, "", -1, "/", "", false, true)
}

func clearRememberCookie(ctx *gin.Context, userID string) {
	if strings.TrimSpace(userID) == "" {
		return
	}
	ctx.SetSameSite(http.SameSiteLaxMode)
	ctx.SetCookie(rememberCookieNameForUser(userID), "", -1, "/", "", false, true)
}

func rememberCookieNameForUser(userID string) string {
	return rememberCookiePrefix + strings.NewReplacer("-", "_", ".", "_", ":", "_").Replace(userID)
}

type rateLimiter struct {
	redis *redis.Client
}

func newRateLimiter(redisAddr ...string) *rateLimiter {
	addr := ""
	if len(redisAddr) > 0 {
		addr = strings.TrimSpace(redisAddr[0])
	}
	return &rateLimiter{redis: redis.NewClient(&redis.Options{Addr: addr})}
}

func (l *rateLimiter) allow(key string, limit int, window time.Duration) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
	defer cancel()
	redisKey := "rate_limit:" + key
	count, err := l.redis.Incr(ctx, redisKey).Result()
	if err != nil {
		return false, err
	}
	if count == 1 {
		_ = l.redis.Expire(ctx, redisKey, window).Err()
	}
	return count <= int64(limit), nil
}

func (h *Handlers) allowSensitiveAuthAttempt(ctx *gin.Context, action string, limit int, window time.Duration) bool {
	if h.rateLimiter == nil {
		h.rateLimiter = newRateLimiter()
	}
	if h.mode == "development" && limit < developmentRateLimit {
		limit = developmentRateLimit
	}
	key := action + ":" + ctx.ClientIP()
	allowed, err := h.rateLimiter.allow(key, limit, window)
	if allowed {
		return true
	}
	if err != nil && h.mode == "development" {
		return true
	}
	writeError(ctx, http.StatusTooManyRequests, "请求过于频繁，请稍后再试")
	return false
}
