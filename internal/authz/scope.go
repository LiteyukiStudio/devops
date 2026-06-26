package authz

var allowedAccessTokenScopes = map[string]bool{
	string(ActionProjectRead):  true,
	string(ActionProjectWrite): true,
	"project:*":                true,

	string(ActionApplicationRead):   true,
	string(ActionApplicationCreate): true,
	string(ActionApplicationUpdate): true,
	string(ActionApplicationDelete): true,
	"application:*":                 true,

	string(ActionDeploymentRead):     true,
	string(ActionDeploymentUpdate):   true,
	string(ActionDeploymentRelease):  true,
	string(ActionDeploymentRestart):  true,
	string(ActionDeploymentRollback): true,
	string(ActionDeploymentDelete):   true,
	string(ActionDeploymentExec):     true,
	"deployment:*":                   true,

	string(ActionBuildRead):    true,
	string(ActionBuildTrigger): true,
	string(ActionBuildCancel):  true,
	string(ActionBuildDelete):  true,
	"build:*":                  true,

	string(ActionGatewayRead):   true,
	string(ActionGatewayManage): true,
	"gateway:*":                 true,

	string(ActionGitRead):  true,
	string(ActionGitWrite): true,
	"git:*":                true,

	string(ActionRegistryRead):  true,
	string(ActionRegistryWrite): true,
	"registry:*":                true,

	string(ActionImageRead):  true,
	string(ActionImageWrite): true,
	"image:*":                true,

	string(ActionUserRead):   true,
	string(ActionUserWrite):  true,
	string(ActionUserManage): true,
	"user:*":                 true,

	string(ActionConfigRead):  true,
	string(ActionConfigWrite): true,
	"config:*":                true,

	string(ActionAuthManage):  true,
	string(ActionTokenManage): true,

	string(ActionBillingRead):   true,
	string(ActionBillingAdjust): true,
	"billing:*":                 true,
}

var userCreatableAccessTokenScopes = map[string]bool{
	string(ActionProjectRead):       true,
	string(ActionApplicationRead):   true,
	string(ActionDeploymentRead):    true,
	string(ActionDeploymentRelease): true,
	string(ActionBuildRead):         true,
	string(ActionBuildTrigger):      true,
	string(ActionGatewayRead):       true,
	string(ActionGitRead):           true,
	string(ActionRegistryRead):      true,
	string(ActionImageRead):         true,
	string(ActionUserRead):          true,
	string(ActionBillingRead):       true,
}
