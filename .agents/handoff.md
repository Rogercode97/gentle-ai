# Session Handoff — Antigravity to Pi Continuity

- **Date**: 2026-09-12T20:43:00Z
- **Source Agent**: Google Antigravity CLI (`agy`)
- **Target Agent**: `gentle-pi` (`pi`) / CLI
- **Workspace**: `/data/data/com.termux/files/home/gentle-ai-termux`
- **Engram Reference**: Observation `#4578` (summary) and `#4019` (`session/bootstrap/gentle-ai`)

---

## 🎯 Objetivo Inmediato
1. **Actualizar `FileSubAgents` a `false` en Antigravity**:
   - Archivo: `internal/agents/capabilitymanifest/manifest.go` (alrededor de línea 294).
   - Cambiar `model.AgentAntigravity: { FileSubAgents: true, ... }` a `FileSubAgents: false`.
2. **Vaciar directorios de subagentes en Antigravity Adapter**:
   - Archivo: `internal/agents/antigravity/adapter.go`.
   - `SubAgentsDir(_ string)` → retornar `""`.
   - `EmbeddedSubAgentsDir()` → retornar `""`.
3. **Corregir schema en `hooks.json` a nivel proyecto**:
   - Archivo: `/data/data/com.termux/files/home/skills-sovereign/.agents/hooks.json`.
   - Reemplazar el array huérfano `{ "hooks": [ ... ] }` por un hook nombrado que `agy` pueda unmarshalizar:
     ```json
     {
       "sovereign-guardrails": {
         "PreToolUse": [
           {
             "matcher": "run_command",
             "hooks": [
               {
                 "type": "command",
                 "command": "./scripts/hooks/safety-guard.sh",
                 "timeout": 30
               }
             ]
           }
         ]
       }
     }
     ```
4. **Verificación**:
   - Correr `go test ./internal/agents/...` en `gentle-ai-termux`.

---

## ⚠️ Gotchas Técnicos Verificados
- En Termux, cualquier script `.sh` invocado por hooks en `hooks.json` debe tener shebang `$PREFIX/bin/bash` o ser invocado con `bash <script>` para evitar `kernel exit status 127: not found`.
- Antigravity no carga hooks de `plugins/`, solo `~/.gemini/config/hooks.json` y `.agents/hooks.json`.
