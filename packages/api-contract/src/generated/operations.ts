// Generated from openapi/openapi.yaml. Do not edit manually.
import type { OpenApiOperationSnapshot, OpenApiSnapshotMetadata } from "../types.js";

export const OPENAPI_SNAPSHOT_METADATA = {
  "source": "openapi/openapi.yaml",
  "openapiVersion": "3.1.0",
  "apiVersion": "0.1.0",
  "sourceDigest": "sha256:816aca50a4e5bdd1c43d3b3368cb4d53755dd4cbaf9864b09e8643ec060e8c5a",
  "operationCount": 109
} as const satisfies OpenApiSnapshotMetadata;

export const OPENAPI_OPERATION_SNAPSHOTS = [
  {
    "method": "get",
    "path": "/healthz",
    "tags": [
      "Health"
    ],
    "deprecated": false,
    "security": [],
    "parameters": [],
    "responses": [
      {
        "status": "200",
        "contentTypes": [],
        "schemaRefs": [],
        "description": "Service is healthy."
      }
    ],
    "summary": "Health check"
  },
  {
    "method": "get",
    "path": "/api/v1/meta",
    "tags": [
      "Health"
    ],
    "deprecated": false,
    "security": [],
    "parameters": [],
    "responses": [
      {
        "status": "200",
        "contentTypes": [
          "application/json"
        ],
        "schemaRefs": [],
        "description": "Public API compatibility and feature metadata."
      }
    ],
    "summary": "Get API version and CLI capability metadata",
    "operationId": "getApiMeta"
  },
  {
    "method": "post",
    "path": "/api/v1/public/configs",
    "tags": [
      "Configs"
    ],
    "deprecated": false,
    "security": [],
    "parameters": [],
    "responses": [
      {
        "status": "200",
        "contentTypes": [],
        "schemaRefs": [],
        "description": "Public config dictionary."
      }
    ],
    "summary": "Get public app configs by keys",
    "requestBody": {
      "required": true,
      "contentTypes": [
        "application/json"
      ],
      "schemaRefs": [
        "#/components/schemas/ConfigKeysInput"
      ]
    }
  },
  {
    "method": "get",
    "path": "/api/v1/build/templates",
    "tags": [
      "Builds"
    ],
    "deprecated": false,
    "security": [],
    "parameters": [],
    "responses": [
      {
        "status": "200",
        "contentTypes": [
          "application/json"
        ],
        "schemaRefs": [
          "#/components/schemas/BuildTemplate"
        ],
        "description": "Built-in template catalog."
      }
    ],
    "summary": "List immutable platform build templates"
  },
  {
    "method": "post",
    "path": "/api/v1/build/templates/{templateId}/preview",
    "tags": [
      "Builds"
    ],
    "deprecated": false,
    "security": [],
    "parameters": [
      {
        "name": "templateId",
        "in": "path",
        "required": true,
        "schema": {
          "type": "string"
        }
      }
    ],
    "responses": [
      {
        "status": "200",
        "contentTypes": [
          "application/json"
        ],
        "schemaRefs": [
          "#/components/schemas/BuildTemplatePreview"
        ],
        "description": "Rendered, immutable build definition preview."
      }
    ],
    "summary": "Validate template parameters and preview the generated Dockerfile",
    "requestBody": {
      "required": true,
      "contentTypes": [
        "application/json"
      ],
      "schemaRefs": [
        "#/components/schemas/BuildTemplatePreviewInput"
      ]
    }
  },
  {
    "method": "get",
    "path": "/api/v1/build/environment-config",
    "tags": [
      "Builds"
    ],
    "deprecated": false,
    "security": [],
    "parameters": [
      {
        "name": "scope",
        "in": "query",
        "required": true,
        "ref": "#/components/parameters/BuildEnvironmentScope",
        "schema": {
          "type": "string",
          "enum": [
            "global",
            "application",
            "deployment"
          ]
        }
      },
      {
        "name": "projectId",
        "in": "query",
        "schema": {
          "type": "string"
        }
      },
      {
        "name": "applicationId",
        "in": "query",
        "schema": {
          "type": "string"
        }
      },
      {
        "name": "deploymentTargetId",
        "in": "query",
        "schema": {
          "type": "string"
        }
      }
    ],
    "responses": [
      {
        "status": "200",
        "contentTypes": [
          "application/json"
        ],
        "schemaRefs": [
          "#/components/schemas/BuildEnvironmentConfig"
        ],
        "description": "Public values and boolean secret presence. Secret values and references are never returned."
      }
    ],
    "summary": "Get one global, application, or deployment build environment"
  },
  {
    "method": "put",
    "path": "/api/v1/build/environment-config",
    "tags": [
      "Builds"
    ],
    "deprecated": false,
    "security": [],
    "parameters": [
      {
        "name": "scope",
        "in": "query",
        "required": true,
        "ref": "#/components/parameters/BuildEnvironmentScope",
        "schema": {
          "type": "string",
          "enum": [
            "global",
            "application",
            "deployment"
          ]
        }
      },
      {
        "name": "projectId",
        "in": "query",
        "schema": {
          "type": "string"
        }
      },
      {
        "name": "applicationId",
        "in": "query",
        "schema": {
          "type": "string"
        }
      },
      {
        "name": "deploymentTargetId",
        "in": "query",
        "schema": {
          "type": "string"
        }
      }
    ],
    "responses": [
      {
        "status": "200",
        "contentTypes": [
          "application/json"
        ],
        "schemaRefs": [
          "#/components/schemas/BuildEnvironmentConfig"
        ],
        "description": "Updated build environment with secret presence only."
      }
    ],
    "summary": "Replace one global, application, or deployment build environment",
    "requestBody": {
      "required": true,
      "contentTypes": [
        "application/json"
      ],
      "schemaRefs": [
        "#/components/schemas/BuildEnvironmentConfigInput"
      ]
    }
  },
  {
    "method": "get",
    "path": "/api/v1/auth/bootstrap",
    "tags": [
      "Auth"
    ],
    "deprecated": false,
    "security": [],
    "parameters": [],
    "responses": [
      {
        "status": "200",
        "contentTypes": [
          "application/json"
        ],
        "schemaRefs": [
          "#/components/schemas/BootstrapStatus"
        ],
        "description": "Bootstrap status. devLoginHint is returned only in development mode."
      }
    ],
    "summary": "Get bootstrap and runtime mode status"
  },
  {
    "method": "post",
    "path": "/api/v1/auth/bootstrap/admin",
    "tags": [
      "Auth"
    ],
    "deprecated": false,
    "security": [],
    "parameters": [],
    "responses": [
      {
        "status": "201",
        "contentTypes": [
          "application/json"
        ],
        "schemaRefs": [
          "#/components/schemas/AuthSessionResponse"
        ],
        "description": "Created platform admin and session."
      },
      {
        "status": "400",
        "contentTypes": [
          "application/json"
        ],
        "schemaRefs": [
          "#/components/schemas/ErrorResponse"
        ],
        "description": "Invalid email, password, language, or JSON (`bootstrap.invalid_input` or `request.invalid_json`)."
      },
      {
        "status": "403",
        "contentTypes": [
          "application/json"
        ],
        "schemaRefs": [
          "#/components/schemas/ErrorResponse"
        ],
        "description": "The production bootstrap token is invalid (`bootstrap.token_invalid`)."
      },
      {
        "status": "409",
        "contentTypes": [
          "application/json"
        ],
        "schemaRefs": [
          "#/components/schemas/ErrorResponse"
        ],
        "description": "Platform admin already exists."
      },
      {
        "status": "503",
        "contentTypes": [
          "application/json"
        ],
        "schemaRefs": [
          "#/components/schemas/ErrorResponse"
        ],
        "description": "Production bootstrap is unavailable because `BOOTSTRAP_TOKEN` is not configured (`bootstrap.unavailable`)."
      }
    ],
    "summary": "Initialize the first platform admin",
    "requestBody": {
      "required": true,
      "contentTypes": [
        "application/json"
      ],
      "schemaRefs": [
        "#/components/schemas/InitializeAdminInput"
      ]
    }
  },
  {
    "method": "post",
    "path": "/api/v1/auth/login",
    "tags": [
      "Auth"
    ],
    "deprecated": false,
    "security": [],
    "parameters": [],
    "responses": [
      {
        "status": "200",
        "contentTypes": [
          "application/json"
        ],
        "schemaRefs": [
          "#/components/schemas/AuthSessionResponse"
        ],
        "description": "Login succeeded."
      },
      {
        "status": "401",
        "contentTypes": [
          "application/json"
        ],
        "schemaRefs": [
          "#/components/schemas/ErrorResponse"
        ],
        "description": "Login failed."
      }
    ],
    "summary": "Login with a local account",
    "requestBody": {
      "required": true,
      "contentTypes": [
        "application/json"
      ],
      "schemaRefs": [
        "#/components/schemas/LoginInput"
      ]
    }
  },
  {
    "method": "post",
    "path": "/api/v1/auth/login/resume",
    "tags": [
      "Auth"
    ],
    "deprecated": false,
    "security": [],
    "parameters": [],
    "responses": [
      {
        "status": "200",
        "contentTypes": [
          "application/json"
        ],
        "schemaRefs": [
          "#/components/schemas/AuthSessionResponse"
        ],
        "description": "Remembered login resumed."
      },
      {
        "status": "401",
        "contentTypes": [
          "application/json"
        ],
        "schemaRefs": [
          "#/components/schemas/ErrorResponse"
        ],
        "description": "Remember token missing, expired, revoked, or the account is disabled."
      }
    ],
    "summary": "Resume login with a remembered account",
    "description": "Rotates the per-user remember token, creates a new 24-hour session, and refreshes the 30-day remember cookie. Browser cookies are required.",
    "requestBody": {
      "required": true,
      "contentTypes": [
        "application/json"
      ],
      "schemaRefs": [
        "#/components/schemas/ResumeLoginInput"
      ]
    }
  },
  {
    "method": "post",
    "path": "/api/v1/auth/logout",
    "tags": [
      "Auth"
    ],
    "deprecated": false,
    "security": [],
    "parameters": [],
    "responses": [
      {
        "status": "204",
        "contentTypes": [],
        "schemaRefs": [],
        "description": "Logged out."
      }
    ],
    "summary": "Logout current session"
  },
  {
    "method": "get",
    "path": "/api/v1/auth/registration",
    "tags": [
      "Auth"
    ],
    "deprecated": false,
    "security": [],
    "parameters": [],
    "responses": [
      {
        "status": "200",
        "contentTypes": [
          "application/json"
        ],
        "schemaRefs": [
          "#/components/schemas/AuthRegistrationStatus"
        ],
        "description": "Public registration capability flags."
      }
    ],
    "summary": "Get public registration capabilities"
  },
  {
    "method": "post",
    "path": "/api/v1/auth/registration/email/code",
    "tags": [
      "Auth"
    ],
    "deprecated": false,
    "security": [],
    "parameters": [],
    "responses": [
      {
        "status": "202",
        "contentTypes": [],
        "schemaRefs": [],
        "description": "Verification challenge created."
      }
    ],
    "summary": "Request an email registration verification code",
    "requestBody": {
      "required": true,
      "contentTypes": [
        "application/json"
      ],
      "schemaRefs": [
        "#/components/schemas/EmailRegistrationCodeInput"
      ]
    }
  },
  {
    "method": "post",
    "path": "/api/v1/auth/registration/email",
    "tags": [
      "Auth"
    ],
    "deprecated": false,
    "security": [],
    "parameters": [],
    "responses": [
      {
        "status": "201",
        "contentTypes": [],
        "schemaRefs": [],
        "description": "Account created and signed in."
      }
    ],
    "summary": "Complete email registration",
    "requestBody": {
      "required": true,
      "contentTypes": [
        "application/json"
      ],
      "schemaRefs": [
        "#/components/schemas/EmailRegistrationInput"
      ]
    }
  },
  {
    "method": "get",
    "path": "/api/v1/auth/registration/settings",
    "tags": [
      "Auth"
    ],
    "deprecated": false,
    "security": [],
    "parameters": [],
    "responses": [
      {
        "status": "200",
        "contentTypes": [
          "application/json"
        ],
        "schemaRefs": [
          "#/components/schemas/AuthRegistrationSettings"
        ],
        "description": "Registration settings with the write-only SMTP password omitted."
      }
    ],
    "summary": "Get registration and SMTP settings"
  },
  {
    "method": "put",
    "path": "/api/v1/auth/registration/settings",
    "tags": [
      "Auth"
    ],
    "deprecated": false,
    "security": [],
    "parameters": [],
    "responses": [
      {
        "status": "200",
        "contentTypes": [],
        "schemaRefs": [],
        "description": "Registration settings updated."
      }
    ],
    "summary": "Update registration and SMTP settings",
    "requestBody": {
      "required": true,
      "contentTypes": [
        "application/json"
      ],
      "schemaRefs": [
        "#/components/schemas/AuthRegistrationSettingsInput"
      ]
    }
  },
  {
    "method": "get",
    "path": "/api/v1/auth/mfa/status",
    "tags": [
      "Auth"
    ],
    "deprecated": false,
    "security": [
      {
        "SessionCookie": []
      }
    ],
    "parameters": [],
    "responses": [
      {
        "status": "200",
        "contentTypes": [
          "application/json"
        ],
        "schemaRefs": [
          "#/components/schemas/MFAStatus"
        ],
        "description": "Current enrollment, policy, and recovery-code status."
      },
      {
        "status": "401",
        "contentTypes": [
          "application/json"
        ],
        "schemaRefs": [
          "#/components/schemas/ErrorResponse"
        ],
        "description": "Browser session is missing or invalid (`mfa.session_required` or an authentication error)."
      },
      {
        "status": "403",
        "contentTypes": [
          "application/json"
        ],
        "schemaRefs": [
          "#/components/schemas/ErrorResponse"
        ],
        "description": "Personal access tokens cannot access MFA session endpoints (`mfa.session_required`)."
      },
      {
        "status": "500",
        "contentTypes": [
          "application/json"
        ],
        "schemaRefs": [
          "#/components/schemas/ErrorResponse"
        ],
        "description": "MFA status could not be loaded."
      }
    ],
    "summary": "Get current user's MFA status",
    "description": "Requires an interactive browser session. Personal access tokens cannot manage or verify MFA."
  },
  {
    "method": "post",
    "path": "/api/v1/auth/mfa/totp/enroll",
    "tags": [
      "Auth"
    ],
    "deprecated": false,
    "security": [
      {
        "SessionCookie": []
      }
    ],
    "parameters": [],
    "responses": [
      {
        "status": "201",
        "contentTypes": [
          "application/json"
        ],
        "schemaRefs": [
          "#/components/schemas/MFAEnrollment"
        ],
        "description": "Pending TOTP enrollment created."
      },
      {
        "status": "401",
        "contentTypes": [
          "application/json"
        ],
        "schemaRefs": [
          "#/components/schemas/ErrorResponse"
        ],
        "description": "Browser session is missing or invalid, or primary reauthentication is required (`mfa.reauth_required`)."
      },
      {
        "status": "403",
        "contentTypes": [
          "application/json"
        ],
        "schemaRefs": [
          "#/components/schemas/ErrorResponse"
        ],
        "description": "Personal access tokens cannot enroll MFA (`mfa.session_required`)."
      },
      {
        "status": "409",
        "contentTypes": [
          "application/json"
        ],
        "schemaRefs": [
          "#/components/schemas/ErrorResponse"
        ],
        "description": "MFA is already enabled (`mfa.already_enabled`)."
      },
      {
        "status": "429",
        "contentTypes": [
          "application/json"
        ],
        "schemaRefs": [
          "#/components/schemas/ErrorResponse"
        ],
        "description": "Enrollment attempts exceeded the user or IP rate limit (`mfa.rate_limited`)."
      },
      {
        "status": "500",
        "contentTypes": [
          "application/json"
        ],
        "schemaRefs": [
          "#/components/schemas/ErrorResponse"
        ],
        "description": "The TOTP secret could not be stored (`mfa.secret_store_failed`) or enrollment persistence failed."
      },
      {
        "status": "503",
        "contentTypes": [
          "application/json"
        ],
        "schemaRefs": [
          "#/components/schemas/ErrorResponse"
        ],
        "description": "MFA rate limiting is unavailable in production (`mfa.rate_limit_unavailable`)."
      }
    ],
    "summary": "Start TOTP enrollment",
    "description": "Replaces any pending enrollment, stores the TOTP secret in the encrypted secret store, and returns the secret only for the current enrollment flow. Local accounts must re-enter their current password. OIDC accounts require non-impersonated primary authentication within the last five minutes; remember-token recovery does not refresh that timestamp.",
    "requestBody": {
      "required": true,
      "contentTypes": [
        "application/json"
      ],
      "schemaRefs": [
        "#/components/schemas/MFAEnrollmentInput"
      ]
    }
  },
  {
    "method": "post",
    "path": "/api/v1/auth/mfa/totp/confirm",
    "tags": [
      "Auth"
    ],
    "deprecated": false,
    "security": [
      {
        "SessionCookie": []
      }
    ],
    "parameters": [],
    "responses": [
      {
        "status": "200",
        "contentTypes": [
          "application/json"
        ],
        "schemaRefs": [
          "#/components/schemas/MFAConfirmResult"
        ],
        "description": "MFA enabled and recovery codes generated."
      },
      {
        "status": "400",
        "contentTypes": [
          "application/json"
        ],
        "schemaRefs": [
          "#/components/schemas/ErrorResponse"
        ],
        "description": "Invalid request body (`request.invalid_json`)."
      },
      {
        "status": "401",
        "contentTypes": [
          "application/json"
        ],
        "schemaRefs": [
          "#/components/schemas/ErrorResponse"
        ],
        "description": "Browser session is invalid or the TOTP code is invalid (`mfa.invalid_code`)."
      },
      {
        "status": "403",
        "contentTypes": [
          "application/json"
        ],
        "schemaRefs": [
          "#/components/schemas/ErrorResponse"
        ],
        "description": "Personal access tokens cannot confirm MFA (`mfa.session_required`)."
      },
      {
        "status": "409",
        "contentTypes": [
          "application/json"
        ],
        "schemaRefs": [
          "#/components/schemas/ErrorResponse"
        ],
        "description": "Enrollment is missing, changed, or already enabled (`mfa.enrollment_required`, `mfa.enrollment_changed`, or `mfa.already_enabled`)."
      },
      {
        "status": "429",
        "contentTypes": [
          "application/json"
        ],
        "schemaRefs": [
          "#/components/schemas/ErrorResponse"
        ],
        "description": "Confirmation attempts exceeded the user or IP rate limit (`mfa.rate_limited`)."
      },
      {
        "status": "500",
        "contentTypes": [
          "application/json"
        ],
        "schemaRefs": [
          "#/components/schemas/ErrorResponse"
        ],
        "description": "Recovery codes or enrollment state could not be persisted."
      },
      {
        "status": "503",
        "contentTypes": [
          "application/json"
        ],
        "schemaRefs": [
          "#/components/schemas/ErrorResponse"
        ],
        "description": "MFA rate limiting is unavailable in production (`mfa.rate_limit_unavailable`)."
      }
    ],
    "summary": "Confirm pending TOTP enrollment",
    "description": "Accepts the current or adjacent 30-second TOTP window. On success, enables MFA and returns ten one-time recovery codes that are shown only once.",
    "requestBody": {
      "required": true,
      "contentTypes": [
        "application/json"
      ],
      "schemaRefs": [
        "#/components/schemas/MFAConfirmInput"
      ]
    }
  },
  {
    "method": "post",
    "path": "/api/v1/auth/mfa/verify",
    "tags": [
      "Auth"
    ],
    "deprecated": false,
    "security": [
      {
        "SessionCookie": []
      }
    ],
    "parameters": [],
    "responses": [
      {
        "status": "200",
        "contentTypes": [
          "application/json"
        ],
        "schemaRefs": [
          "#/components/schemas/MFAVerifyResult"
        ],
        "description": "Step-up assertion created."
      },
      {
        "status": "400",
        "contentTypes": [
          "application/json"
        ],
        "schemaRefs": [
          "#/components/schemas/ErrorResponse"
        ],
        "description": "Unsupported purpose or both/neither credentials were supplied (`mfa.invalid_purpose` or `mfa.credential_required`)."
      },
      {
        "status": "401",
        "contentTypes": [
          "application/json"
        ],
        "schemaRefs": [
          "#/components/schemas/ErrorResponse"
        ],
        "description": "Browser session or MFA credential is invalid (`mfa.invalid_code`)."
      },
      {
        "status": "403",
        "contentTypes": [
          "application/json"
        ],
        "schemaRefs": [
          "#/components/schemas/ErrorResponse"
        ],
        "description": "Personal access tokens cannot create MFA assertions (`mfa.session_required`)."
      },
      {
        "status": "409",
        "contentTypes": [
          "application/json"
        ],
        "schemaRefs": [
          "#/components/schemas/ErrorResponse"
        ],
        "description": "MFA is not enabled for the current user (`mfa.not_enabled`)."
      },
      {
        "status": "429",
        "contentTypes": [
          "application/json"
        ],
        "schemaRefs": [
          "#/components/schemas/ErrorResponse"
        ],
        "description": "Verification attempts exceeded the user or IP rate limit (`mfa.rate_limited`)."
      },
      {
        "status": "500",
        "contentTypes": [
          "application/json"
        ],
        "schemaRefs": [
          "#/components/schemas/ErrorResponse"
        ],
        "description": "The Step-up assertion could not be persisted."
      },
      {
        "status": "503",
        "contentTypes": [
          "application/json"
        ],
        "schemaRefs": [
          "#/components/schemas/ErrorResponse"
        ],
        "description": "MFA rate limiting is unavailable in production (`mfa.rate_limit_unavailable`)."
      }
    ],
    "summary": "Verify MFA for a sensitive-operation purpose",
    "description": "Accepts exactly one TOTP code or one recovery code. A successful recovery code is consumed atomically. The resulting assertion is bound to the current user, browser session, and purpose.",
    "requestBody": {
      "required": true,
      "contentTypes": [
        "application/json"
      ],
      "schemaRefs": [
        "#/components/schemas/MFAVerifyInput"
      ]
    }
  },
  {
    "method": "post",
    "path": "/api/v1/auth/mfa/recovery-codes",
    "tags": [
      "Auth"
    ],
    "deprecated": false,
    "security": [
      {
        "SessionCookie": []
      }
    ],
    "parameters": [],
    "responses": [
      {
        "status": "200",
        "contentTypes": [
          "application/json"
        ],
        "schemaRefs": [
          "#/components/schemas/MFARecoveryCodes"
        ],
        "description": "Recovery codes replaced."
      },
      {
        "status": "401",
        "contentTypes": [
          "application/json"
        ],
        "schemaRefs": [
          "#/components/schemas/ErrorResponse"
        ],
        "description": "Browser session is missing or invalid."
      },
      {
        "status": "403",
        "contentTypes": [
          "application/json"
        ],
        "schemaRefs": [
          "#/components/schemas/ErrorResponse"
        ],
        "description": "MFA management assertion is missing or expired (`mfa_required`), or a personal access token was used (`mfa.session_required`)."
      },
      {
        "status": "409",
        "contentTypes": [
          "application/json"
        ],
        "schemaRefs": [
          "#/components/schemas/ErrorResponse"
        ],
        "description": "MFA is not enabled (`mfa.not_enabled`)."
      },
      {
        "status": "500",
        "contentTypes": [
          "application/json"
        ],
        "schemaRefs": [
          "#/components/schemas/ErrorResponse"
        ],
        "description": "Recovery codes could not be generated or persisted."
      }
    ],
    "summary": "Regenerate MFA recovery codes",
    "description": "Requires a valid `mfa_manage` assertion. Replaces and invalidates all previous recovery codes; the new plaintext codes are returned only once."
  },
  {
    "method": "delete",
    "path": "/api/v1/auth/mfa",
    "tags": [
      "Auth"
    ],
    "deprecated": false,
    "security": [
      {
        "SessionCookie": []
      }
    ],
    "parameters": [],
    "responses": [
      {
        "status": "204",
        "contentTypes": [],
        "schemaRefs": [],
        "description": "MFA disabled and assertions revoked."
      },
      {
        "status": "401",
        "contentTypes": [
          "application/json"
        ],
        "schemaRefs": [
          "#/components/schemas/ErrorResponse"
        ],
        "description": "Browser session is missing or invalid."
      },
      {
        "status": "403",
        "contentTypes": [
          "application/json"
        ],
        "schemaRefs": [
          "#/components/schemas/ErrorResponse"
        ],
        "description": "MFA management assertion is missing or expired (`mfa_required`), or a personal access token was used (`mfa.session_required`)."
      },
      {
        "status": "409",
        "contentTypes": [
          "application/json"
        ],
        "schemaRefs": [
          "#/components/schemas/ErrorResponse"
        ],
        "description": "The global policy requires another MFA-enabled platform administrator (`mfa.last_admin_required`)."
      },
      {
        "status": "500",
        "contentTypes": [
          "application/json"
        ],
        "schemaRefs": [
          "#/components/schemas/ErrorResponse"
        ],
        "description": "MFA state or encrypted secret data could not be deleted."
      }
    ],
    "summary": "Disable current user's MFA",
    "description": "Requires a valid `mfa_manage` assertion. Deletes the TOTP secret, recovery codes, and all current step-up assertions. While the global policy is enabled, the last MFA-enabled platform administrator cannot disable MFA."
  },
  {
    "method": "get",
    "path": "/api/v1/auth/providers",
    "tags": [
      "Auth"
    ],
    "deprecated": false,
    "security": [],
    "parameters": [],
    "responses": [
      {
        "status": "200",
        "contentTypes": [],
        "schemaRefs": [],
        "description": "Auth provider list."
      }
    ],
    "summary": "List auth providers"
  },
  {
    "method": "post",
    "path": "/api/v1/auth/providers",
    "tags": [
      "Auth"
    ],
    "deprecated": false,
    "security": [],
    "parameters": [],
    "responses": [
      {
        "status": "201",
        "contentTypes": [],
        "schemaRefs": [],
        "description": "Created auth provider."
      }
    ],
    "summary": "Create auth provider",
    "requestBody": {
      "required": true,
      "contentTypes": [
        "application/json"
      ],
      "schemaRefs": [
        "#/components/schemas/AuthProviderInput"
      ]
    }
  },
  {
    "method": "put",
    "path": "/api/v1/auth/providers/{providerId}",
    "tags": [
      "Auth"
    ],
    "deprecated": false,
    "security": [],
    "parameters": [
      {
        "name": "providerId",
        "in": "path",
        "required": true,
        "ref": "#/components/parameters/ProviderId",
        "schema": {
          "type": "string"
        }
      }
    ],
    "responses": [
      {
        "status": "200",
        "contentTypes": [],
        "schemaRefs": [],
        "description": "Updated auth provider."
      }
    ],
    "summary": "Update auth provider",
    "requestBody": {
      "required": true,
      "contentTypes": [
        "application/json"
      ],
      "schemaRefs": [
        "#/components/schemas/AuthProviderInput"
      ]
    }
  },
  {
    "method": "get",
    "path": "/api/v1/auth/admission-policy",
    "tags": [
      "Auth"
    ],
    "deprecated": false,
    "security": [],
    "parameters": [],
    "responses": [
      {
        "status": "200",
        "contentTypes": [],
        "schemaRefs": [],
        "description": "Auth admission policy."
      }
    ],
    "summary": "Get auth admission policy"
  },
  {
    "method": "put",
    "path": "/api/v1/auth/admission-policy",
    "tags": [
      "Auth"
    ],
    "deprecated": false,
    "security": [],
    "parameters": [],
    "responses": [
      {
        "status": "200",
        "contentTypes": [],
        "schemaRefs": [],
        "description": "Updated auth admission policy."
      }
    ],
    "summary": "Update auth admission policy",
    "requestBody": {
      "required": true,
      "contentTypes": [
        "application/json"
      ],
      "schemaRefs": [
        "#/components/schemas/AuthAdmissionPolicyInput"
      ]
    }
  },
  {
    "method": "get",
    "path": "/api/v1/auth/oidc/{providerId}/start",
    "tags": [
      "Auth"
    ],
    "deprecated": false,
    "security": [],
    "parameters": [
      {
        "name": "providerId",
        "in": "path",
        "required": true,
        "ref": "#/components/parameters/ProviderId",
        "schema": {
          "type": "string"
        }
      },
      {
        "name": "mode",
        "in": "query",
        "schema": {
          "type": "string",
          "enum": [
            "login",
            "bind"
          ]
        }
      },
      {
        "name": "redirect",
        "in": "query",
        "schema": {
          "type": "string"
        }
      }
    ],
    "responses": [
      {
        "status": "302",
        "contentTypes": [],
        "schemaRefs": [],
        "description": "Redirect to OIDC provider."
      }
    ],
    "summary": "Start OIDC login or binding flow"
  },
  {
    "method": "get",
    "path": "/api/v1/auth/oidc/callback",
    "tags": [
      "Auth"
    ],
    "deprecated": false,
    "security": [],
    "parameters": [
      {
        "name": "state",
        "in": "query",
        "required": true,
        "ref": "#/components/parameters/OAuthState",
        "schema": {
          "type": "string"
        }
      },
      {
        "name": "code",
        "in": "query",
        "required": true,
        "ref": "#/components/parameters/OAuthCode",
        "schema": {
          "type": "string"
        }
      }
    ],
    "responses": [
      {
        "status": "302",
        "contentTypes": [],
        "schemaRefs": [],
        "description": "Redirect after OIDC callback."
      }
    ],
    "summary": "Complete OIDC callback"
  },
  {
    "method": "get",
    "path": "/api/v1/users/me",
    "tags": [
      "Users"
    ],
    "deprecated": false,
    "security": [],
    "parameters": [],
    "responses": [
      {
        "status": "200",
        "contentTypes": [
          "application/json"
        ],
        "schemaRefs": [
          "#/components/schemas/CurrentUser"
        ],
        "description": "Current user."
      }
    ],
    "summary": "Get current user"
  },
  {
    "method": "put",
    "path": "/api/v1/users/me",
    "tags": [
      "Users"
    ],
    "deprecated": false,
    "security": [],
    "parameters": [],
    "responses": [
      {
        "status": "200",
        "contentTypes": [
          "application/json"
        ],
        "schemaRefs": [
          "#/components/schemas/CurrentUser"
        ],
        "description": "Updated current user."
      }
    ],
    "summary": "Update current user preferences",
    "requestBody": {
      "required": true,
      "contentTypes": [
        "application/json"
      ],
      "schemaRefs": [
        "#/components/schemas/UpdateCurrentUserInput"
      ]
    }
  },
  {
    "method": "put",
    "path": "/api/v1/users/me/password",
    "tags": [
      "Users"
    ],
    "deprecated": false,
    "security": [],
    "parameters": [],
    "responses": [
      {
        "status": "204",
        "contentTypes": [],
        "schemaRefs": [],
        "description": "Password updated and all sessions revoked."
      }
    ],
    "summary": "Set or change the current user's local password",
    "requestBody": {
      "required": true,
      "contentTypes": [
        "application/json"
      ],
      "schemaRefs": [
        "#/components/schemas/UpdateMyPasswordInput"
      ]
    }
  },
  {
    "method": "get",
    "path": "/api/v1/users/me/external-identities",
    "tags": [
      "Users"
    ],
    "deprecated": false,
    "security": [],
    "parameters": [],
    "responses": [
      {
        "status": "200",
        "contentTypes": [],
        "schemaRefs": [],
        "description": "External identity list."
      }
    ],
    "summary": "List current user's external identities"
  },
  {
    "method": "delete",
    "path": "/api/v1/users/me/external-identities/{identityId}",
    "tags": [
      "Users"
    ],
    "deprecated": false,
    "security": [],
    "parameters": [
      {
        "name": "identityId",
        "in": "path",
        "required": true,
        "ref": "#/components/parameters/IdentityId",
        "schema": {
          "type": "string"
        }
      }
    ],
    "responses": [
      {
        "status": "204",
        "contentTypes": [],
        "schemaRefs": [],
        "description": "External identity unbound."
      }
    ],
    "summary": "Unbind current user's external identity"
  },
  {
    "method": "get",
    "path": "/api/v1/users",
    "tags": [
      "Users"
    ],
    "deprecated": false,
    "security": [],
    "parameters": [
      {
        "name": "page",
        "in": "query",
        "ref": "#/components/parameters/Page",
        "schema": {
          "type": "integer"
        }
      },
      {
        "name": "pageSize",
        "in": "query",
        "ref": "#/components/parameters/PageSize",
        "schema": {
          "type": "integer"
        }
      },
      {
        "name": "sortBy",
        "in": "query",
        "schema": {
          "type": "string",
          "enum": [
            "createdAt",
            "email",
            "name",
            "role",
            "passwordSet",
            "status"
          ]
        }
      },
      {
        "name": "sortOrder",
        "in": "query",
        "ref": "#/components/parameters/SortOrder",
        "schema": {
          "type": "string",
          "enum": [
            "asc",
            "desc"
          ]
        }
      }
    ],
    "responses": [
      {
        "status": "200",
        "contentTypes": [],
        "schemaRefs": [],
        "description": "Paginated user list."
      }
    ],
    "summary": "List users"
  },
  {
    "method": "post",
    "path": "/api/v1/users",
    "tags": [
      "Users"
    ],
    "deprecated": false,
    "security": [],
    "parameters": [],
    "responses": [
      {
        "status": "201",
        "contentTypes": [],
        "schemaRefs": [],
        "description": "Created user."
      }
    ],
    "summary": "Create local user",
    "requestBody": {
      "required": true,
      "contentTypes": [
        "application/json"
      ],
      "schemaRefs": [
        "#/components/schemas/UserInput"
      ]
    }
  },
  {
    "method": "put",
    "path": "/api/v1/users/{userId}",
    "tags": [
      "Users"
    ],
    "deprecated": false,
    "security": [],
    "parameters": [
      {
        "name": "userId",
        "in": "path",
        "required": true,
        "ref": "#/components/parameters/UserId",
        "schema": {
          "type": "string"
        }
      }
    ],
    "responses": [
      {
        "status": "200",
        "contentTypes": [],
        "schemaRefs": [],
        "description": "Updated user."
      }
    ],
    "summary": "Update user",
    "requestBody": {
      "required": true,
      "contentTypes": [
        "application/json"
      ],
      "schemaRefs": [
        "#/components/schemas/UserInput"
      ]
    }
  },
  {
    "method": "delete",
    "path": "/api/v1/users/{userId}/mfa",
    "tags": [
      "Users"
    ],
    "deprecated": false,
    "security": [
      {
        "SessionCookie": []
      }
    ],
    "parameters": [
      {
        "name": "userId",
        "in": "path",
        "required": true,
        "ref": "#/components/parameters/UserId",
        "schema": {
          "type": "string"
        }
      }
    ],
    "responses": [
      {
        "status": "204",
        "contentTypes": [],
        "schemaRefs": [],
        "description": "Target MFA state reset."
      },
      {
        "status": "401",
        "contentTypes": [
          "application/json"
        ],
        "schemaRefs": [
          "#/components/schemas/ErrorResponse"
        ],
        "description": "Interactive browser session is missing or invalid."
      },
      {
        "status": "403",
        "contentTypes": [
          "application/json"
        ],
        "schemaRefs": [
          "#/components/schemas/ErrorResponse"
        ],
        "description": "Platform-administrator role or `user_admin_update` Step-up verification is required."
      },
      {
        "status": "404",
        "contentTypes": [
          "application/json"
        ],
        "schemaRefs": [
          "#/components/schemas/ErrorResponse"
        ],
        "description": "Target user or MFA enrollment was not found (`mfa.reset_target_not_found`)."
      },
      {
        "status": "409",
        "contentTypes": [
          "application/json"
        ],
        "schemaRefs": [
          "#/components/schemas/ErrorResponse"
        ],
        "description": "Self-reset is forbidden (`mfa.admin_reset_self_forbidden`) or the target is the last MFA-enabled platform administrator (`mfa.last_admin_required`)."
      }
    ],
    "summary": "Reset another user's MFA enrollment",
    "description": "Requires an interactive platform-administrator session and an active `user_admin_update` Step-up assertion. Deletes the target user's authenticator secret, recovery codes, and active Step-up assertions. Administrators cannot reset their own MFA through this endpoint and cannot remove the last enabled administrator MFA while the global policy is active."
  },
  {
    "method": "get",
    "path": "/api/v1/configs/definitions",
    "tags": [
      "Configs"
    ],
    "deprecated": false,
    "security": [],
    "parameters": [],
    "responses": [
      {
        "status": "200",
        "contentTypes": [
          "application/json"
        ],
        "schemaRefs": [
          "#/components/schemas/ConfigDefinition"
        ],
        "description": "Config definitions."
      }
    ],
    "summary": "List configurable app config definitions"
  },
  {
    "method": "put",
    "path": "/api/v1/configs",
    "tags": [
      "Configs"
    ],
    "deprecated": false,
    "security": [],
    "parameters": [],
    "responses": [
      {
        "status": "200",
        "contentTypes": [],
        "schemaRefs": [],
        "description": "Updated config dictionary."
      }
    ],
    "summary": "Update app configs",
    "requestBody": {
      "required": true,
      "contentTypes": [
        "application/json"
      ],
      "schemaRefs": [
        "#/components/schemas/UpdateConfigsInput"
      ]
    }
  },
  {
    "method": "get",
    "path": "/api/v1/data-retention/catalog",
    "tags": [
      "DataRetention"
    ],
    "deprecated": false,
    "security": [
      {
        "SessionCookie": []
      }
    ],
    "parameters": [],
    "responses": [
      {
        "status": "200",
        "contentTypes": [
          "application/json"
        ],
        "schemaRefs": [
          "#/components/schemas/DataRetentionCatalogResponse"
        ],
        "description": "Retention dataset catalog."
      },
      {
        "status": "403",
        "contentTypes": [
          "application/json"
        ],
        "schemaRefs": [
          "#/components/schemas/ErrorResponse"
        ],
        "description": "Platform-administrator role is required."
      }
    ],
    "summary": "List supported data-retention datasets",
    "description": "Returns the fixed cleanup catalog. Audit logs, billing data, and build, release, or Hook metadata are intentionally excluded."
  },
  {
    "method": "post",
    "path": "/api/v1/data-retention/preview",
    "tags": [
      "DataRetention"
    ],
    "deprecated": false,
    "security": [
      {
        "SessionCookie": []
      }
    ],
    "parameters": [],
    "responses": [
      {
        "status": "200",
        "contentTypes": [
          "application/json"
        ],
        "schemaRefs": [
          "#/components/schemas/DataRetentionResultResponse"
        ],
        "description": "Matching counts by dataset. `deleted` is zero for a preview."
      },
      {
        "status": "400",
        "contentTypes": [
          "application/json"
        ],
        "schemaRefs": [
          "#/components/schemas/ErrorResponse"
        ],
        "description": "Invalid range or unknown dataset (`retention.invalid_range` or `retention.invalid_dataset`)."
      },
      {
        "status": "403",
        "contentTypes": [
          "application/json"
        ],
        "schemaRefs": [
          "#/components/schemas/ErrorResponse"
        ],
        "description": "Platform-administrator role is required."
      }
    ],
    "summary": "Preview data matching a retention range",
    "description": "Counts rows without changing data. The selected range is left-closed and right-open (`startAt <= timestamp < endAt`). Active runtime records and protected datasets are never matched.",
    "requestBody": {
      "required": true,
      "contentTypes": [
        "application/json"
      ],
      "schemaRefs": [
        "#/components/schemas/DataRetentionRequest"
      ]
    }
  },
  {
    "method": "post",
    "path": "/api/v1/data-retention/cleanup",
    "tags": [
      "DataRetention"
    ],
    "deprecated": false,
    "security": [
      {
        "SessionCookie": []
      }
    ],
    "parameters": [],
    "responses": [
      {
        "status": "200",
        "contentTypes": [
          "application/json"
        ],
        "schemaRefs": [
          "#/components/schemas/DataRetentionResultResponse"
        ],
        "description": "Matched and deleted counts by dataset."
      },
      {
        "status": "400",
        "contentTypes": [
          "application/json"
        ],
        "schemaRefs": [
          "#/components/schemas/ErrorResponse"
        ],
        "description": "Invalid range or unknown dataset (`retention.invalid_range` or `retention.invalid_dataset`)."
      },
      {
        "status": "403",
        "contentTypes": [
          "application/json"
        ],
        "schemaRefs": [
          "#/components/schemas/ErrorResponse"
        ],
        "description": "Platform-administrator role or Step-up MFA assertion is required."
      },
      {
        "status": "500",
        "contentTypes": [
          "application/json"
        ],
        "schemaRefs": [
          "#/components/schemas/ErrorResponse"
        ],
        "description": "Cleanup failed (`retention.cleanup_failed`)."
      }
    ],
    "summary": "Permanently remove data matching a retention range",
    "description": "Runs the same fixed whitelist and protection rules as preview, then writes only the aggregate result to the audit log. The operation does not accept table names or SQL expressions. When Step-up MFA is enabled, a valid `data_retention_cleanup` assertion is also required.",
    "requestBody": {
      "required": true,
      "contentTypes": [
        "application/json"
      ],
      "schemaRefs": [
        "#/components/schemas/DataRetentionRequest"
      ]
    }
  },
  {
    "method": "post",
    "path": "/api/v1/runtime/clusters/{clusterId}/pods/terminal/authorize",
    "tags": [
      "Runtime"
    ],
    "deprecated": false,
    "security": [
      {
        "SessionCookie": []
      }
    ],
    "parameters": [
      {
        "name": "clusterId",
        "in": "path",
        "required": true,
        "ref": "#/components/parameters/ClusterId",
        "schema": {
          "type": "string"
        }
      },
      {
        "name": "namespace",
        "in": "query",
        "required": true,
        "schema": {
          "type": "string"
        }
      },
      {
        "name": "name",
        "in": "query",
        "required": true,
        "schema": {
          "type": "string"
        }
      }
    ],
    "responses": [
      {
        "status": "204",
        "contentTypes": [],
        "schemaRefs": [],
        "description": "Terminal preflight authorized. The WebSocket endpoint must still perform its own authorization checks."
      },
      {
        "status": "400",
        "contentTypes": [
          "application/json"
        ],
        "schemaRefs": [
          "#/components/schemas/ErrorResponse"
        ],
        "description": "Pod namespace or name is empty."
      },
      {
        "status": "401",
        "contentTypes": [
          "application/json"
        ],
        "schemaRefs": [
          "#/components/schemas/ErrorResponse"
        ],
        "description": "Interactive browser session is missing or invalid."
      },
      {
        "status": "403",
        "contentTypes": [
          "application/json"
        ],
        "schemaRefs": [
          "#/components/schemas/ErrorResponse"
        ],
        "description": "The current user is not a platform administrator, a personal access token was used (`mfa.session_required`), or Step-up verification is required (`mfa_required` with purpose `runtime_terminal`)."
      },
      {
        "status": "404",
        "contentTypes": [
          "application/json"
        ],
        "schemaRefs": [
          "#/components/schemas/ErrorResponse"
        ],
        "description": "Runtime cluster was not found."
      }
    ],
    "summary": "Authorize a runtime-cluster Pod terminal connection",
    "description": "Normal HTTP preflight used before opening the Pod terminal WebSocket. It verifies the interactive session, platform-administrator role, target cluster, and `runtime_terminal` Step-up assertion. A missing assertion returns `mfa_required`, allowing the frontend to show the MFA dialog and retry. A 204 authorizes only the preflight; the WebSocket repeats all checks before upgrading and revalidates session, role, assertion, Pod identity, and platform ownership every three seconds while connected. Revocation or expiry closes the shell."
  },
  {
    "method": "get",
    "path": "/api/v1/runtime/clusters",
    "tags": [
      "Runtime"
    ],
    "deprecated": false,
    "security": [],
    "parameters": [
      {
        "name": "projectId",
        "in": "query",
        "schema": {
          "type": "string"
        }
      },
      {
        "name": "page",
        "in": "query",
        "ref": "#/components/parameters/Page",
        "schema": {
          "type": "integer"
        }
      },
      {
        "name": "pageSize",
        "in": "query",
        "ref": "#/components/parameters/PageSize",
        "schema": {
          "type": "integer"
        }
      },
      {
        "name": "sortBy",
        "in": "query",
        "schema": {
          "type": "string",
          "enum": [
            "name",
            "type",
            "scope",
            "status",
            "createdAt"
          ]
        }
      },
      {
        "name": "sortOrder",
        "in": "query",
        "ref": "#/components/parameters/SortOrder",
        "schema": {
          "type": "string",
          "enum": [
            "asc",
            "desc"
          ]
        }
      }
    ],
    "responses": [
      {
        "status": "200",
        "contentTypes": [],
        "schemaRefs": [],
        "description": "Runtime cluster list or paginated runtime cluster list."
      }
    ],
    "summary": "List runtime clusters",
    "description": "Returns the legacy array response when pagination parameters are omitted, or a paginated response when `page`/`pageSize` is supplied."
  },
  {
    "method": "get",
    "path": "/api/v1/git/providers",
    "tags": [
      "Git"
    ],
    "deprecated": false,
    "security": [],
    "parameters": [
      {
        "name": "projectId",
        "in": "query",
        "schema": {
          "type": "string"
        }
      }
    ],
    "responses": [
      {
        "status": "200",
        "contentTypes": [],
        "schemaRefs": [],
        "description": "Git provider list."
      }
    ],
    "summary": "List Git providers"
  },
  {
    "method": "post",
    "path": "/api/v1/git/providers",
    "tags": [
      "Git"
    ],
    "deprecated": false,
    "security": [],
    "parameters": [],
    "responses": [
      {
        "status": "201",
        "contentTypes": [],
        "schemaRefs": [],
        "description": "Created Git provider."
      }
    ],
    "summary": "Create Git provider",
    "requestBody": {
      "required": true,
      "contentTypes": [
        "application/json"
      ],
      "schemaRefs": [
        "#/components/schemas/GitProviderInput"
      ]
    }
  },
  {
    "method": "put",
    "path": "/api/v1/git/providers/{providerId}",
    "tags": [
      "Git"
    ],
    "deprecated": false,
    "security": [],
    "parameters": [
      {
        "name": "providerId",
        "in": "path",
        "required": true,
        "ref": "#/components/parameters/ProviderId",
        "schema": {
          "type": "string"
        }
      }
    ],
    "responses": [
      {
        "status": "200",
        "contentTypes": [],
        "schemaRefs": [],
        "description": "Updated Git provider."
      }
    ],
    "summary": "Update Git provider",
    "requestBody": {
      "required": true,
      "contentTypes": [
        "application/json"
      ],
      "schemaRefs": [
        "#/components/schemas/GitProviderInput"
      ]
    }
  },
  {
    "method": "delete",
    "path": "/api/v1/git/providers/{providerId}",
    "tags": [
      "Git"
    ],
    "deprecated": false,
    "security": [],
    "parameters": [
      {
        "name": "providerId",
        "in": "path",
        "required": true,
        "ref": "#/components/parameters/ProviderId",
        "schema": {
          "type": "string"
        }
      }
    ],
    "responses": [
      {
        "status": "204",
        "contentTypes": [],
        "schemaRefs": [],
        "description": "Deleted Git provider."
      }
    ],
    "summary": "Delete Git provider"
  },
  {
    "method": "get",
    "path": "/api/v1/git/providers/{providerId}/oauth/start",
    "tags": [
      "Git"
    ],
    "deprecated": false,
    "security": [],
    "parameters": [
      {
        "name": "providerId",
        "in": "path",
        "required": true,
        "ref": "#/components/parameters/ProviderId",
        "schema": {
          "type": "string"
        }
      },
      {
        "name": "redirect",
        "in": "query",
        "schema": {
          "type": "string"
        }
      },
      {
        "name": "frontendOrigin",
        "in": "query",
        "schema": {
          "type": "string"
        }
      }
    ],
    "responses": [
      {
        "status": "302",
        "contentTypes": [],
        "schemaRefs": [],
        "description": "Redirect to Git OAuth provider."
      }
    ],
    "summary": "Start GitHub or Gitea OAuth flow"
  },
  {
    "method": "get",
    "path": "/api/v1/git/oauth/callback",
    "tags": [
      "Git"
    ],
    "deprecated": false,
    "security": [],
    "parameters": [
      {
        "name": "state",
        "in": "query",
        "required": true,
        "ref": "#/components/parameters/OAuthState",
        "schema": {
          "type": "string"
        }
      },
      {
        "name": "code",
        "in": "query",
        "required": true,
        "ref": "#/components/parameters/OAuthCode",
        "schema": {
          "type": "string"
        }
      }
    ],
    "responses": [
      {
        "status": "302",
        "contentTypes": [],
        "schemaRefs": [],
        "description": "Redirect after Git OAuth callback."
      }
    ],
    "summary": "Complete Git OAuth callback"
  },
  {
    "method": "post",
    "path": "/api/v1/git/webhooks/{bindingId}",
    "tags": [
      "Git"
    ],
    "deprecated": false,
    "security": [],
    "parameters": [
      {
        "name": "bindingId",
        "in": "path",
        "required": true,
        "ref": "#/components/parameters/BindingId",
        "schema": {
          "type": "string"
        }
      }
    ],
    "responses": [
      {
        "status": "200",
        "contentTypes": [],
        "schemaRefs": [],
        "description": "Webhook accepted."
      },
      {
        "status": "401",
        "contentTypes": [],
        "schemaRefs": [],
        "description": "Invalid webhook signature."
      }
    ],
    "summary": "Receive Git webhook event"
  },
  {
    "method": "get",
    "path": "/api/v1/git/accounts",
    "tags": [
      "Git"
    ],
    "deprecated": false,
    "security": [],
    "parameters": [
      {
        "name": "projectId",
        "in": "query",
        "schema": {
          "type": "string"
        }
      }
    ],
    "responses": [
      {
        "status": "200",
        "contentTypes": [],
        "schemaRefs": [],
        "description": "Git account list."
      }
    ],
    "summary": "List current user Git accounts"
  },
  {
    "method": "post",
    "path": "/api/v1/git/accounts",
    "tags": [
      "Git"
    ],
    "deprecated": false,
    "security": [],
    "parameters": [],
    "responses": [
      {
        "status": "201",
        "contentTypes": [],
        "schemaRefs": [],
        "description": "Created Git account."
      }
    ],
    "summary": "Create current user Git account manually",
    "requestBody": {
      "required": true,
      "contentTypes": [
        "application/json"
      ],
      "schemaRefs": [
        "#/components/schemas/GitAccountInput"
      ]
    }
  },
  {
    "method": "put",
    "path": "/api/v1/git/accounts/{accountId}",
    "tags": [
      "Git"
    ],
    "deprecated": false,
    "security": [],
    "parameters": [
      {
        "name": "accountId",
        "in": "path",
        "required": true,
        "ref": "#/components/parameters/AccountId",
        "schema": {
          "type": "string"
        }
      }
    ],
    "responses": [
      {
        "status": "200",
        "contentTypes": [],
        "schemaRefs": [],
        "description": "Updated Git account."
      }
    ],
    "summary": "Update current user Git account",
    "requestBody": {
      "required": true,
      "contentTypes": [
        "application/json"
      ],
      "schemaRefs": [
        "#/components/schemas/GitAccountInput"
      ]
    }
  },
  {
    "method": "delete",
    "path": "/api/v1/git/accounts/{accountId}",
    "tags": [
      "Git"
    ],
    "deprecated": false,
    "security": [],
    "parameters": [
      {
        "name": "accountId",
        "in": "path",
        "required": true,
        "ref": "#/components/parameters/AccountId",
        "schema": {
          "type": "string"
        }
      }
    ],
    "responses": [
      {
        "status": "204",
        "contentTypes": [],
        "schemaRefs": [],
        "description": "Deleted Git account."
      }
    ],
    "summary": "Delete current user Git account"
  },
  {
    "method": "post",
    "path": "/api/v1/git/accounts/{accountId}/refresh",
    "tags": [
      "Git"
    ],
    "deprecated": false,
    "security": [],
    "parameters": [
      {
        "name": "accountId",
        "in": "path",
        "required": true,
        "ref": "#/components/parameters/AccountId",
        "schema": {
          "type": "string"
        }
      }
    ],
    "responses": [
      {
        "status": "200",
        "contentTypes": [],
        "schemaRefs": [],
        "description": "Refreshed Git account."
      }
    ],
    "summary": "Refresh current user Git account token"
  },
  {
    "method": "get",
    "path": "/api/v1/git/accounts/{accountId}/repositories",
    "tags": [
      "Git"
    ],
    "deprecated": false,
    "security": [],
    "parameters": [
      {
        "name": "accountId",
        "in": "path",
        "required": true,
        "ref": "#/components/parameters/AccountId",
        "schema": {
          "type": "string"
        }
      },
      {
        "name": "page",
        "in": "query",
        "ref": "#/components/parameters/Page",
        "schema": {
          "type": "integer"
        }
      },
      {
        "name": "pageSize",
        "in": "query",
        "ref": "#/components/parameters/PageSize",
        "schema": {
          "type": "integer"
        }
      },
      {
        "name": "search",
        "in": "query",
        "schema": {
          "type": "string"
        }
      }
    ],
    "responses": [
      {
        "status": "200",
        "contentTypes": [],
        "schemaRefs": [],
        "description": "Repository list."
      }
    ],
    "summary": "List repositories visible to a Git account"
  },
  {
    "method": "get",
    "path": "/api/v1/git/accounts/{accountId}/repositories/{owner}/{repo}/branches",
    "tags": [
      "Git"
    ],
    "deprecated": false,
    "security": [],
    "parameters": [
      {
        "name": "accountId",
        "in": "path",
        "required": true,
        "ref": "#/components/parameters/AccountId",
        "schema": {
          "type": "string"
        }
      },
      {
        "name": "owner",
        "in": "path",
        "required": true,
        "ref": "#/components/parameters/Owner",
        "schema": {
          "type": "string"
        }
      },
      {
        "name": "repo",
        "in": "path",
        "required": true,
        "ref": "#/components/parameters/Repo",
        "schema": {
          "type": "string"
        }
      }
    ],
    "responses": [
      {
        "status": "200",
        "contentTypes": [],
        "schemaRefs": [],
        "description": "Branch list."
      }
    ],
    "summary": "List repository branches"
  },
  {
    "method": "get",
    "path": "/api/v1/git/accounts/{accountId}/repositories/{owner}/{repo}/file",
    "tags": [
      "Git"
    ],
    "deprecated": false,
    "security": [],
    "parameters": [
      {
        "name": "accountId",
        "in": "path",
        "required": true,
        "ref": "#/components/parameters/AccountId",
        "schema": {
          "type": "string"
        }
      },
      {
        "name": "owner",
        "in": "path",
        "required": true,
        "ref": "#/components/parameters/Owner",
        "schema": {
          "type": "string"
        }
      },
      {
        "name": "repo",
        "in": "path",
        "required": true,
        "ref": "#/components/parameters/Repo",
        "schema": {
          "type": "string"
        }
      },
      {
        "name": "path",
        "in": "query",
        "required": true,
        "schema": {
          "type": "string"
        }
      },
      {
        "name": "ref",
        "in": "query",
        "schema": {
          "type": "string"
        }
      }
    ],
    "responses": [
      {
        "status": "200",
        "contentTypes": [],
        "schemaRefs": [],
        "description": "File content."
      }
    ],
    "summary": "Read repository file content"
  },
  {
    "method": "get",
    "path": "/api/v1/registries",
    "tags": [
      "Registries"
    ],
    "deprecated": false,
    "security": [],
    "parameters": [
      {
        "name": "projectId",
        "in": "query",
        "schema": {
          "type": "string"
        }
      },
      {
        "name": "page",
        "in": "query",
        "ref": "#/components/parameters/Page",
        "schema": {
          "type": "integer"
        }
      },
      {
        "name": "pageSize",
        "in": "query",
        "ref": "#/components/parameters/PageSize",
        "schema": {
          "type": "integer"
        }
      },
      {
        "name": "sortBy",
        "in": "query",
        "schema": {
          "type": "string",
          "enum": [
            "name",
            "scope",
            "createdAt"
          ]
        }
      },
      {
        "name": "sortOrder",
        "in": "query",
        "ref": "#/components/parameters/SortOrder",
        "schema": {
          "type": "string",
          "enum": [
            "asc",
            "desc"
          ]
        }
      }
    ],
    "responses": [
      {
        "status": "200",
        "contentTypes": [],
        "schemaRefs": [],
        "description": "Artifact registry list."
      }
    ],
    "summary": "List artifact registries"
  },
  {
    "method": "post",
    "path": "/api/v1/registries",
    "tags": [
      "Registries"
    ],
    "deprecated": false,
    "security": [],
    "parameters": [],
    "responses": [
      {
        "status": "201",
        "contentTypes": [],
        "schemaRefs": [],
        "description": "Created artifact registry."
      }
    ],
    "summary": "Create artifact registry",
    "requestBody": {
      "required": true,
      "contentTypes": [
        "application/json"
      ],
      "schemaRefs": [
        "#/components/schemas/ArtifactRegistryInput"
      ]
    }
  },
  {
    "method": "put",
    "path": "/api/v1/registries/{registryId}",
    "tags": [
      "Registries"
    ],
    "deprecated": false,
    "security": [],
    "parameters": [
      {
        "name": "registryId",
        "in": "path",
        "required": true,
        "ref": "#/components/parameters/RegistryId",
        "schema": {
          "type": "string"
        }
      }
    ],
    "responses": [
      {
        "status": "200",
        "contentTypes": [],
        "schemaRefs": [],
        "description": "Updated artifact registry."
      }
    ],
    "summary": "Update artifact registry",
    "requestBody": {
      "required": true,
      "contentTypes": [
        "application/json"
      ],
      "schemaRefs": [
        "#/components/schemas/ArtifactRegistryInput"
      ]
    }
  },
  {
    "method": "delete",
    "path": "/api/v1/registries/{registryId}",
    "tags": [
      "Registries"
    ],
    "deprecated": false,
    "security": [],
    "parameters": [
      {
        "name": "registryId",
        "in": "path",
        "required": true,
        "ref": "#/components/parameters/RegistryId",
        "schema": {
          "type": "string"
        }
      }
    ],
    "responses": [
      {
        "status": "204",
        "contentTypes": [],
        "schemaRefs": [],
        "description": "Deleted artifact registry."
      }
    ],
    "summary": "Delete artifact registry"
  },
  {
    "method": "post",
    "path": "/api/v1/registries/{registryId}/test",
    "tags": [
      "Registries"
    ],
    "deprecated": false,
    "security": [],
    "parameters": [
      {
        "name": "registryId",
        "in": "path",
        "required": true,
        "ref": "#/components/parameters/RegistryId",
        "schema": {
          "type": "string"
        }
      }
    ],
    "responses": [
      {
        "status": "200",
        "contentTypes": [],
        "schemaRefs": [],
        "description": "Registry test result."
      }
    ],
    "summary": "Test artifact registry connectivity"
  },
  {
    "method": "get",
    "path": "/api/v1/registries/{registryId}/credentials",
    "tags": [
      "Registries"
    ],
    "deprecated": false,
    "security": [],
    "parameters": [
      {
        "name": "registryId",
        "in": "path",
        "required": true,
        "ref": "#/components/parameters/RegistryId",
        "schema": {
          "type": "string"
        }
      },
      {
        "name": "page",
        "in": "query",
        "ref": "#/components/parameters/Page",
        "schema": {
          "type": "integer"
        }
      },
      {
        "name": "pageSize",
        "in": "query",
        "ref": "#/components/parameters/PageSize",
        "schema": {
          "type": "integer"
        }
      },
      {
        "name": "sortBy",
        "in": "query",
        "schema": {
          "type": "string",
          "enum": [
            "name",
            "username",
            "createdAt"
          ]
        }
      },
      {
        "name": "sortOrder",
        "in": "query",
        "ref": "#/components/parameters/SortOrder",
        "schema": {
          "type": "string",
          "enum": [
            "asc",
            "desc"
          ]
        }
      }
    ],
    "responses": [
      {
        "status": "200",
        "contentTypes": [],
        "schemaRefs": [],
        "description": "Registry credential list."
      }
    ],
    "summary": "List registry credentials"
  },
  {
    "method": "post",
    "path": "/api/v1/registries/{registryId}/credentials",
    "tags": [
      "Registries"
    ],
    "deprecated": false,
    "security": [],
    "parameters": [
      {
        "name": "registryId",
        "in": "path",
        "required": true,
        "ref": "#/components/parameters/RegistryId",
        "schema": {
          "type": "string"
        }
      }
    ],
    "responses": [
      {
        "status": "201",
        "contentTypes": [],
        "schemaRefs": [],
        "description": "Created registry credential."
      }
    ],
    "summary": "Create registry credential",
    "requestBody": {
      "required": true,
      "contentTypes": [
        "application/json"
      ],
      "schemaRefs": [
        "#/components/schemas/RegistryCredentialInput"
      ]
    }
  },
  {
    "method": "get",
    "path": "/api/v1/registry-credentials",
    "tags": [
      "Registries"
    ],
    "deprecated": false,
    "security": [],
    "parameters": [
      {
        "name": "page",
        "in": "query",
        "ref": "#/components/parameters/Page",
        "schema": {
          "type": "integer"
        }
      },
      {
        "name": "pageSize",
        "in": "query",
        "ref": "#/components/parameters/PageSize",
        "schema": {
          "type": "integer"
        }
      },
      {
        "name": "sortBy",
        "in": "query",
        "schema": {
          "type": "string",
          "enum": [
            "name",
            "username",
            "createdAt"
          ]
        }
      },
      {
        "name": "sortOrder",
        "in": "query",
        "ref": "#/components/parameters/SortOrder",
        "schema": {
          "type": "string",
          "enum": [
            "asc",
            "desc"
          ]
        }
      }
    ],
    "responses": [
      {
        "status": "200",
        "contentTypes": [],
        "schemaRefs": [],
        "description": "Paginated registry credential list."
      }
    ],
    "summary": "List visible registry credentials across registries"
  },
  {
    "method": "put",
    "path": "/api/v1/registries/{registryId}/credentials/{credentialId}",
    "tags": [
      "Registries"
    ],
    "deprecated": false,
    "security": [],
    "parameters": [
      {
        "name": "registryId",
        "in": "path",
        "required": true,
        "ref": "#/components/parameters/RegistryId",
        "schema": {
          "type": "string"
        }
      },
      {
        "name": "credentialId",
        "in": "path",
        "required": true,
        "ref": "#/components/parameters/CredentialId",
        "schema": {
          "type": "string"
        }
      }
    ],
    "responses": [
      {
        "status": "200",
        "contentTypes": [
          "application/json"
        ],
        "schemaRefs": [
          "#/components/schemas/RegistryCredential"
        ],
        "description": "Updated registry credential."
      }
    ],
    "summary": "Update registry credential",
    "requestBody": {
      "required": true,
      "contentTypes": [
        "application/json"
      ],
      "schemaRefs": [
        "#/components/schemas/RegistryCredentialInput"
      ]
    }
  },
  {
    "method": "delete",
    "path": "/api/v1/registries/{registryId}/credentials/{credentialId}",
    "tags": [
      "Registries"
    ],
    "deprecated": false,
    "security": [],
    "parameters": [
      {
        "name": "registryId",
        "in": "path",
        "required": true,
        "ref": "#/components/parameters/RegistryId",
        "schema": {
          "type": "string"
        }
      },
      {
        "name": "credentialId",
        "in": "path",
        "required": true,
        "ref": "#/components/parameters/CredentialId",
        "schema": {
          "type": "string"
        }
      }
    ],
    "responses": [
      {
        "status": "204",
        "contentTypes": [],
        "schemaRefs": [],
        "description": "Deleted registry credential."
      }
    ],
    "summary": "Delete registry credential"
  },
  {
    "method": "get",
    "path": "/api/v1/container-images",
    "tags": [
      "Registries"
    ],
    "deprecated": false,
    "security": [],
    "parameters": [
      {
        "name": "projectId",
        "in": "query",
        "schema": {
          "type": "string"
        }
      },
      {
        "name": "applicationId",
        "in": "query",
        "schema": {
          "type": "string"
        }
      },
      {
        "name": "registryId",
        "in": "query",
        "schema": {
          "type": "string"
        }
      }
    ],
    "responses": [
      {
        "status": "200",
        "contentTypes": [],
        "schemaRefs": [],
        "description": "Container image list."
      }
    ],
    "summary": "List container image records"
  },
  {
    "method": "post",
    "path": "/api/v1/container-images",
    "tags": [
      "Registries"
    ],
    "deprecated": false,
    "security": [],
    "parameters": [],
    "responses": [
      {
        "status": "201",
        "contentTypes": [],
        "schemaRefs": [],
        "description": "Created container image record."
      }
    ],
    "summary": "Create container image record",
    "requestBody": {
      "required": true,
      "contentTypes": [
        "application/json"
      ],
      "schemaRefs": [
        "#/components/schemas/ContainerImageInput"
      ]
    }
  },
  {
    "method": "get",
    "path": "/api/v1/dashboard",
    "tags": [
      "Dashboard"
    ],
    "deprecated": false,
    "security": [],
    "parameters": [],
    "responses": [
      {
        "status": "200",
        "contentTypes": [
          "application/json"
        ],
        "schemaRefs": [
          "#/components/schemas/DashboardOverview"
        ],
        "description": "Dashboard overview scoped to the current user's visible project spaces and resources."
      },
      {
        "status": "401",
        "contentTypes": [
          "application/json"
        ],
        "schemaRefs": [
          "#/components/schemas/ErrorResponse"
        ],
        "description": "Authentication is required."
      },
      {
        "status": "500",
        "contentTypes": [
          "application/json"
        ],
        "schemaRefs": [
          "#/components/schemas/ErrorResponse"
        ],
        "description": "Dashboard aggregation failed (`dashboard.load_failed`)."
      }
    ],
    "summary": "Get the current user's dashboard overview",
    "description": "Returns the task-oriented dashboard aggregation in one response. Future dashboard read models are added to this contract instead of being composed from multiple browser requests."
  },
  {
    "method": "get",
    "path": "/api/v1/projects",
    "tags": [
      "Projects"
    ],
    "deprecated": false,
    "security": [],
    "parameters": [
      {
        "name": "page",
        "in": "query",
        "ref": "#/components/parameters/Page",
        "schema": {
          "type": "integer"
        }
      },
      {
        "name": "pageSize",
        "in": "query",
        "ref": "#/components/parameters/PageSize",
        "schema": {
          "type": "integer"
        }
      },
      {
        "name": "sortBy",
        "in": "query",
        "schema": {
          "type": "string",
          "enum": [
            "createdAt",
            "name",
            "identifier"
          ]
        }
      },
      {
        "name": "sortOrder",
        "in": "query",
        "ref": "#/components/parameters/SortOrder",
        "schema": {
          "type": "string",
          "enum": [
            "asc",
            "desc"
          ]
        }
      }
    ],
    "responses": [
      {
        "status": "200",
        "contentTypes": [
          "application/json"
        ],
        "schemaRefs": [
          "#/components/schemas/Project",
          "#/components/schemas/PaginatedProjectList"
        ],
        "description": "Project list or paginated project list."
      }
    ],
    "summary": "List projects",
    "description": "Returns the legacy project array when pagination parameters are omitted. Returns a paginated object when page or pageSize is provided."
  },
  {
    "method": "post",
    "path": "/api/v1/projects",
    "tags": [
      "Projects"
    ],
    "deprecated": false,
    "security": [],
    "parameters": [],
    "responses": [
      {
        "status": "201",
        "contentTypes": [],
        "schemaRefs": [],
        "description": "Created project."
      }
    ],
    "summary": "Create project",
    "requestBody": {
      "required": true,
      "contentTypes": [
        "application/json"
      ],
      "schemaRefs": [
        "#/components/schemas/ProjectInput"
      ]
    }
  },
  {
    "method": "get",
    "path": "/api/v1/projects/pins",
    "tags": [
      "Projects"
    ],
    "deprecated": false,
    "security": [],
    "parameters": [],
    "responses": [
      {
        "status": "200",
        "contentTypes": [],
        "schemaRefs": [],
        "description": "Pinned project list."
      }
    ],
    "summary": "List current user's pinned projects"
  },
  {
    "method": "get",
    "path": "/api/v1/projects/{projectId}",
    "tags": [
      "Projects"
    ],
    "deprecated": false,
    "security": [],
    "parameters": [
      {
        "name": "projectId",
        "in": "path",
        "required": true,
        "ref": "#/components/parameters/ProjectId",
        "schema": {
          "type": "string"
        }
      }
    ],
    "responses": [
      {
        "status": "200",
        "contentTypes": [],
        "schemaRefs": [],
        "description": "Project."
      }
    ],
    "summary": "Get project"
  },
  {
    "method": "put",
    "path": "/api/v1/projects/{projectId}",
    "tags": [
      "Projects"
    ],
    "deprecated": false,
    "security": [],
    "parameters": [
      {
        "name": "projectId",
        "in": "path",
        "required": true,
        "ref": "#/components/parameters/ProjectId",
        "schema": {
          "type": "string"
        }
      }
    ],
    "responses": [
      {
        "status": "200",
        "contentTypes": [],
        "schemaRefs": [],
        "description": "Updated project."
      }
    ],
    "summary": "Update project",
    "requestBody": {
      "required": true,
      "contentTypes": [
        "application/json"
      ],
      "schemaRefs": [
        "#/components/schemas/ProjectInput"
      ]
    }
  },
  {
    "method": "delete",
    "path": "/api/v1/projects/{projectId}",
    "tags": [
      "Projects"
    ],
    "deprecated": false,
    "security": [],
    "parameters": [
      {
        "name": "projectId",
        "in": "path",
        "required": true,
        "ref": "#/components/parameters/ProjectId",
        "schema": {
          "type": "string"
        }
      }
    ],
    "responses": [
      {
        "status": "204",
        "contentTypes": [],
        "schemaRefs": [],
        "description": "Deleted project."
      }
    ],
    "summary": "Delete project"
  },
  {
    "method": "put",
    "path": "/api/v1/projects/{projectId}/pin",
    "tags": [
      "Projects"
    ],
    "deprecated": false,
    "security": [],
    "parameters": [
      {
        "name": "projectId",
        "in": "path",
        "required": true,
        "ref": "#/components/parameters/ProjectId",
        "schema": {
          "type": "string"
        }
      }
    ],
    "responses": [
      {
        "status": "200",
        "contentTypes": [],
        "schemaRefs": [],
        "description": "Updated pinned project."
      },
      {
        "status": "201",
        "contentTypes": [],
        "schemaRefs": [],
        "description": "Created pinned project."
      }
    ],
    "summary": "Pin project for current user"
  },
  {
    "method": "delete",
    "path": "/api/v1/projects/{projectId}/pin",
    "tags": [
      "Projects"
    ],
    "deprecated": false,
    "security": [],
    "parameters": [
      {
        "name": "projectId",
        "in": "path",
        "required": true,
        "ref": "#/components/parameters/ProjectId",
        "schema": {
          "type": "string"
        }
      }
    ],
    "responses": [
      {
        "status": "204",
        "contentTypes": [],
        "schemaRefs": [],
        "description": "Unpinned project."
      }
    ],
    "summary": "Unpin project for current user"
  },
  {
    "method": "get",
    "path": "/api/v1/projects/{projectId}/registries/default",
    "tags": [
      "Registries"
    ],
    "deprecated": false,
    "security": [],
    "parameters": [
      {
        "name": "projectId",
        "in": "path",
        "required": true,
        "ref": "#/components/parameters/ProjectId",
        "schema": {
          "type": "string"
        }
      }
    ],
    "responses": [
      {
        "status": "200",
        "contentTypes": [],
        "schemaRefs": [],
        "description": "Default artifact registry."
      }
    ],
    "summary": "Get default artifact registry for a project"
  },
  {
    "method": "get",
    "path": "/api/v1/projects/{projectId}/members",
    "tags": [
      "Projects"
    ],
    "deprecated": false,
    "security": [],
    "parameters": [
      {
        "name": "projectId",
        "in": "path",
        "required": true,
        "ref": "#/components/parameters/ProjectId",
        "schema": {
          "type": "string"
        }
      }
    ],
    "responses": [
      {
        "status": "200",
        "contentTypes": [],
        "schemaRefs": [],
        "description": "Project member list."
      }
    ],
    "summary": "List project members"
  },
  {
    "method": "post",
    "path": "/api/v1/projects/{projectId}/members",
    "tags": [
      "Projects"
    ],
    "deprecated": false,
    "security": [],
    "parameters": [
      {
        "name": "projectId",
        "in": "path",
        "required": true,
        "ref": "#/components/parameters/ProjectId",
        "schema": {
          "type": "string"
        }
      }
    ],
    "responses": [
      {
        "status": "201",
        "contentTypes": [],
        "schemaRefs": [],
        "description": "Created project member."
      }
    ],
    "summary": "Create project member",
    "requestBody": {
      "required": true,
      "contentTypes": [
        "application/json"
      ],
      "schemaRefs": [
        "#/components/schemas/ProjectMemberInput"
      ]
    }
  },
  {
    "method": "put",
    "path": "/api/v1/projects/{projectId}/members/{memberId}",
    "tags": [
      "Projects"
    ],
    "deprecated": false,
    "security": [],
    "parameters": [
      {
        "name": "projectId",
        "in": "path",
        "required": true,
        "ref": "#/components/parameters/ProjectId",
        "schema": {
          "type": "string"
        }
      },
      {
        "name": "memberId",
        "in": "path",
        "required": true,
        "ref": "#/components/parameters/MemberId",
        "schema": {
          "type": "string"
        }
      }
    ],
    "responses": [
      {
        "status": "200",
        "contentTypes": [],
        "schemaRefs": [],
        "description": "Updated project member."
      }
    ],
    "summary": "Update project member",
    "requestBody": {
      "required": true,
      "contentTypes": [
        "application/json"
      ],
      "schemaRefs": [
        "#/components/schemas/ProjectMemberInput"
      ]
    }
  },
  {
    "method": "delete",
    "path": "/api/v1/projects/{projectId}/members/{memberId}",
    "tags": [
      "Projects"
    ],
    "deprecated": false,
    "security": [],
    "parameters": [
      {
        "name": "projectId",
        "in": "path",
        "required": true,
        "ref": "#/components/parameters/ProjectId",
        "schema": {
          "type": "string"
        }
      },
      {
        "name": "memberId",
        "in": "path",
        "required": true,
        "ref": "#/components/parameters/MemberId",
        "schema": {
          "type": "string"
        }
      }
    ],
    "responses": [
      {
        "status": "204",
        "contentTypes": [],
        "schemaRefs": [],
        "description": "Deleted project member."
      }
    ],
    "summary": "Delete project member"
  },
  {
    "method": "get",
    "path": "/api/v1/projects/{projectId}/applications",
    "tags": [
      "Applications"
    ],
    "deprecated": false,
    "security": [],
    "parameters": [
      {
        "name": "projectId",
        "in": "path",
        "required": true,
        "ref": "#/components/parameters/ProjectId",
        "schema": {
          "type": "string"
        }
      }
    ],
    "responses": [
      {
        "status": "200",
        "contentTypes": [],
        "schemaRefs": [],
        "description": "Application list."
      }
    ],
    "summary": "List applications"
  },
  {
    "method": "post",
    "path": "/api/v1/projects/{projectId}/applications",
    "tags": [
      "Applications"
    ],
    "deprecated": false,
    "security": [],
    "parameters": [
      {
        "name": "projectId",
        "in": "path",
        "required": true,
        "ref": "#/components/parameters/ProjectId",
        "schema": {
          "type": "string"
        }
      }
    ],
    "responses": [
      {
        "status": "201",
        "contentTypes": [],
        "schemaRefs": [],
        "description": "Created application."
      }
    ],
    "summary": "Create application",
    "requestBody": {
      "required": true,
      "contentTypes": [
        "application/json"
      ],
      "schemaRefs": [
        "#/components/schemas/ApplicationInput"
      ]
    }
  },
  {
    "method": "get",
    "path": "/api/v1/projects/{projectId}/applications/{applicationId}",
    "tags": [
      "Applications"
    ],
    "deprecated": false,
    "security": [],
    "parameters": [
      {
        "name": "projectId",
        "in": "path",
        "required": true,
        "ref": "#/components/parameters/ProjectId",
        "schema": {
          "type": "string"
        }
      },
      {
        "name": "applicationId",
        "in": "path",
        "required": true,
        "ref": "#/components/parameters/ApplicationId",
        "schema": {
          "type": "string"
        }
      }
    ],
    "responses": [
      {
        "status": "200",
        "contentTypes": [],
        "schemaRefs": [],
        "description": "Application."
      }
    ],
    "summary": "Get application"
  },
  {
    "method": "put",
    "path": "/api/v1/projects/{projectId}/applications/{applicationId}",
    "tags": [
      "Applications"
    ],
    "deprecated": false,
    "security": [],
    "parameters": [
      {
        "name": "projectId",
        "in": "path",
        "required": true,
        "ref": "#/components/parameters/ProjectId",
        "schema": {
          "type": "string"
        }
      },
      {
        "name": "applicationId",
        "in": "path",
        "required": true,
        "ref": "#/components/parameters/ApplicationId",
        "schema": {
          "type": "string"
        }
      }
    ],
    "responses": [
      {
        "status": "200",
        "contentTypes": [],
        "schemaRefs": [],
        "description": "Updated application."
      }
    ],
    "summary": "Update application",
    "requestBody": {
      "required": true,
      "contentTypes": [
        "application/json"
      ],
      "schemaRefs": [
        "#/components/schemas/ApplicationInput"
      ]
    }
  },
  {
    "method": "delete",
    "path": "/api/v1/projects/{projectId}/applications/{applicationId}",
    "tags": [
      "Applications"
    ],
    "deprecated": false,
    "security": [],
    "parameters": [
      {
        "name": "projectId",
        "in": "path",
        "required": true,
        "ref": "#/components/parameters/ProjectId",
        "schema": {
          "type": "string"
        }
      },
      {
        "name": "applicationId",
        "in": "path",
        "required": true,
        "ref": "#/components/parameters/ApplicationId",
        "schema": {
          "type": "string"
        }
      }
    ],
    "responses": [
      {
        "status": "204",
        "contentTypes": [],
        "schemaRefs": [],
        "description": "Deleted application."
      }
    ],
    "summary": "Delete application"
  },
  {
    "method": "get",
    "path": "/api/v1/projects/{projectId}/applications/{applicationId}/topology",
    "tags": [
      "Applications"
    ],
    "deprecated": false,
    "security": [],
    "parameters": [
      {
        "name": "projectId",
        "in": "path",
        "required": true,
        "ref": "#/components/parameters/ProjectId",
        "schema": {
          "type": "string"
        }
      },
      {
        "name": "applicationId",
        "in": "path",
        "required": true,
        "ref": "#/components/parameters/ApplicationId",
        "schema": {
          "type": "string"
        }
      }
    ],
    "responses": [
      {
        "status": "200",
        "contentTypes": [
          "application/json"
        ],
        "schemaRefs": [
          "#/components/schemas/ApplicationTopology"
        ],
        "description": "Live application topology. Unavailable deployment targets are returned as warnings while readable targets remain available."
      }
    ],
    "summary": "Compute the current Kubernetes resource topology for an application"
  },
  {
    "method": "get",
    "path": "/api/v1/projects/{projectId}/applications/{applicationId}/deployment-targets",
    "tags": [
      "Deployments"
    ],
    "deprecated": false,
    "security": [],
    "parameters": [
      {
        "name": "projectId",
        "in": "path",
        "required": true,
        "ref": "#/components/parameters/ProjectId",
        "schema": {
          "type": "string"
        }
      },
      {
        "name": "applicationId",
        "in": "path",
        "required": true,
        "ref": "#/components/parameters/ApplicationId",
        "schema": {
          "type": "string"
        }
      }
    ],
    "responses": [
      {
        "status": "200",
        "contentTypes": [
          "application/json"
        ],
        "schemaRefs": [
          "#/components/schemas/DeploymentTarget"
        ],
        "description": "Deployment target list."
      }
    ],
    "summary": "List deployment targets for an application"
  },
  {
    "method": "post",
    "path": "/api/v1/projects/{projectId}/applications/{applicationId}/deployment-targets",
    "tags": [
      "Deployments"
    ],
    "deprecated": false,
    "security": [],
    "parameters": [
      {
        "name": "projectId",
        "in": "path",
        "required": true,
        "ref": "#/components/parameters/ProjectId",
        "schema": {
          "type": "string"
        }
      },
      {
        "name": "applicationId",
        "in": "path",
        "required": true,
        "ref": "#/components/parameters/ApplicationId",
        "schema": {
          "type": "string"
        }
      }
    ],
    "responses": [
      {
        "status": "201",
        "contentTypes": [
          "application/json"
        ],
        "schemaRefs": [
          "#/components/schemas/DeploymentTarget"
        ],
        "description": "Created deployment target."
      }
    ],
    "summary": "Create a deployment target",
    "requestBody": {
      "required": true,
      "contentTypes": [
        "application/json"
      ],
      "schemaRefs": [
        "#/components/schemas/DeploymentTargetInput"
      ]
    }
  },
  {
    "method": "put",
    "path": "/api/v1/projects/{projectId}/applications/{applicationId}/deployment-targets/{targetId}",
    "tags": [
      "Deployments"
    ],
    "deprecated": false,
    "security": [],
    "parameters": [
      {
        "name": "projectId",
        "in": "path",
        "required": true,
        "ref": "#/components/parameters/ProjectId",
        "schema": {
          "type": "string"
        }
      },
      {
        "name": "applicationId",
        "in": "path",
        "required": true,
        "ref": "#/components/parameters/ApplicationId",
        "schema": {
          "type": "string"
        }
      },
      {
        "name": "targetId",
        "in": "path",
        "required": true,
        "ref": "#/components/parameters/TargetId",
        "schema": {
          "type": "string"
        }
      }
    ],
    "responses": [
      {
        "status": "200",
        "contentTypes": [
          "application/json"
        ],
        "schemaRefs": [
          "#/components/schemas/DeploymentTarget"
        ],
        "description": "Updated deployment target."
      }
    ],
    "summary": "Update a deployment target",
    "requestBody": {
      "required": true,
      "contentTypes": [
        "application/json"
      ],
      "schemaRefs": [
        "#/components/schemas/DeploymentTargetInput"
      ]
    }
  },
  {
    "method": "delete",
    "path": "/api/v1/projects/{projectId}/applications/{applicationId}/deployment-targets/{targetId}",
    "tags": [
      "Deployments"
    ],
    "deprecated": false,
    "security": [],
    "parameters": [
      {
        "name": "projectId",
        "in": "path",
        "required": true,
        "ref": "#/components/parameters/ProjectId",
        "schema": {
          "type": "string"
        }
      },
      {
        "name": "applicationId",
        "in": "path",
        "required": true,
        "ref": "#/components/parameters/ApplicationId",
        "schema": {
          "type": "string"
        }
      },
      {
        "name": "targetId",
        "in": "path",
        "required": true,
        "ref": "#/components/parameters/TargetId",
        "schema": {
          "type": "string"
        }
      }
    ],
    "responses": [
      {
        "status": "204",
        "contentTypes": [],
        "schemaRefs": [],
        "description": "Deployment target deletion accepted and queued for asynchronous runtime cleanup."
      }
    ],
    "summary": "Delete a deployment target"
  },
  {
    "method": "get",
    "path": "/api/v1/projects/{projectId}/applications/{applicationId}/deployment-targets/{targetId}/data-export",
    "tags": [
      "Deployments"
    ],
    "deprecated": false,
    "security": [
      {
        "SessionCookie": []
      }
    ],
    "parameters": [
      {
        "name": "projectId",
        "in": "path",
        "required": true,
        "ref": "#/components/parameters/ProjectId",
        "schema": {
          "type": "string"
        }
      },
      {
        "name": "applicationId",
        "in": "path",
        "required": true,
        "ref": "#/components/parameters/ApplicationId",
        "schema": {
          "type": "string"
        }
      },
      {
        "name": "targetId",
        "in": "path",
        "required": true,
        "ref": "#/components/parameters/TargetId",
        "schema": {
          "type": "string"
        }
      },
      {
        "name": "ticket",
        "in": "query",
        "required": true,
        "description": "One-time export ticket returned by the authorize endpoint. It expires after 60 seconds and is consumed even when its resource binding does not match.",
        "schema": {
          "type": "string"
        }
      }
    ],
    "responses": [
      {
        "status": "200",
        "contentTypes": [
          "application/gzip"
        ],
        "schemaRefs": [],
        "description": "Gzip-compressed tar archive streamed as an attachment."
      },
      {
        "status": "400",
        "contentTypes": [
          "application/json"
        ],
        "schemaRefs": [
          "#/components/schemas/ErrorResponse"
        ],
        "description": "Runtime data retention is disabled, the runtime cluster cannot export the target data, or the ticket is missing (`data_export.ticket_required`)."
      },
      {
        "status": "401",
        "contentTypes": [
          "application/json"
        ],
        "schemaRefs": [
          "#/components/schemas/ErrorResponse"
        ],
        "description": "Interactive session cookie is missing or invalid (`auth.session.missing` or another authentication error)."
      },
      {
        "status": "403",
        "contentTypes": [
          "application/json"
        ],
        "schemaRefs": [
          "#/components/schemas/ErrorResponse"
        ],
        "description": "A personal access token was used (`auth.interactive_session_required`), the role is insufficient, MFA is required, or the ticket is invalid/expired/consumed/bound to another request (`data_export.ticket_invalid`)."
      },
      {
        "status": "404",
        "contentTypes": [
          "application/json"
        ],
        "schemaRefs": [
          "#/components/schemas/ErrorResponse"
        ],
        "description": "Project, application, deployment target, or runtime dependency was not found."
      },
      {
        "status": "409",
        "contentTypes": [
          "application/json"
        ],
        "schemaRefs": [
          "#/components/schemas/ErrorResponse"
        ],
        "description": "The project, application, or deployment target is being deleted and cannot be exported."
      },
      {
        "status": "502",
        "contentTypes": [
          "application/json"
        ],
        "schemaRefs": [
          "#/components/schemas/ErrorResponse"
        ],
        "description": "The temporary export Pod or archive stream could not be started (`data_export.stream_failed`)."
      },
      {
        "status": "503",
        "contentTypes": [
          "application/json"
        ],
        "schemaRefs": [
          "#/components/schemas/ErrorResponse"
        ],
        "description": "The shared production ticket store is unavailable (`data_export.ticket_unavailable`)."
      }
    ],
    "summary": "Export persistent runtime data",
    "description": "Consumes a short-lived, one-time export ticket issued by the authorize endpoint, then repeats the interactive session, project Owner/Admin, resource-state, and `data_export` Step-up checks. Personal access tokens are rejected. Each export uses an isolated temporary Pod and streams a gzip archive without persisting the ticket or archive in business tables."
  },
  {
    "method": "post",
    "path": "/api/v1/projects/{projectId}/applications/{applicationId}/deployment-targets/{targetId}/data-export/authorize",
    "tags": [
      "Deployments"
    ],
    "deprecated": false,
    "security": [
      {
        "SessionCookie": []
      }
    ],
    "parameters": [
      {
        "name": "projectId",
        "in": "path",
        "required": true,
        "ref": "#/components/parameters/ProjectId",
        "schema": {
          "type": "string"
        }
      },
      {
        "name": "applicationId",
        "in": "path",
        "required": true,
        "ref": "#/components/parameters/ApplicationId",
        "schema": {
          "type": "string"
        }
      },
      {
        "name": "targetId",
        "in": "path",
        "required": true,
        "ref": "#/components/parameters/TargetId",
        "schema": {
          "type": "string"
        }
      }
    ],
    "responses": [
      {
        "status": "200",
        "contentTypes": [
          "application/json"
        ],
        "schemaRefs": [
          "#/components/schemas/DataExportAuthorization"
        ],
        "description": "Data-export ticket issued. The download endpoint still repeats authorization and atomically consumes the ticket."
      },
      {
        "status": "400",
        "contentTypes": [
          "application/json"
        ],
        "schemaRefs": [
          "#/components/schemas/ErrorResponse"
        ],
        "description": "Runtime data retention is disabled or the runtime cluster cannot export the target data."
      },
      {
        "status": "401",
        "contentTypes": [
          "application/json"
        ],
        "schemaRefs": [
          "#/components/schemas/ErrorResponse"
        ],
        "description": "Interactive browser session is missing or invalid."
      },
      {
        "status": "403",
        "contentTypes": [
          "application/json"
        ],
        "schemaRefs": [
          "#/components/schemas/ErrorResponse"
        ],
        "description": "Project role is insufficient, a personal access token was used, or `data_export` Step-up verification is required."
      },
      {
        "status": "404",
        "contentTypes": [
          "application/json"
        ],
        "schemaRefs": [
          "#/components/schemas/ErrorResponse"
        ],
        "description": "Project, application, deployment target, or runtime dependency was not found."
      },
      {
        "status": "409",
        "contentTypes": [
          "application/json"
        ],
        "schemaRefs": [
          "#/components/schemas/ErrorResponse"
        ],
        "description": "Project, application, or deployment target is being deleted."
      },
      {
        "status": "503",
        "contentTypes": [
          "application/json"
        ],
        "schemaRefs": [
          "#/components/schemas/ErrorResponse"
        ],
        "description": "The shared production ticket store is unavailable (`data_export.ticket_unavailable`)."
      }
    ],
    "summary": "Authorize a persistent runtime data export",
    "description": "Requires an interactive project Owner/Admin session, a mutable project/application/deployment target, exportable runtime data, and an active `data_export` Step-up assertion when the global policy is enabled. Returns a random 60-second one-time ticket bound to the current user, session, project, application, and deployment target. Production uses the shared Redis ticket store and fails closed when Redis is unavailable."
  },
  {
    "method": "post",
    "path": "/api/v1/projects/{projectId}/releases/{releaseId}/terminal/authorize",
    "tags": [
      "Deployments"
    ],
    "deprecated": false,
    "security": [
      {
        "SessionCookie": []
      }
    ],
    "parameters": [
      {
        "name": "projectId",
        "in": "path",
        "required": true,
        "ref": "#/components/parameters/ProjectId",
        "schema": {
          "type": "string"
        }
      },
      {
        "name": "releaseId",
        "in": "path",
        "required": true,
        "ref": "#/components/parameters/ReleaseId",
        "schema": {
          "type": "string"
        }
      }
    ],
    "responses": [
      {
        "status": "204",
        "contentTypes": [],
        "schemaRefs": [],
        "description": "Terminal preflight authorized. The WebSocket endpoint must still perform its own authorization checks."
      },
      {
        "status": "401",
        "contentTypes": [
          "application/json"
        ],
        "schemaRefs": [
          "#/components/schemas/ErrorResponse"
        ],
        "description": "Interactive browser session is missing or invalid."
      },
      {
        "status": "403",
        "contentTypes": [
          "application/json"
        ],
        "schemaRefs": [
          "#/components/schemas/ErrorResponse"
        ],
        "description": "Project role is insufficient, Web Console is disabled (`runtime.web_console_disabled`), a personal access token was used (`mfa.session_required`), or Step-up verification is required (`mfa_required` with purpose `runtime_terminal`)."
      },
      {
        "status": "404",
        "contentTypes": [
          "application/json"
        ],
        "schemaRefs": [
          "#/components/schemas/ErrorResponse"
        ],
        "description": "Project, release, or deployment target was not found."
      },
      {
        "status": "409",
        "contentTypes": [
          "application/json"
        ],
        "schemaRefs": [
          "#/components/schemas/ErrorResponse"
        ],
        "description": "Project or deployment target is being deleted and cannot open Web Console."
      }
    ],
    "summary": "Authorize a release Web Console terminal connection",
    "description": "Normal HTTP preflight used before opening the release terminal WebSocket. It verifies project Owner/Admin/Developer access, project and deployment-target mutation state, the effective project/deployment `webConsoleEnabled` policy, and the `runtime_terminal` Step-up assertion. A missing assertion returns `mfa_required`, allowing the frontend to show the MFA dialog and retry. A 204 authorizes only the preflight; the WebSocket repeats all checks before upgrading and revalidates session, membership, role, resource state, Web Console policy, and assertion every three seconds while connected. Revocation or expiry closes the shell."
  },
  {
    "method": "get",
    "path": "/api/v1/projects/{projectId}/applications/{applicationId}/deployment-targets/{targetId}/release-image-candidates",
    "tags": [
      "Applications"
    ],
    "deprecated": false,
    "security": [],
    "parameters": [
      {
        "name": "projectId",
        "in": "path",
        "required": true,
        "ref": "#/components/parameters/ProjectId",
        "schema": {
          "type": "string"
        }
      },
      {
        "name": "applicationId",
        "in": "path",
        "required": true,
        "ref": "#/components/parameters/ApplicationId",
        "schema": {
          "type": "string"
        }
      },
      {
        "name": "targetId",
        "in": "path",
        "required": true,
        "ref": "#/components/parameters/TargetId",
        "schema": {
          "type": "string"
        }
      }
    ],
    "responses": [
      {
        "status": "200",
        "contentTypes": [
          "application/json"
        ],
        "schemaRefs": [
          "#/components/schemas/ReleaseImageCandidates"
        ],
        "description": "Release image candidates."
      }
    ],
    "summary": "List release image candidates",
    "description": "Reads tags from the target registry first and falls back to saved build records when the registry is unavailable."
  },
  {
    "method": "get",
    "path": "/api/v1/projects/{projectId}/repository-bindings",
    "tags": [
      "Git"
    ],
    "deprecated": false,
    "security": [],
    "parameters": [
      {
        "name": "projectId",
        "in": "path",
        "required": true,
        "ref": "#/components/parameters/ProjectId",
        "schema": {
          "type": "string"
        }
      }
    ],
    "responses": [
      {
        "status": "200",
        "contentTypes": [],
        "schemaRefs": [],
        "description": "Repository binding list."
      }
    ],
    "summary": "List repository bindings for a project"
  },
  {
    "method": "post",
    "path": "/api/v1/projects/{projectId}/repository-bindings",
    "tags": [
      "Git"
    ],
    "deprecated": false,
    "security": [],
    "parameters": [
      {
        "name": "projectId",
        "in": "path",
        "required": true,
        "ref": "#/components/parameters/ProjectId",
        "schema": {
          "type": "string"
        }
      }
    ],
    "responses": [
      {
        "status": "201",
        "contentTypes": [],
        "schemaRefs": [],
        "description": "Created repository binding."
      }
    ],
    "summary": "Bind an application to a Git repository",
    "requestBody": {
      "required": true,
      "contentTypes": [
        "application/json"
      ],
      "schemaRefs": [
        "#/components/schemas/RepositoryBindingInput"
      ]
    }
  },
  {
    "method": "put",
    "path": "/api/v1/projects/{projectId}/repository-bindings/{bindingId}",
    "tags": [
      "Git"
    ],
    "deprecated": false,
    "security": [],
    "parameters": [
      {
        "name": "projectId",
        "in": "path",
        "required": true,
        "ref": "#/components/parameters/ProjectId",
        "schema": {
          "type": "string"
        }
      },
      {
        "name": "bindingId",
        "in": "path",
        "required": true,
        "ref": "#/components/parameters/BindingId",
        "schema": {
          "type": "string"
        }
      }
    ],
    "responses": [
      {
        "status": "200",
        "contentTypes": [],
        "schemaRefs": [],
        "description": "Updated repository binding."
      }
    ],
    "summary": "Update repository binding",
    "requestBody": {
      "required": true,
      "contentTypes": [
        "application/json"
      ],
      "schemaRefs": [
        "#/components/schemas/RepositoryBindingInput"
      ]
    }
  },
  {
    "method": "delete",
    "path": "/api/v1/projects/{projectId}/repository-bindings/{bindingId}",
    "tags": [
      "Git"
    ],
    "deprecated": false,
    "security": [],
    "parameters": [
      {
        "name": "projectId",
        "in": "path",
        "required": true,
        "ref": "#/components/parameters/ProjectId",
        "schema": {
          "type": "string"
        }
      },
      {
        "name": "bindingId",
        "in": "path",
        "required": true,
        "ref": "#/components/parameters/BindingId",
        "schema": {
          "type": "string"
        }
      }
    ],
    "responses": [
      {
        "status": "204",
        "contentTypes": [],
        "schemaRefs": [],
        "description": "Deleted repository binding."
      }
    ],
    "summary": "Delete repository binding"
  },
  {
    "method": "post",
    "path": "/api/v1/projects/{projectId}/repository-bindings/{bindingId}/webhook",
    "tags": [
      "Git"
    ],
    "deprecated": false,
    "security": [],
    "parameters": [
      {
        "name": "projectId",
        "in": "path",
        "required": true,
        "ref": "#/components/parameters/ProjectId",
        "schema": {
          "type": "string"
        }
      },
      {
        "name": "bindingId",
        "in": "path",
        "required": true,
        "ref": "#/components/parameters/BindingId",
        "schema": {
          "type": "string"
        }
      }
    ],
    "responses": [
      {
        "status": "200",
        "contentTypes": [],
        "schemaRefs": [],
        "description": "Created repository webhook."
      }
    ],
    "summary": "Create webhook for a repository binding"
  },
  {
    "method": "get",
    "path": "/api/v1/access-tokens",
    "tags": [
      "AccessTokens"
    ],
    "deprecated": false,
    "security": [],
    "parameters": [
      {
        "name": "page",
        "in": "query",
        "ref": "#/components/parameters/Page",
        "schema": {
          "type": "integer"
        }
      },
      {
        "name": "pageSize",
        "in": "query",
        "ref": "#/components/parameters/PageSize",
        "schema": {
          "type": "integer"
        }
      },
      {
        "name": "sortBy",
        "in": "query",
        "schema": {
          "type": "string",
          "enum": [
            "createdAt",
            "expiresAt",
            "name",
            "scope",
            "status"
          ]
        }
      },
      {
        "name": "sortOrder",
        "in": "query",
        "ref": "#/components/parameters/SortOrder",
        "schema": {
          "type": "string",
          "enum": [
            "asc",
            "desc"
          ]
        }
      }
    ],
    "responses": [
      {
        "status": "200",
        "contentTypes": [],
        "schemaRefs": [],
        "description": "Paginated access token list."
      }
    ],
    "summary": "List access tokens",
    "description": "Returns only non-revoked access tokens."
  },
  {
    "method": "post",
    "path": "/api/v1/access-tokens",
    "tags": [
      "AccessTokens"
    ],
    "deprecated": false,
    "security": [],
    "parameters": [],
    "responses": [
      {
        "status": "201",
        "contentTypes": [],
        "schemaRefs": [],
        "description": "Created access token with one-time secret."
      }
    ],
    "summary": "Create access token",
    "requestBody": {
      "required": true,
      "contentTypes": [
        "application/json"
      ],
      "schemaRefs": [
        "#/components/schemas/AccessTokenInput"
      ]
    }
  },
  {
    "method": "delete",
    "path": "/api/v1/access-tokens/{tokenId}",
    "tags": [
      "AccessTokens"
    ],
    "deprecated": false,
    "security": [],
    "parameters": [
      {
        "name": "tokenId",
        "in": "path",
        "required": true,
        "ref": "#/components/parameters/TokenId",
        "schema": {
          "type": "string"
        }
      }
    ],
    "responses": [
      {
        "status": "204",
        "contentTypes": [],
        "schemaRefs": [],
        "description": "Revoked access token."
      }
    ],
    "summary": "Revoke access token"
  }
] as const satisfies readonly OpenApiOperationSnapshot[];
