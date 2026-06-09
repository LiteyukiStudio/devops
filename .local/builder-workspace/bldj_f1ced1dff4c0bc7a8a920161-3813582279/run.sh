#!/bin/sh
set -eu

cd /workspace

AUTH_CLONE_URL="$GIT_CLONE_URL"
if [ -n "${GIT_ACCESS_TOKEN:-}" ]; then
  case "$AUTH_CLONE_URL" in
    https://*) AUTH_CLONE_URL="$(printf "%s" "$AUTH_CLONE_URL" | sed "s#https://#https://x-access-token:${GIT_ACCESS_TOKEN}@#")" ;;
  esac
fi

clone_with_retry() {
  attempt=1
  while [ "$attempt" -le 3 ]; do
    rm -rf source
    if [ -n "${SOURCE_BRANCH:-}" ]; then
      if git clone --depth 1 --single-branch --branch "$SOURCE_BRANCH" "$AUTH_CLONE_URL" source; then
        return 0
      fi
    else
      if git clone --depth 1 "$AUTH_CLONE_URL" source; then
        return 0
      fi
    fi
    if [ "$attempt" -eq 3 ]; then
      return 1
    fi
    sleep $((attempt * 2))
    attempt=$((attempt + 1))
  done
}

clone_with_retry
cd source

if [ -n "${SOURCE_TAG:-}" ]; then git checkout "$SOURCE_TAG"; fi
if [ -n "${SOURCE_BRANCH:-}" ]; then git checkout "$SOURCE_BRANCH"; fi
if [ -n "${SOURCE_COMMIT:-}" ]; then git checkout "$SOURCE_COMMIT"; fi

CHECKED_OUT_COMMIT="$(git rev-parse HEAD)"
SOURCE_AUTHOR_NAME="$(git log -1 --format=%an)"
SOURCE_AUTHOR_EMAIL="$(git log -1 --format=%ae)"

short_commit() {
  value="$1"
  printf "%s" "$(printf "%.12s" "$value")"
}

json_escape() {
  printf "%s" "$1" | tr '\n\r' '  ' | sed 's/\\/\\\\/g; s/"/\\"/g'
}

sanitize_tag() {
  printf "%s" "$1" \
    | sed -E 's/[^A-Za-z0-9_.-]+/-/g; s/^[.-]+//; s/[.-]+$//' \
    | cut -c 1-128
}

escape_sed_pattern() {
  printf "%s" "$1" | sed 's/[][\/.^$*+?{}()|]/\\&/g'
}

escape_sed_replacement() {
  printf "%s" "$1" | sed 's/[\/&]/\\&/g'
}

replace_token() {
  pattern="$(escape_sed_pattern "$1")"
  replacement="$(escape_sed_replacement "$2")"
  printf "%s" "$3" | sed -E "s/$pattern/$replacement/g"
}

render_image_tag() {
  template="${IMAGE_TAG_TEMPLATE:-latest}"
  ref_name="${SOURCE_TAG:-$SOURCE_BRANCH}"
  ref_type="branch"
  ref_value=""
  if [ -n "${SOURCE_TAG:-}" ]; then
    ref_type="tag"
    ref_value="refs/tags/$SOURCE_TAG"
  elif [ -n "${SOURCE_BRANCH:-}" ]; then
    ref_value="refs/heads/$SOURCE_BRANCH"
  fi
  rendered="$template"
  short_sha="$(short_commit "$CHECKED_OUT_COMMIT")"
  rendered="$(replace_token '${{ github.sha }}' "$CHECKED_OUT_COMMIT" "$rendered")"
  rendered="$(replace_token '${{ github.ref_name }}' "$ref_name" "$rendered")"
  rendered="$(replace_token '${{ github.ref_type }}' "$ref_type" "$rendered")"
  rendered="$(replace_token '${{ github.ref }}' "$ref_value" "$rendered")"
  rendered="$(replace_token '{short_sha}' "$short_sha" "$rendered")"
  sanitized="$(sanitize_tag "$rendered")"
  if [ -z "$sanitized" ]; then
    sanitized="latest"
  fi
  printf "%s" "$sanitized"
}

if [ -n "${IMAGE_NAME_PREFIX:-}" ]; then
  IMAGE_REF="${IMAGE_NAME_PREFIX}:$(render_image_tag)"
fi

if [ -n "${NPM_REGISTRY:-}" ]; then
  mkdir -p "$PWD/$BUILD_CONTEXT"
  printf "registry=%s\n" "$NPM_REGISTRY" > "$PWD/$BUILD_CONTEXT/.npmrc"
fi

set --
OLDIFS="$IFS"
IFS=","
for key in ${BUILD_ENV_KEYS:-}; do
  if [ -z "$key" ]; then
    continue
  fi
  eval "value=\${$key:-}"
  set -- "$@" --opt "build-arg:${key}=${value}"
done
IFS="$OLDIFS"

mkdir -p "$HOME/.docker"
AUTH="$(printf "%s:%s" "$REGISTRY_USERNAME" "$REGISTRY_PASSWORD" | base64 | tr -d "\n")"
printf '{"auths":{"%s":{"auth":"%s"}}}' "$REGISTRY_ENDPOINT" "$AUTH" > "$HOME/.docker/config.json"

build_with_retry() {
  attempt=1
  while [ "$attempt" -le 3 ]; do
    if buildctl-daemonless.sh build \
      --progress=plain \
      --frontend dockerfile.v0 \
      --local context="$PWD/$BUILD_CONTEXT" \
      --local dockerfile="$PWD/$(dirname "$DOCKERFILE_PATH")" \
      --opt filename="$(basename "$DOCKERFILE_PATH")" \
      "$@" \
      --output type=image,name="$IMAGE_REF",push=true; then
      return 0
    fi
    if [ "$attempt" -eq 3 ]; then
      return 1
    fi
    sleep $((attempt * 3))
    attempt=$((attempt + 1))
  done
}

build_with_retry "$@"

printf '{"imageRef":"%s","sourceCommit":"%s","sourceAuthorName":"%s","sourceAuthorEmail":"%s","message":"builder task succeeded"}' "$(json_escape "$IMAGE_REF")" "$(json_escape "$CHECKED_OUT_COMMIT")" "$(json_escape "$SOURCE_AUTHOR_NAME")" "$(json_escape "$SOURCE_AUTHOR_EMAIL")" > /workspace/result.json
