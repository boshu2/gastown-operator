apiVersion: gastown.gastown.io/v1alpha1
kind: Rig
metadata:
  name: local-smoke
spec:
  gitURL: "GIT_REPO_PLACEHOLDER"
  beadsPrefix: "ls"
  settings:
    namepoolTheme: "mad-max"
    maxPolecats: 2
---
apiVersion: gastown.gastown.io/v1alpha1
kind: Polecat
metadata:
  name: madeye-smoke
  namespace: gastown-system
spec:
  rig: local-smoke
  desiredState: Working
  beadID: ls-001
  taskDescription: |
    Smoke test only: reply confirming you can see this task and which model
    you appear to be using. Do not modify repository files. Finish quickly.
  executionMode: kubernetes
  agent: claude-code
  agentConfig:
    provider: litellm
    model: claude-opus-4-8
    modelProvider:
      endpoint: http://litellm.gastown-system.svc:4000
      apiKeySecretRef:
        name: litellm-auth
        key: master-key
    env:
      - name: CLAUDE_CODE_ENABLE_GATEWAY_MODEL_DISCOVERY
        value: "1"
  kubernetes:
    gitRepository: "GIT_REPO_PLACEHOLDER"
    gitBranch: "GIT_BRANCH_PLACEHOLDER"
    workBranch: feature/ls-001-smoke
    gitSecretRef:
      name: git-creds
    # Satisfies auth validation; gateway Bearer token comes from agentConfig above
    apiKeySecretRef:
      name: litellm-auth
      key: master-key
    activeDeadlineSeconds: 900
    resources:
      requests:
        cpu: 250m
        memory: 512Mi
      limits:
        cpu: "1"
        memory: 2Gi
