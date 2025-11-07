#!/usr/bin/env bash
#
# scale-to-zero-hyperion-dev.sh
#
#   Set replicas of all Deployments, StatefulSets and ReplicationControllers
#   in the "hyperion-dev" namespace to 0.
#
#   Requirements:
#     * kubectl in $PATH
#     * A kubeconfig/context that can access the target cluster
#
#   Usage:
#       ./scale-to-zero-hyperion-dev.sh [-n <namespace>] [-d] [-f]
#
#   Options:
#     -n <ns>   Namespace to act on (default: hyperion-dev)
#     -d        Dry‑run only – show what would be changed, don't apply.
#     -f        Force – skip the "are you sure?" prompt.
#
#   Example:
#       ./scale-to-zero-hyperion-dev.sh -d   # dry‑run
#       ./scale-to-zero-hyperion-dev.sh -f   # no interactive prompt
#

set -euo pipefail

# ---------- Configurable defaults ----------
NAMESPACE="hyperion-dev"
DRY_RUN=false
FORCE=false

# ---------- Helper functions ----------
usage() {
    grep '^##' "$0" | cut -c4-
    exit 1
}

log()   { printf "[%s] %s\n" "$(date +%H:%M:%S)" "$*"; }
warn()  { log "WARN: $*"; }
error() { log "ERROR: $*"; exit 1; }

# ---------- Parse flags ----------
while getopts ":n:dfh" opt; do
    case $opt in
        n) NAMESPACE="${OPTARG}" ;;
        d) DRY_RUN=true ;;
        f) FORCE=true ;;
        h) usage ;;
        \?) error "Invalid option: -$OPTARG" ;;
        :)  error "Option -$OPTARG requires an argument." ;;
    esac
done
shift $((OPTIND-1))

# ---------- Verify kubectl ----------
if ! command -v kubectl >/dev/null 2>&1; then
    error "kubectl not found in PATH"
fi

# ---------- Check namespace existence ----------
if ! kubectl get namespace "${NAMESPACE}" >/dev/null 2>&1; then
    error "Namespace '${NAMESPACE}' does not exist or you lack permission to view it."
fi

# ---------- Prompt (unless forced) ----------
if [[ "$FORCE" = false ]]; then
    cat <<EOF
You are about to scale **ALL** Deployments, StatefulSets and ReplicationControllers
in namespace '${NAMESPACE}' to 0 replicas.

$(if $DRY_RUN; then echo "  → DRY‑RUN mode – no changes will be applied."; else echo "  → THIS WILL MODIFY LIVE RESOURCES!"; fi)

Proceed? (y/N) 
EOF
    read -r answer
    if [[ ! "$answer" =~ ^[Yy]$ ]]; then
        log "Aborted by user."
        exit 0
    fi
fi

# ---------- Gather resources ----------
log "Collecting scalable resources in namespace '${NAMESPACE}' …"

# Use jsonpath to extract name and kind in a tab‑separated format.
# We include Deployments, StatefulSets, and ReplicationControllers.
# (If you also want DaemonSets, remove the `replicas` field – they don't have it.)
resources=$(kubectl get deployment,statefulset,rc \
    -n "${NAMESPACE}" \
    -o jsonpath='{range .items[*]}{.kind}{"\t"}{.metadata.name}{"\n"}{end}')

if [[ -z "$resources" ]]; then
    log "No Deployments, StatefulSets or ReplicationControllers found in ${NAMESPACE}."
    exit 0
fi

# ---------- Scale loop ----------
IFS=$'\n'   # split on newlines only
for line in $resources; do
    kind=$(awk '{print $1}' <<<"$line")
    name=$(awk '{print $2}' <<<"$line")

    # Determine the appropriate kubectl sub‑command for the kind
    case "$kind" in
        Deployment)  cmd="deployment" ;;
        StatefulSet) cmd="statefulset" ;;
        ReplicationController) cmd="rc" ;;
        *) warn "Unsupported kind $kind – skipping"; continue ;;
    esac

    # Build the kubectl command
    kubectl_cmd=(
        kubectl scale "${cmd}/${name}"
        --replicas=0
        -n "${NAMESPACE}"
    )
    $DRY_RUN && kubectl_cmd+=(--dry-run=client)

    # Run it and capture any error
    if output=$("${kubectl_cmd[@]}" 2>&1); then
        log "Scaled ${kind}/${name} → 0"
    else
        warn "Failed to scale ${kind}/${name}: $output"
    fi
done

log "Done."
