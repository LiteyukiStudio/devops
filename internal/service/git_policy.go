package service

import "github.com/LiteyukiStudio/devops/internal/model"

func CanUseGitAccount(user model.User, account model.GitAccount, userHasProject func(userID, projectID string) bool) bool {
	if NormalizeGitAccessScope(account.AccessScope) == "personal" {
		return account.UserID == user.ID
	}

	switch account.Scope {
	case "global":
		return true
	case "user":
		return account.OwnerRef == user.ID
	case "project":
		return user.Role == "platform_admin" || userHasProject(user.ID, account.OwnerRef)
	default:
		return false
	}
}

func CanUseGitProvider(user model.User, provider model.GitProvider, userHasProject func(userID, projectID string) bool) bool {
	switch provider.Scope {
	case "global":
		return true
	case "user":
		return provider.OwnerRef == user.ID
	case "project":
		return user.Role == "platform_admin" || userHasProject(user.ID, provider.OwnerRef)
	default:
		return false
	}
}

func NormalizeGitAccessScope(value string) string {
	switch value {
	case "provider":
		return "provider"
	default:
		return "personal"
	}
}
