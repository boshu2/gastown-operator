# Customizing the Polecat Agent Image

The base `polecat-agent` image is minimal by design. It includes git, curl, jq, and Claude Code - enough to clone repos and run the agent. If your polecats need additional tools, you have two options.

## Option 1: Install at Runtime

For quick experiments, install tools when the polecat starts:

```bash
# In your polecat's initial prompt or CLAUDE.md
apt-get update && apt-get install -y python3 python3-pip
```

**Pros:** No image building required.
**Cons:** Slower startup, requires network access, not reproducible.

## Option 2: Build Your Own Image (Recommended)

Create a custom image with your tools pre-installed.

### Python Stack

```dockerfile
FROM ghcr.io/boshu2/polecat-agent:latest

USER root
RUN apt-get update && apt-get install -y --no-install-recommends \
    python3 \
    python3-pip \
    python3-venv \
    && rm -rf /var/lib/apt/lists/*
USER polecat
```

### Go Stack

```dockerfile
FROM ghcr.io/boshu2/polecat-agent:latest

USER root
RUN apt-get update && apt-get install -y --no-install-recommends \
    golang \
    && rm -rf /var/lib/apt/lists/*
ENV PATH="/usr/local/go/bin:${PATH}"
USER polecat
```

### Node.js Stack

```dockerfile
FROM ghcr.io/boshu2/polecat-agent:latest

USER root
RUN apt-get update && apt-get install -y --no-install-recommends \
    nodejs \
    npm \
    && rm -rf /var/lib/apt/lists/*
USER polecat
```

### Full Dev Stack

```dockerfile
FROM ghcr.io/boshu2/polecat-agent:latest

USER root
RUN apt-get update && apt-get install -y --no-install-recommends \
    python3 \
    python3-pip \
    golang \
    nodejs \
    npm \
    make \
    gcc \
    && rm -rf /var/lib/apt/lists/*
USER polecat
```

## Building and Using Your Image

```bash
# Build
docker build -t my-polecat:latest -f Dockerfile.custom .

# Push to your registry
docker tag my-polecat:latest ghcr.io/myorg/my-polecat:latest
docker push ghcr.io/myorg/my-polecat:latest

# Use with operator
kubectl apply -f - <<EOF
apiVersion: gastown.gastown.io/v1alpha1
kind: Polecat
metadata:
  name: my-polecat
spec:
  execution:
    kubernetes:
      image: "ghcr.io/myorg/my-polecat:latest"
EOF
```

## Tips

- **Always switch back to `USER polecat`** - the agent runs as non-root for security
- **Use `--no-install-recommends`** - keeps image size down
- **Clean apt cache** - `rm -rf /var/lib/apt/lists/*` saves space
- **Pin versions** - use `python3=3.11.*` for reproducibility

## Need Help?

Tell Claude: "Build me a polecat image with Python and Go." It'll generate the Dockerfile.
