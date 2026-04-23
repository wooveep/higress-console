#!/bin/sh
set -eu

SCRIPT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
ROOT_DIR="$(CDPATH= cd -- "$SCRIPT_DIR/../.." && pwd)"
IMAGE_NAME="${IMAGE_NAME:-aigateway/console:dev}"
RELEASE_NAME="${RELEASE_NAME:-aigateway}"
NAMESPACE="${NAMESPACE:-aigateway-system}"
CONSOLE_DEPLOYMENT_NAME="${CONSOLE_DEPLOYMENT_NAME:-aigateway-console}"
IMAGE_PULL_POLICY="${IMAGE_PULL_POLICY:-IfNotPresent}"
LOAD_IMAGE="${LOAD_IMAGE:-true}"
LOAD_TOOL="${LOAD_TOOL:-auto}"
HELM_TIMEOUT="${HELM_TIMEOUT:-10m}"
# Keep a single chart source of truth in higress/helm/higress.
DEFAULT_HIGRESS_CHART_DIR="$ROOT_DIR/higress/helm/higress"
if [ ! -d "$DEFAULT_HIGRESS_CHART_DIR" ]; then
    DEFAULT_HIGRESS_CHART_DIR="$ROOT_DIR/helm/higress"
fi
HIGRESS_CHART_DIR="${HIGRESS_CHART_DIR:-$DEFAULT_HIGRESS_CHART_DIR}"
if [ -d "$HIGRESS_CHART_DIR" ]; then
    HIGRESS_CHART_DIR="$(CDPATH= cd -- "$HIGRESS_CHART_DIR" && pwd -P)"
fi

load_image() {
    if [ "$LOAD_IMAGE" != "true" ]; then
        return 0
    fi

    current_context="$(kubectl config current-context 2>/dev/null || true)"
    tool="$LOAD_TOOL"

    if [ "$tool" = "auto" ]; then
        case "$current_context" in
            minikube)
                tool="minikube"
                ;;
            kind|kind-*)
                tool="kind"
                ;;
            *)
                tool=""
                ;;
        esac
    fi

    case "$tool" in
        minikube)
            echo "Loading image into minikube: $IMAGE_NAME"
            minikube image load "$IMAGE_NAME"
            ;;
        kind)
            echo "Loading image into kind: $IMAGE_NAME"
            kind load docker-image "$IMAGE_NAME"
            ;;
        "")
            echo "Skipping cluster image load for context: ${current_context:-unknown}"
            ;;
        *)
            echo "Unsupported LOAD_TOOL: $tool" >&2
            exit 1
            ;;
    esac
}

parse_image_repository() {
    image_ref="${IMAGE_NAME%@*}"
    last_segment="${image_ref##*/}"
    case "$last_segment" in
        *:*)
            printf '%s\n' "${image_ref%:*}"
            ;;
        *)
            echo "IMAGE_NAME must include an explicit tag: $IMAGE_NAME" >&2
            exit 1
            ;;
    esac
}

parse_image_tag() {
    image_ref="${IMAGE_NAME%@*}"
    last_segment="${image_ref##*/}"
    case "$last_segment" in
        *:*)
            printf '%s\n' "${last_segment##*:}"
            ;;
        *)
            echo "IMAGE_NAME must include an explicit tag: $IMAGE_NAME" >&2
            exit 1
            ;;
    esac
}

if [ ! -d "$HIGRESS_CHART_DIR" ]; then
    echo "HIGRESS_CHART_DIR does not exist: $HIGRESS_CHART_DIR" >&2
    exit 1
fi

load_image

IMAGE_REPOSITORY="$(parse_image_repository)"
IMAGE_TAG="$(parse_image_tag)"

helm dependency build "$HIGRESS_CHART_DIR"

helm upgrade "$RELEASE_NAME" "$HIGRESS_CHART_DIR" \
    --namespace "$NAMESPACE" \
    --reuse-values \
    --timeout "$HELM_TIMEOUT" \
    --set-string aigateway-console.image.repository="$IMAGE_REPOSITORY" \
    --set-string aigateway-console.image.tag="$IMAGE_TAG" \
    --set-string aigateway-console.image.pullPolicy="$IMAGE_PULL_POLICY"

kubectl -n "$NAMESPACE" rollout status "deployment/$CONSOLE_DEPLOYMENT_NAME" --timeout=180s
