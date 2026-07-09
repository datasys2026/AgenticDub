# xAI OAuth Provider Strategy

## Goal

AgenticDub should support xAI / Grok through OAuth-only authentication, without requiring `XAI_API_KEY` or xAI metered API credits.

The intended long-term target is to use a Grok / SuperGrok / X Premium+ OAuth session for:

- LLM translation and planning
- STT transcription
- TTS synthesis

## Current Status

- Token store: implemented
- Token status CLI: implemented via `agenticdub auth xai status`
- LLM provider: implemented for xAI Responses API
- STT provider: implemented for xAI STT API
- TTS provider: implemented for xAI TTS API
- Model profiles: `models.llm.grok`, `models.stt.xai`, `models.tts.xai`
- Live entitlement probe: implemented via `agenticdub auth xai probe`
- Browser OAuth login: not implemented

## Why OAuth-Only

The project currently assumes the user has access to Grok through a subscription account rather than through xAI API billing.

This means the provider should not require:

- xAI API key creation
- API credit purchase
- `XAI_API_KEY`

Instead, AgenticDub should load a local OAuth token and send it as a bearer token when calling xAI-compatible endpoints.

## External Reality

xAI official REST API documentation uses API-key based bearer authentication as the primary supported developer path.

Hermes Agent documents a separate Grok OAuth flow for SuperGrok / X Premium+ users. That flow suggests browser OAuth can provide bearer credentials reusable for direct-to-xAI surfaces, but those surfaces may still be gated by subscription tier, account state, or backend allowlist.

Therefore, xAI OAuth support in AgenticDub must be treated as experimental until live entitlement checks pass.

## Architecture

The implementation is intentionally split by responsibility:

- `internal/auth/xai`
  - Owns token loading and storage.
  - Exposes a `BearerTokenSource`.
  - Does not know about LLM, STT, or TTS.

- `internal/providers/llm`
  - Owns the xAI OAuth LLM provider.
  - Calls `/v1/responses`.
  - Does not know where the token came from.

- `internal/providers/stt`
  - Owns the xAI OAuth STT provider.
  - Calls `/v1/stt`.
  - Converts word timings into AgenticDub transcription data.

- `internal/providers/tts`
  - Owns the xAI OAuth TTS provider.
  - Calls `/v1/tts`.
  - Writes the returned audio stream to the requested output file.

- `internal/service`
  - Wires `llm.provider = "xai-oauth"` to the xAI OAuth LLM provider.
  - Wires `transcribe.provider = "xai-oauth"` to the xAI OAuth STT provider.
  - Wires `tts.provider = "xai-oauth"` to the xAI OAuth TTS provider.

- `cmd/cli`
  - Exposes OAuth diagnostics such as `agenticdub auth xai status`.
  - Exposes a live entitlement smoke test via `agenticdub auth xai probe`.

This keeps the provider open for future STT/TTS extensions without mixing audio behavior into the LLM provider.

## Configuration

Example:

```toml
[xai_oauth]
base_url = "https://api.x.ai/v1"
token_path = "~/.agenticdub/auth/xai.json"

[models.llm.grok]
provider = "xai-oauth"
base_url = "https://api.x.ai/v1"
model = "grok-4.20-0309-non-reasoning"

[models.stt.xai]
provider = "xai-oauth"
base_url = "https://api.x.ai/v1"
model = "xai-stt"

[models.tts.xai]
provider = "xai-oauth"
base_url = "https://api.x.ai/v1"
model = "xai-tts"
voices = ["eve", "ara", "rex", "sal", "leo"]
```

Live OAuth testing on this machine showed `grok-4.20-0309-non-reasoning`
returns stable JSON for short translation prompts. `grok-4.3` entitlement
checks pass, but translation prompts can return malformed JSON through the
OAuth Responses path, so it is not the current AgenticDub translation default.

To reuse this machine's Hermes Agent OAuth login, point `token_path` at Hermes:

