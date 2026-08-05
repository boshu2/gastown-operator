# WORKFLOW 4 — THE RENDER · v1 (three stations: one writes, one repairs, one restates; the operator judges)

**Entry gate:** the LOCKED briefs file on disk with all OPEN items ruled · `dependencies/writer_constitution.md` current · `dependencies/lineedit_codex.md` **control-calibrated** (near-zero edits on Ebony + Draft Selection — an editor that "fixes" a control is miscalibrated; tune before any live use).

**Output:** `working files/w4 scripts/[Series]_[Angle]_v[X].md` + the per-scene log. **No promo ships without the operator's read — the read is the only taste gate.**

**THE DELIVERY RULE (hard — installed after two consecutive runs skipped the stations).** The unit of delivery is **ONE SCENE + its receipt + its edit log.** A stitched multi-scene draft that did not pass through per-scene operator gates, the line editor, and the cold reader is a **VOIDED RUN** — not a draft to give notes on; it is discarded and the run restarts at the last gated scene. A scene delivered without its edit log is undelivered (the log is how a skipped editor becomes visible). **Every artifact carries a version stamp in its header: `built on promo_tool_v4.N`** — a run that can't name its package version can't claim its rules ran.

---

## STATION 1 — THE WRITER (the session protocol)

**One writing session renders the whole promo, one scene at a time.** The session is a CLEAN ROOM: a fresh conversation with **no folder access, no pipeline files, no prior versions** — contamination voids the artifact.

- **The loadout, total:** the session preamble + `writer_constitution.md` (pasted whole, voice model included) + the CURRENT scene's brief. Nothing else. Prior scenes of this promo accumulate naturally in the session — that is the continuity mechanism; nothing crosses in from outside.
- **The preamble (fixed):** you are writing a full promo one scene at a time; write ONLY the current brief's scene, whole, in one go; read it aloud once at talking speed against the voice model; then STOP and wait. Keep perfect continuity with what you've already written. Never resolve the story.
- **Between scenes:** the operator's gate (below) runs; then the next brief pastes into the SAME session. Operator notes paste in as a numbered packet before the next brief — operator-authored replacement lines are used VERBATIM. Point-fixes end with "reprint the corrected scene in full."
- The writer call runs at maximum capability; every other station runs cheap.

## STATION 2 — THE LINE EDITOR (the mechanized notes)

A separate, fresh-context agent receives ONLY the draft scene + `lineedit_codex.md`, and does one thing: walk the codex classes against the draft, fix violations in place, and log every edit as `[class] before → after`.

- It never rewrites voice, never restructures, never judges quality. Where a codex class quotes an operator model, it matches register, never content.
- A sentence that seems wrong but fits no class is FLAGGED, not touched — flags route to the operator; a ruled flag becomes the codex's next class.
- Output: the edited scene + the edit log. Target state: zero edits (the writer has internalized the codex); calibration state: zero edits on the two controls, always.

## STATION 3 — THE COLD READER

A zero-context agent reads the edited scene once (plus, mid-promo, a two-line "previously" assembled from the briefs' prerequisite facts) and restates it: who acted, what changed, what each named thing is. Misparses and untaught references flag with the quoted line. Comprehension only — it never grades feeling; a "felt nothing" is information for the operator, never a verdict.

## THE PER-SCENE RECEIPT (arithmetic only — prints with every scene)

**Two disciplines govern it. ONE ALTITUDE, ONE PASS:** every check has exactly one home and runs exactly once; later stages read the recorded result, never re-run it. **REPORT EXCEPTIONS, NOT CONFIRMATIONS:** a clean check emits one line — the receipt never documents its own innocence; that is the named source of receipt bloat.

`CONTINUITY: the scene's opening matches the map's IN (time, place, CARRY items touched) and its close lands the OUT + seam voice — checked against the prior scene's actual last lines · length vs budget — WITH the density rule: a scene >20% over its event-derived budget is presumed padded (route: cut description, never events); a scene short is a BRIEF defect routed upstream, never filled · EXCHANGE RECONCILIATION (each mapped exchange: floor turns → rendered turns → closes on the hero? — map by map) · dialogue share + narration-block census (blocks >2 sentences, listed) — printed diagnostics, never gates · XFERs verbatim (string-checked) + triggers honored · numbers sourced (every figure names its brief/world-doc source; unsourced = cut) · plants/re-arms due this scene, quoted · edit log (count by class) · restatement result · [after operator read] notes count + bin split (repeat / new / taste)`

## THE OPERATOR GATE (between every scene)

The operator reads the edited, checked scene with its receipt. Notes triage into the three bins (CLAUDE.md governance): **REPEAT CLASS** → strengthen that codex class's exemplars — an editor miss, fix the instrument · **NEW CLASS** → one line into constitution or codex + the operator's pair; conserve mass (name what it merges into) · **PURE TASTE** → apply to the scene, deposit the operator's lines to `exemplar_bank.md` by slot, add NO law. Then the fixes packet (Station 1 protocol) or, if clean, the next brief.

**Health metrics, logged per scene:** notes count · repeat-class share. Converging = down and toward zero. A rising repeat share indicts the editor; a rising new-class rate on a mature register indicts the constitution — both are instrument problems, never grounds for a pipeline restructure without a conservation table.

## ROUTING (pre-decided)

Mechanical miss (length, XFER breach, plant unpaid) → fix in place, same session. Comprehension fail → plainer line, same session. **Structural symptom** (a beat that can't run hot, a concept that won't teach, a passive-hero unit) → a MAP defect: route to W2 with a diff; W4 never relitigates structure. Register drift across a whole scene → re-paste the constitution into the session and regenerate the scene once; a second drift in the same session = end the session, start fresh from the last approved scene.

## LOCK + DEPOSIT

All scenes approved → assemble the full script, run the receipt on the whole (length floor 3,500 hard · every XFER · every plant · the cliff resolves nothing), save, and deposit the operator's authored lines from this run into `exemplar_bank.md` and any keep-lines to the ledger as CANDIDATE (quarantined — unshipped material anchors nothing, per the Harvest Provenance Law).

> **Script locked, pending the operator's independent read of the assembled whole.** A passed gate is a measurement, never a verdict.

## FINISH (automated runs)

When the script and receipt are saved, run `promo-finish finish` per `scripts/PROMO_FINISH.md` — one command, key=path for map, briefs, script, receipt. Stop on `PROMO_FINISH_OK`.
