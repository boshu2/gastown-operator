"""Strip trailing assistant prefills before MadEye rejects them.

MadEye (Vertex-backed claude-opus-4-8) requires conversations to end with a
user message. Claude Code often sends a trailing assistant turn (prefill).
This LiteLLM proxy hook drops those trailing assistant messages.
"""

from __future__ import annotations

from typing import Any, Literal, Optional

from litellm.integrations.custom_logger import CustomLogger

try:
    from litellm.proxy.proxy_server import DualCache, UserAPIKeyAuth
except ImportError:  # pragma: no cover - version drift
    from litellm.caching.caching import DualCache
    from litellm.proxy._types import UserAPIKeyAuth


def _role(msg: Any) -> str | None:
    if isinstance(msg, dict):
        role = msg.get("role")
    else:
        role = getattr(msg, "role", None)
    return str(role).lower() if role is not None else None


def _strip_trailing_assistant(messages: list[Any]) -> list[Any]:
    if not isinstance(messages, list) or not messages:
        return messages
    out = list(messages)
    while out and _role(out[-1]) == "assistant":
        out.pop()
    return out


class StripPrefillHandler(CustomLogger):
    async def async_pre_call_hook(
        self,
        user_api_key_dict: UserAPIKeyAuth,
        cache: DualCache,
        data: dict,
        call_type: Literal[
            "completion",
            "text_completion",
            "embeddings",
            "image_generation",
            "moderation",
            "audio_transcription",
        ],
    ) -> Optional[dict]:
        messages = data.get("messages")
        if not messages:
            # Always log trailing roles so we can debug Claude Code shapes.
            print("[strip_prefill] no messages in request data", flush=True)
            return data
        roles = [_role(m) for m in messages[-4:]]
        cleaned = _strip_trailing_assistant(messages)
        if len(cleaned) != len(messages):
            print(
                f"[strip_prefill] dropped {len(messages) - len(cleaned)} "
                f"trailing assistant message(s); last_roles={roles}",
                flush=True,
            )
            data["messages"] = cleaned
        elif roles and roles[-1] == "assistant":
            print(
                f"[strip_prefill] WARNING still ends with assistant after strip? "
                f"last_roles={roles}",
                flush=True,
            )
        return data


proxy_handler_instance = StripPrefillHandler()
