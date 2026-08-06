#!/usr/bin/env python3
"""Local dev: catch GenAxis webhooks + save W4 scripts to ~/Desktop/bte-script."""
from http.server import HTTPServer, BaseHTTPRequestHandler
import json
import os
from pathlib import Path

OUT_DIR = Path(os.environ.get("BTE_SCRIPT_DIR", Path.home() / "Desktop" / "bte-script"))
OUT_DIR.mkdir(parents=True, exist_ok=True)


class Handler(BaseHTTPRequestHandler):
    def do_POST(self):
        length = int(self.headers.get("Content-Length", 0))
        body = self.rfile.read(length)
        data = json.loads(body) if body else {}

        if self.path.rstrip("/") == "/v1/bte/internal/dev-save":
            filename = data.get("filename") or f"{data.get('job_name', 'script')}.md"
            content = data.get("content", "")
            dest = OUT_DIR / Path(filename).name
            dest.write_text(content, encoding="utf-8")
            print(f"\n=== DEV-SAVE ===\n  => {dest}\n  bytes={len(content.encode('utf-8'))}\n")
            self._ok({"saved": str(dest)})
            return

        if self.path.rstrip("/") == "/v1/bte/internal/webhook":
            print("\n=== WEBHOOK ===\n", json.dumps(data, indent=2))
            self._ok({"ok": True})
            return

        print(f"\n=== POST {self.path} ===\n", json.dumps(data, indent=2))
        self._ok({"ok": True})

    def _ok(self, payload):
        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.end_headers()
        self.wfile.write(json.dumps(payload).encode())

    def log_message(self, *args):
        pass


if __name__ == "__main__":
    port = int(os.environ.get("PORT", "9999"))
    print(f"Listening :{port}")
    print(f"  webhooks => /v1/bte/internal/webhook")
    print(f"  W4 save  => /v1/bte/internal/dev-save  -> {OUT_DIR}/")
    HTTPServer(("0.0.0.0", port), Handler).serve_forever()
