#!/bin/sh
set -eu

SCRIPT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
CHART_DIR="${CHART_DIR:-$SCRIPT_DIR}"
IMAGE_NAME="${IMAGE_NAME:-aigateway/console:dev}"
RELEASE_NAME="${RELEASE_NAME:-aigateway-console}"
NAMESPACE="${NAMESPACE:-aigateway-system}"
IMAGE_PULL_POLICY="${IMAGE_PULL_POLICY:-IfNotPresent}"
LOAD_IMAGE="${LOAD_IMAGE:-true}"
LOAD_TOOL="${LOAD_TOOL:-auto}"
HELM_TIMEOUT="${HELM_TIMEOUT:-5m}"

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

load_image

IMAGE_REPOSITORY="$(parse_image_repository)"
IMAGE_TAG="$(parse_image_tag)"

helm upgrade --install "$RELEASE_NAME" "$CHART_DIR" \
    --namespace "$NAMESPACE" \
    --create-namespace \
    --timeout "$HELM_TIMEOUT" \
    --set image.repository="$IMAGE_REPOSITORY" \
    --set image.tag="$IMAGE_TAG" \
    --set image.pullPolicy="$IMAGE_PULL_POLICY"
