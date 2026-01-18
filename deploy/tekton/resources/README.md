# Namespace Resources for Labyrinth CI

This document describes the Kubernetes resources needed to run the
Labyrinth Tekton pipeline.

## GitLab CI Variables

The following variables must be set in GitLab CI/CD settings:

| Variable | Required | Purpose |
|----------|----------|---------|
| `KUBE_CONFIG` | Yes | Base64-encoded kubeconfig for olympus-ci namespace |
| `REGISTRY_URL` | Yes | Container registry URL (e.g., `dprusocplvjmp01.deepsky.lab:5000`) |
| `REGISTRY_USER` | Yes | Registry username |
| `REGISTRY_PASSWORD` | Yes | Registry password |
| `GITHUB_DEPLOY_KEY` | Yes | Base64-encoded SSH deploy key for GitHub push (code sync) |
| `GITHUB_TOKEN` | Yes | GitHub PAT with `write:packages` scope (GHCR push) |
| `TEKTON_NAMESPACE` | No | Override default (`olympus-ci`) |
| `TRIVY_SEVERITY` | No | Override scan threshold (default: `CRITICAL,HIGH`) |

### Setting Up GitHub Integration

**1. Create GitHub Deploy Key (for code sync):**

```bash
# Generate SSH key
ssh-keygen -t ed25519 -C "gitlab-ci" -f gitlab-deploy-key

# Add public key to GitHub repo: Settings → Deploy keys → Add (with write access)
# Base64 encode private key for GitLab CI variable
cat gitlab-deploy-key | base64 -w0
```

**2. Create GitHub PAT (for GHCR push):**

- Go to GitHub: Settings → Developer settings → Personal access tokens
- Create token with `write:packages` scope
- Add as `GITHUB_TOKEN` in GitLab CI variables

## Required Resources

### 1. CI Namespace

Create a namespace for your CI pipelines:

```bash
kubectl create namespace olympus-ci
```

### 2. Registry Credentials Secret

The pipeline pushes to DPR (primary) and mirrors to GHCR (public).
Both registries must be in the same secret. See `registry-secret.yaml`
for detailed instructions.

**Quick setup for dual-registry (DPR + GHCR):**

```bash
# Create temp files for each registry
kubectl create secret docker-registry dpr-creds \
  --docker-server=dprusocplvjmp01.deepsky.lab:5000 \
  --docker-username=${DPR_USER} \
  --docker-password=${DPR_PASSWORD} \
  -n olympus-ci --dry-run=client -o json > /tmp/dpr.json

kubectl create secret docker-registry ghcr-creds \
  --docker-server=ghcr.io \
  --docker-username=${GITHUB_USER} \
  --docker-password=${GITHUB_TOKEN} \
  -n olympus-ci --dry-run=client -o json > /tmp/ghcr.json

# Merge into single secret
jq -s '.[0] * {
  "metadata": {"name": "registry-credentials"},
  "data": {
    ".dockerconfigjson": (
      [.[].data[".dockerconfigjson"] | @base64d | fromjson] |
      reduce .[] as $x ({}; .auths += $x.auths) |
      @json | @base64
    )
  }
}' /tmp/dpr.json /tmp/ghcr.json | kubectl apply -n olympus-ci -f -

# Cleanup
rm /tmp/dpr.json /tmp/ghcr.json
```

**Single registry examples:**

```bash
# GitHub Container Registry (ghcr.io)
kubectl create secret docker-registry registry-credentials \
  --docker-server=ghcr.io \
  --docker-username=${GITHUB_USER} \
  --docker-password=${GITHUB_TOKEN} \
  -n olympus-ci

# Docker Hub
kubectl create secret docker-registry registry-credentials \
  --docker-server=docker.io \
  --docker-username=${DOCKER_USER} \
  --docker-password=${DOCKER_PASSWORD} \
  -n olympus-ci

# Quay.io
kubectl create secret docker-registry registry-credentials \
  --docker-server=quay.io \
  --docker-username=${QUAY_USER} \
  --docker-password=${QUAY_PASSWORD} \
  -n olympus-ci
```

### 3. ServiceAccount

The `pipeline` ServiceAccount is typically created automatically when
Tekton Pipelines is installed. Verify it exists:

```bash
kubectl get sa pipeline -n olympus-ci
```

If missing, create it:

```bash
kubectl create serviceaccount pipeline -n olympus-ci
```

### 4. Storage Class (Optional)

The PipelineRun uses dynamic volume provisioning. If your cluster
doesn't have a default StorageClass, either:

1. Set a default StorageClass:
   ```bash
   kubectl patch storageclass <YOUR_SC> -p '{"metadata": {"annotations":{"storageclass.kubernetes.io/is-default-class":"true"}}}'
   ```

2. Or update the PipelineRun to specify your StorageClass explicitly.

## Verification Script

Run this to check all required resources:

```bash
#!/bin/bash
NS=olympus-ci

echo "Checking namespace..."
kubectl get namespace $NS || echo "ERROR: namespace missing"

echo ""
echo "Checking ServiceAccount..."
kubectl get sa pipeline -n $NS || echo "ERROR: pipeline SA missing"

echo ""
echo "Checking registry credentials..."
kubectl get secret registry-credentials -n $NS || \
  echo "WARNING: No registry credentials found"

echo ""
echo "Checking default StorageClass..."
kubectl get storageclass -o jsonpath='{.items[?(@.metadata.annotations.storageclass\.kubernetes\.io/is-default-class=="true")].metadata.name}' || \
  echo "WARNING: No default StorageClass found"
```

## ClusterTask Requirements

The pipeline uses these ClusterTasks from the Tekton catalog:

| ClusterTask | Tekton Hub Link |
|-------------|-----------------|
| git-clone | https://hub.tekton.dev/tekton/task/git-clone |
| kaniko | https://hub.tekton.dev/tekton/task/kaniko |
| trivy-scanner | https://hub.tekton.dev/tekton/task/trivy-scanner |
| hadolint | https://hub.tekton.dev/tekton/task/hadolint |

Install them with:

```bash
# Using tkn CLI
tkn hub install task git-clone
tkn hub install task kaniko
tkn hub install task trivy-scanner
tkn hub install task hadolint

# Or apply directly
kubectl apply -f https://raw.githubusercontent.com/tektoncd/catalog/main/task/git-clone/0.9/git-clone.yaml
kubectl apply -f https://raw.githubusercontent.com/tektoncd/catalog/main/task/kaniko/0.6/kaniko.yaml
# etc.
```
