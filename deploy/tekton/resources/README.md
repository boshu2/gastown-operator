# Namespace Resources for gastown-operator CI

The gastown-operator pipeline runs in the shared `olympus-ci` namespace.
Most resources are already created by hephaestus (kagent). This document
describes what's needed and how to verify/create them.

## Required Resources

### 1. Registry Credentials Secret

Allows Kaniko to push images to DPR.

```bash
# Check if exists (shared with hephaestus)
oc get secret gastown-registry-credentials -n olympus-ci

# Create if missing (same as kagent, just different name)
oc create secret docker-registry gastown-registry-credentials \
  --docker-server=dprusocplvjmp01.deepsky.lab:5000 \
  --docker-username=${DPR_USER} \
  --docker-password=${DPR_PASSWORD} \
  -n olympus-ci
```

**Note:** You can also reuse `kagent-registry-credentials` if it exists,
by updating the PipelineRun to reference it instead.

### 2. ServiceAccount (Shared)

The `pipeline` ServiceAccount is typically created automatically when
Tekton Pipelines is installed. Verify it exists:

```bash
oc get sa pipeline -n olympus-ci
```

### 3. Storage Class (Cluster-Level)

The `flash` StorageClass is used for workspace PVCs. This is a cluster-level
resource that should already exist:

```bash
oc get storageclass flash
```

If not available, use `gp2` or your cluster's default StorageClass by updating
the PipelineRun template.

## Optional Resources

### SonarQube Token (Optional)

For SAST scanning integration. Skip if not using SonarQube:

```bash
oc create secret generic sonar-token \
  --from-literal=SONAR_TOKEN=xxx \
  -n olympus-ci
```

## Verification Script

Run this to check all required resources:

```bash
#!/bin/bash
NS=olympus-ci

echo "Checking namespace..."
oc get namespace $NS || echo "ERROR: namespace missing"

echo ""
echo "Checking ServiceAccount..."
oc get sa pipeline -n $NS || echo "ERROR: pipeline SA missing"

echo ""
echo "Checking registry credentials..."
oc get secret gastown-registry-credentials -n $NS 2>/dev/null || \
  oc get secret kagent-registry-credentials -n $NS 2>/dev/null || \
  echo "WARNING: No registry credentials found"

echo ""
echo "Checking StorageClass..."
oc get storageclass flash || echo "WARNING: flash StorageClass not found"
```

## Resource Sharing with hephaestus

The gastown-operator pipeline shares the `olympus-ci` namespace with
hephaestus (kagent). Resources that can be shared:

| Resource | hephaestus Name | gastown-operator | Shared? |
|----------|-----------------|------------------|---------|
| Registry Secret | kagent-registry-credentials | gastown-registry-credentials | No (separate) |
| ServiceAccount | pipeline | pipeline | Yes |
| StorageClass | flash | flash | Yes (cluster) |
| SonarQube Token | sonar-token | sonar-token | Yes (optional) |

Using separate registry secrets allows independent credential rotation.
