# PACKAGE CHANGELOG — promo_tool_v4
*(One entry per versioned zip. Rule: every change to the package ships as a NEW `promo_tool_v4.N.zip` — never overwrite a shipped zip; the changelog line is written in the same pass as the zip. "What changed?" is answered by diffing two zips, never by memory.)*

## v4.5 — 2026-08-05
*(Finish handoff — run promo-finish when W4 is done.)*
- **`scripts/PROMO_FINISH.md` (new)** — key → path table + one `promo-finish finish` command. No orchestrator/env details.
- **`CLAUDE.md`**, **`workflow4_render.md`**, **`HANDOFF.md`** — one-line finish pointer.

## v4.4 — 2026-07-29
*(Layout release — operator ruling: the package installs at the export ROOT, not a separate folder, so an agent starting at the root loads v4, never the retired v3 loadout.)*
- **Installed at root:** v4 `CLAUDE.md` replaces the old v3 root CLAUDE.md · v4 workflows 2/3/4 join W1/W5 in `workflows/` · v4 dependencies (constitution, brief template, codex, exemplar bank, both controls) join `dependencies/` · `HANDOFF.md`, `CHANGELOG.md`, `CONSERVATION_Rebuild_v1.md` at root.
- **Retired to `_to_delete/`:** the v3 root `CLAUDE.md` · `workflow2_skeleton.md` · `workflow3_beatsheet.md` · `workflow4_script.md` · stale `Ledger_RBM_v2.md` · the superseded `promo_tool_v4/` subfolder (zips are the archive of record).
- **Ledgers made current at root:** `Ledger_RBM_v4.md`, `Ledger_MVS_v1.md`, `Ledger_RotHC_v1.md` copied in from the v3 export (PH_v2, WBT_v1 already current). Root `dependencies/microbeat_taxonomy.md` (W2 §4b's reference) confirmed present — housekeeping item closed.
- `CLAUDE.md` — LAYOUT note added; W1/W5 routing rows no longer point at the v3 export. `HANDOFF.md` §2–3 rewritten for the root install. Zip now mirrors the root layout.

## v4.3 — 2026-07-29
*(Source: the layered Disowned review — W2 map verified healthy; the fat-narration script traced to the W3 layer. Measured: 130 narrator blocks vs 61 voiced lines (~83% narration), longest desert 12 consecutive blocks / 310 words; total exactly on budget — the briefs specified the mass, the writer obeyed.)*
- `brief_template.md` **v4** — two derivation laws (operator-ruled): **law 7 EXCHANGES ARE THE COUNTED UNIT** ("narration beat" banned as a counted unit; narration is mortar, never counted mass; a derivation that can't name its exchanges routes to W2 — absorbs the old turns-plus-narration-beats arithmetic) · **law 8 XFERS ARE RUNGS, NOT THE EXCHANGE** (turn floors count rendered turns; the writer voices the turns around and between XFERs; an all-XFER floor stages the documented v3 transfer-and-narrate anti-pattern).
- `workflow3_briefs.md` **v2** — §1 budget derivation rewritten exchanges-first per law 7; self-gate ban list gains the two enforcement items (narration-beat units · all-XFER floors).
- No W4 change: A16 (voice desert) + the DELIVERY RULE already cover the symptom downstream; they need to run, not grow.

## v4.2 — 2026-07-29
- `lineedit_codex.md` — four classes from the Disowned run: **A14 because-weld** (gloss welded into event sentences via ", because/so/and" chains) · **A15 listener conferencing** (narrator direct address banned outside the CTA) · **A16 voice desert** (restored: ≤150 spoken words voiceless while a licensed speaker is on scene) · **B19 re-arm re-teach** (a re-arm is one clause, never a re-explained reveal).
- `workflow4_render.md` — **THE DELIVERY RULE (hard):** one scene + receipt + edit log is the only deliverable unit; stitched ungated drafts are VOIDED RUNS; artifacts stamp `built on promo_tool_v4.N`.
- `HANDOFF.md` — post-mortem #2: second consecutive run skipped the stations; ~80% of operator notes on both runs were already-written law unenforced.

## v4.1 — 2026-07-29
- `workflow2_map.md` **v2 — major:** full merge restoring the W2 construction brain after file-level audit (intake lock · Spine + Relatable Thruline · WHY NOW · injustice rotation/trajectory/density · 6-question loop + furniture test · ANTI-SETUP GATE · Beat-1 anchor protocol · full inheritance discipline · cliff withholding ratchet + fidelity + BOFU HARD STOP) + set-piece realization (DEPTH/SCOPE, CANON-FABRICATED) + engine matrix + backward plant pass + IN→OUT/TIME/CARRY/BECAUSE chain + words-follow-events.
- `workflow3_briefs.md` — powerstart fact-check vs Scene 1 · canon-banter OPEN class · bottom-up budget derivation.
- `workflow4_render.md` — receipt: exchange reconciliation · continuity row · density rule · sourced-numbers · dialogue-share diagnostic; one-altitude-one-pass + report-exceptions-only.
- `writer_constitution.md` — additive only, validated core byte-identical: budget-as-forecast · THE LOOP posture · THE PRODUCTION BLOCK (SFX/tags/AIVO/CTA + verbatim download line).
- `lineedit_codex.md` — A10 oversize blocks · A11 pre-announcement · A12 gloss family · A13 reporting mode · B13 costume test · B16 rationalized passivity · B17 ledger close · B18 outcome-promising wink.
- `brief_template.md` — continuity contract (IN/CARRY/BECAUSE open the events; OUT + seam close them).
- `CONSERVATION_Rebuild_v1.md` — all three workflow rows rewritten with file-level audit dispositions and evidence citations.
- `HANDOFF.md` — new since v4.0; carries the first-run post-mortem and the Midnight version pin.
- Unchanged from v4.0: `CLAUDE.md` · `exemplar_bank.md` · both control scripts · `powerstart.md`.

## v4.0 — 2026-07-29 (first compile; zip overwritten before versioning began — deltas reconstructed above from the session edit log)
- Initial package: CLAUDE.md · workflows 2/3/4 · writer_constitution · brief_template · lineedit_codex · exemplar_bank · Ebony + Draft Selection controls · powerstart · conservation table.