```toml
[xai_oauth]
base_url = "https://api.x.ai/v1"
token_path = "~/.hermes/auth.json"
```

The CLI status command also accepts `XAI_OAUTH_TOKEN_PATH`:

```bash
XAI_OAUTH_TOKEN_PATH=~/.hermes/auth.json agenticdub auth xai status
```

## Token Formats

AgenticDub supports its own token JSON:

```json
{
  "access_token": "...",
  "refresh_token": "...",
  "id_token": "...",
  "token_type": "Bearer",
  "expires_at": "2026-06-10T12:00:00Z"
}
```

It also supports legacy Hermes-style direct provider tokens:

```json
{
  "active_provider": "xai-oauth",
  "providers": {
    "xai-oauth": {
      "access_token": "...",
      "refresh_token": "...",
      "id_token": "...",
      "token_type": "Bearer"
    }
  }
}
```

For current Hermes credential pool files, AgenticDub reads `credential_pool["xai-oauth"]`
first and chooses the first credential with `last_status == "ok"` and an
`access_token`. Exhausted entries are skipped. If no usable pool credential exists,
AgenticDub falls back to `providers["xai-oauth"].tokens`.

```json
{
  "active_provider": "xai-oauth",
  "credential_pool": {
    "xai-oauth": [
      {
        "access_token": "...",
        "refresh_token": "...",
        "last_status": "ok",
        "token_type": "Bearer"
      }
    ]
  },
  "providers": {
    "xai-oauth": {
      "tokens": {
        "access_token": "...",
        "refresh_token": "...",
        "token_type": "Bearer"
      }
    }
  }
}
```

## Testing Strategy

Unit and integration-style tests should remain network-free by default.

Current tests cover:

- token save/load
- missing token detection
- expired token detection
- Hermes direct provider token parsing
- Hermes credential pool parsing, preferring `last_status == "ok"`
- xAI `/v1/responses` request shape
- OAuth bearer authorization header
- response parsing
- `403` entitlement diagnostics
- STT/TTS OAuth request shape
- service factory wiring for `xai-oauth` LLM/STT/TTS

Live tests must be explicit manual commands because they depend on account entitlement.

## Manual Smoke Test Plan

1. Verify token is readable:

```bash
agenticdub auth xai status --token-path ~/.agenticdub/auth/xai.json
```

2. If using Hermes OAuth output:

```bash
agenticdub auth xai status --token-path ~/.hermes/auth.json
```

3. Run a live Responses API probe:

```bash
agenticdub auth xai probe --token-path ~/.hermes/auth.json --model grok-4.20-0309-non-reasoning
```

The probe sends a minimal prompt to `/v1/responses` and surfaces failures:

- missing token
- expired token
- network error
- `401` invalid token
- `403` subscription / entitlement blocked
- empty model response

4. Run live STT/TTS endpoint checks with the same OAuth token before enabling a long pipeline job.

On this machine, Hermes OAuth token smoke tests passed against `/v1/tts` with voice
`eve` and `/v1/stt` with word timings enabled.

## Risks

- xAI official REST documentation does not present OAuth as the primary API developer path.
- OAuth behavior may change without the same compatibility guarantees as API keys.
- A token may be valid for login but not authorized for inference.
- STT/TTS may have different entitlement requirements from LLM.
- TTS built-in voices may not match the current Qwen custom voice workflow.
- Browser OAuth in headless or remote environments may require loopback tunneling.

## Implementation Roadmap

1. Token store and token status command.
2. xAI OAuth LLM provider.
3. Manual live probe for Grok 4.3.
4. xAI STT provider.
5. xAI TTS provider.
6. Add `grok` and `xai` profiles to MCP and CLI workflows.
7. Browser OAuth login flow.
8. Decide whether OAuth support is stable enough to become the default.

## Non-Goals For Now

- Replacing all providers with xAI in one step.
- Making OAuth the default before live entitlement tests pass.
- Supporting xAI custom voices before account and region constraints are understood.
- Removing existing aiark local endpoints.
