#!/usr/bin/env bash
# Build frontend static assets for Docker / release packaging.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

if [ -z "${VITE_FRONTEND_COMMIT:-}" ]; then
	# shellcheck source=/dev/null
	eval "$("$PROJECT_ROOT/scripts/get_version.sh" env)"
	export VITE_FRONTEND_COMMIT="${COMMIT_ID:-unknown}"
fi

export VITE_IS_DOCKER="${VITE_IS_DOCKER:-true}"

BRAND_PROFILE="${XELORA_BRAND_PROFILE:-${VITE_BRAND_PROFILE:-public}}"
case "$BRAND_PROFILE" in
	public|internal)
		;;
	*)
		echo "XELORA_BRAND_PROFILE must be public or internal (got: $BRAND_PROFILE)" >&2
		exit 1
		;;
esac
export VITE_BRAND_PROFILE="$BRAND_PROFILE"

cd "$PROJECT_ROOT/frontend"
npm ci
npm run build
