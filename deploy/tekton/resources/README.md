# Namespace Resources for Labyrinth CI

This document describes the Kubernetes resources needed to run the
Labyrinth Tekton pipeline.

## Required Resources

### 1. CI Namespace

Create a namespace for your CI pipelines:

```bash
kubectl create namespace olympus-ci
```

### 2. Registry Credentials Secret

Allows Kaniko to push images to your container registry:

```bash
kubectl create secret docker-registry registry-credentials \
  --docker-server=<YOUR_REGISTRY> \
  --docker-username=${REGISTRY_USER} \
  --docker-password=${REGISTRY_PASSWORD} \
  -n olympus-ci
```

**Examples for common registries:**

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
