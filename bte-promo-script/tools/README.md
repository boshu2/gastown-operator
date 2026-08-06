# BTE polecat tools

CLI tools baked into the `polecat-agent` image and run by Claude inside polecat pods.

| Tool | Path | Binary |
|------|------|--------|
| promo-finish | `tools/finish/` | `/usr/local/bin/promo-finish` |

Add new tools here as sibling directories; wire each into `images/polecat-agent/Dockerfile`.
