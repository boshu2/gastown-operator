"""HTTP front-proxy: strip trailing assistant prefills before LiteLLM.

Also logs message roles for every /v1/messages call so we can debug Claude Code.
"""

from __future__ import annotations

import json
import os
import urllib.error
import urllib.request
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from typing import Any
from urllib.parse import urlsplit


UPSTREAM = os.environ.get("LITELLM_UPSTREAM", "http://127.0.0.1:4001").rstrip("/")
LISTEN_HOST = os.environ.get("PREFILL_PROXY_HOST", "0.0.0.0")
LISTEN_PORT = int(os.environ.get("PREFILL_PROXY_PORT", "4000"))
HOP_BY_HOP = {
    "connection",
    "keep-alive",
    "proxy-authenticate",
    "proxy-authorization",
    "te",
    "trailers",
    "transfer-encoding",
    "upgrade",
    "content-length",
    "host",
}


def _role(msg: Any) -> str | None:
    if isinstance(msg, dict):
        role = msg.get("role")
    else:
        role = getattr(msg, "role", None)
    return str(role).lower() if role is not None else None


def strip_trailing_assistant(messages: list[Any]) -> tuple[list[Any], int]:
    if not isinstance(messages, list) or not messages:
        return messages, 0
    out = list(messages)
    dropped = 0
    while out and _role(out[-1]) == "assistant":
        out.pop()
        dropped += 1
    return out, dropped


def maybe_rewrite_body(path: str, body: bytes) -> bytes:
    if not body or "/messages" not in path:
        return body
    try:
        payload = json.loads(body.decode("utf-8"))
    except (UnicodeDecodeError, json.JSONDecodeError):
        return body
    if not isinstance(payload, dict) or "messages" not in payload:
        return body
    messages = payload.get("messages") or []
    roles = [_role(m) for m in messages[-6:]]
    cleaned, dropped = strip_trailing_assistant(messages)
    print(
        f"[prefill_proxy] {path}: roles_tail={roles} dropped={dropped} "
        f"model={payload.get('model')}",
        flush=True,
    )
    if not dropped:
        return body
    payload["messages"] = cleaned
    return json.dumps(payload, separators=(",", ":")).encode("utf-8")


class PrefillProxy(BaseHTTPRequestHandler):
    protocol_version = "HTTP/1.1"

    def log_message(self, fmt: str, *args: Any) -> None:
        print("[prefill_proxy] " + (fmt % args), flush=True)

    def _proxy(self) -> None:
        parsed = urlsplit(self.path)
        path_q = parsed.path
        if parsed.query:
            path_q = f"{path_q}?{parsed.query}"

        length = int(self.headers.get("content-length", "0") or "0")
        body = self.rfile.read(length) if length > 0 else b""
        if self.command in ("POST", "PUT", "PATCH"):
            body = maybe_rewrite_body(parsed.path, body)

        headers = {
            k: v
            for k, v in self.headers.items()
            if k.lower() not in HOP_BY_HOP
        }
        req = urllib.request.Request(
            url=f"{UPSTREAM}{path_q}",
            data=body if self.command not in ("GET", "HEAD", "DELETE") else None,
            headers=headers,
            method=self.command,
        )
        try:
            with urllib.request.urlopen(req, timeout=600) as resp:
                resp_body = resp.read()
                self.send_response(resp.status)
                for k, v in resp.headers.items():
                    if k.lower() in HOP_BY_HOP:
                        continue
                    self.send_header(k, v)
                self.send_header("Content-Length", str(len(resp_body)))
                self.end_headers()
                if self.command != "HEAD":
                    self.wfile.write(resp_body)
        except urllib.error.HTTPError as exc:
            resp_body = exc.read()
            print(
                f"[prefill_proxy] upstream HTTP {exc.code} for {path_q}: "
                f"{resp_body[:300]!r}",
                flush=True,
            )
            self.send_response(exc.code)
            for k, v in exc.headers.items():
                if k.lower() in HOP_BY_HOP:
                    continue
                self.send_header(k, v)
            self.send_header("Content-Length", str(len(resp_body)))
            self.end_headers()
            self.wfile.write(resp_body)
        except Exception as exc:  # noqa: BLE001 - edge proxy
            msg = json.dumps({"error": f"prefill_proxy upstream failure: {exc}"}).encode()
            self.send_response(502)
            self.send_header("Content-Type", "application/json")
            self.send_header("Content-Length", str(len(msg)))
            self.end_headers()
            self.wfile.write(msg)

    def do_GET(self) -> None:  # noqa: N802
        self._proxy()

    def do_HEAD(self) -> None:  # noqa: N802
        self._proxy()

    def do_POST(self) -> None:  # noqa: N802
        self._proxy()

    def do_PUT(self) -> None:  # noqa: N802
        self._proxy()

    def do_PATCH(self) -> None:  # noqa: N802
        self._proxy()

    def do_DELETE(self) -> None:  # noqa: N802
        self._proxy()

    def do_OPTIONS(self) -> None:  # noqa: N802
        self._proxy()


def main() -> None:
    server = ThreadingHTTPServer((LISTEN_HOST, LISTEN_PORT), PrefillProxy)
    print(
        f"[prefill_proxy] listening on {LISTEN_HOST}:{LISTEN_PORT} → {UPSTREAM}",
        flush=True,
    )
    server.serve_forever()


if __name__ == "__main__":
    main()
