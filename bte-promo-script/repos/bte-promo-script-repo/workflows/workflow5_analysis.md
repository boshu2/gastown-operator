# WORKFLOW 5: ANALYSIS

**Purpose:** Judge finished and in-progress work — skeletons, beat sheets, and scripts — against the two keys that decide whether a promo is worth shipping. Diagnosis only; no rewriting unless the operator asks to shift back into W2/W3/W4.

W5 is **not** a beat-polishing pass with an eye toward W4. Its first job is the verdict: will this work for us? W4-improvement suggestions are a *secondary* output, made only after the two keys have been answered — never a substitute for the verdict.

---

## THE TWO KEYS (the core objective — applied to EVERY artifact)

Every W5 analysis — of a skeleton, a set of beats, or a finished script — answers the same two questions, in order. They apply identically across all three sub-functions. If an analysis hasn't answered both, it isn't finished.

### KEY 1 — CPI: will this obligate an install?

Is this a strong enough piece of work that a cold viewer paying for the impression is *obligated* to install? This is the funnel-top question: opening, cliffhanger, writing, scene craft, dialogue. For a **script**, Key 1 is scored by the Quality Gate in `bucketing.md` (bucket ≥ 3 = PASS). For a **beat sheet or skeleton**, Key 1 is the same judgment applied to structure: does this beat-order, executed competently, obligate an install — or is the opening soft, the cliffhanger leaky, the spine inert?

### KEY 2 — BOFU: will this convert and hold a paying user?

Will a cold install become a paying user who finds the show they were promised? Key 2 is a **scorecard of four checks, ranked by importance.** Check #1 is a hard gate; #2–#4 are weighted and a severe failure on any can sink BOFU.

1. **True to the show (MOST IMPORTANT — hard gate).** Is it recognizably this show — world, cast, mechanics, Essence, AND tone? A tone or premise the show doesn't support (e.g. a torture/death-collar stakes premise on a show that isn't that dark) FAILS here even with flawless craft. For scripts this is the Show Fidelity Gate in `bucketing.md`. **A Key 2 #1 failure fails the artifact** — name it and give the reasons.
2. **Extreme empathy, especially right before the cliffhanger (second most important).** Has the audience earned the right to care by the climax, and is empathy *peaking* in the run-up to the cut? Injustice density is the engine of this. Empathy that depletes after the first third, or runs cold into the cliffhanger, is a BOFU failure.
3. **A unifying theme / thruline that aligns with the show.** Is there one relatable thruline — "everyone has been underestimated," "everyone has had something to hide" — and does it match the show the listener actually gets at BOFU? A thruline that promises a different emotional contract than the show delivers is a BOFU failure.
4. **A discrete, must-resolve cliffhanger — not a vague world-expansion.** Are we selling a specific conclusion a listener will be *desperate* not to miss (the killer is revealed, a one-on-one match is called) — or something diffuse and "bigger" that merely expands the world? Discrete, must-resolve hooks convert harder at BOFU. A world-expansion-only cliffhanger is a flag.

### Verdict

An artifact is worth advancing only if **Key 1 PASSES and Key 2 PASSES** (with #1 being a hard gate). Either key failing = a FAIL with explicit, actionable reasons. State the verdict first; the diagnostic detail and any W4-improvement notes follow it.

---

W5 also feeds the **W4↔W5 automation loop.** Every analysis produces dual-channel output: operator-facing markdown AND a structured findings block W4 can parse for revision. This is a downstream convenience, not the objective — the two keys come first.

---

## INPUTS — TIERED LOAD MODEL

W5 prioritizes speed-to-diagnosis. Do not bulk-load all reference files at start.

### Load at start (minimal)

- **The artifact being analyzed** — beat sheet or script (already in conversation).
- **`microbeat_taxonomy.md`** — kill flags + anti-pattern flags + tag/subtype reference. Used in every sub-function.
- **`voice_style.md`** — §6 LLM failure modes + §7 diagnostic tests. Used in every sub-function.

### Load on demand

- **`bucketing.md`** — only when running Bucketing (Sub-function 2). Standalone two-gate framework (Quality Gate + Show Fidelity Gate); do not preload.
- **`world_[show].md`** — Essence block only, pulled when fidelity checks fire or when running the Show Fidelity Gate (Key 2 #1).
- **`three_pillars.md`** — pulled when assessing Key 2 #3 (thruline alignment with the show).
- **`hook_engine_library.md`** — only when verifying engine contract during Deep Dive.
- **`functional_skeletons.md`** — only when verifying skeleton inheritance during Deep Dive.

### Do not load

- W1/W2/W3/W4 spec files (W5 operates on outputs, not specs).
- Example files.
- Any reference file not listed above.

---

## ROUTING

| Input | Default sub-function | Deep Dive trigger |
|---|---|---|
| Beat sheet / skeleton | **Beat Analysis** (Sub-function 1) | On request — "go deep" or "full audit" |
| Script | **Bucketing** (Sub-function 2) | After both gates pass (Bucket ≥ 4 AND Fidelity PASS) OR on request |

If the operator says "analyze," "what's wrong," "notes" → default to the routing above. If they say "deep dive" or "full audit" → run Sub-function 3 directly. If they say "bucket this" → Sub-function 2.

---

## POSTURE

This is the only workflow where full diagnostic density is correct. You are evaluating work, not building.

**The two keys override all other lenses.** Every diagnostic below is in service of answering them:

1. **Key 1 — CPI.** Will this obligate an install? (Funnel-top craft + structure.)
2. **Key 2 — BOFU.** Will it convert and hold a paying user? Its four ranked checks — true-to-show (incl. tone), empathy peaking before the cliffhanger, an aligned thruline, and a discrete must-resolve cliffhanger — are the through-line of every sub-function. Injustice density is not a separate lens; it is the *engine* of Key 2 check #2, and depleting injustice is flagged as a BOFU failure.

**Be critical.** You are seeing this for the first time. You did not build it. Find the weaknesses. State the verdict before the polish.

**Cold read, every time.** Treat every analysis — beats or script, first pass or fifth — as an independent first read. Never diff against a prior pass or check only whether the last note was addressed. Re-derive the verdict from scratch. The writer's incentive is to revise until the tool agrees; the defense is that approval is never inherited. A piece that would fail on a clean read in a new conversation must fail here, regardless of how many revisions preceded it. If a fresh read lands lower than last time, say so and explain why. (Full statement in Sub-function 1 → Cold-Read Mandate; it applies equally to re-bucketing in Sub-function 2.)

**Token discipline.** Default outputs are compressed. Deep dive only on request or when the script passes both gates.

---

## OUTPUT MODEL — DUAL CHANNEL

Every W5 analysis produces two outputs:

### Channel 1 — Operator-Facing Markdown

Compact diagnostic for human review. Format varies by sub-function (see below). Default posture: tight, actionable, no filler.

### Channel 2 — Structured Findings Block

Machine-readable table W4 can parse for the automation loop. Same data as the markdown, normalized into rows. Always appended at the end of the analysis output.

Every analysis doc ends by updating the show/angle findings ledger (append its findings, mark any it retires) — so the ledger W3 loads is a maintained artifact, not a scavenger hunt.

**Format:**

```
## STRUCTURED FINDINGS

| ID | Severity | Location | Category | Issue | Proposed Fix |
|---|---|---|---|---|---|
| F-001 | CRITICAL | Beat 7 | taxonomy:kill:STATIC-BEAT | Beat has no nameable movement; state description only | Rewrite with internal change OR merge into Beat 8 |
| F-002 | HIGH | Scene 2 MB5 | craft:dialogue-thinness | Dialogue volume 45%, target 60%+ for Single-Cast | Generate crowd reactions + antagonist escalation per W4 §Dialogue Generation |
| F-003 | MED | Beat 12 | structural:im-desert | 3 consecutive beats without IM | Layer T1 IM at Beat 11 |
```

**Severity tiers:**

| Tier | Meaning | W4 loop behavior |
|---|---|---|
| **CRITICAL** | Kill flag, HARD BLOCK, Bucket 1-2 trigger, structural break that prevents shipping | W4 must revise; cannot auto-advance. Surfaces immediately. |
| **HIGH** | Significant execution or structural failure requiring revision before lock | W4 auto-revises once; if still failing, surfaces to operator |
| **MED** | Improvement opportunity, polish-level for Bucket 4-5, structural pattern worth fixing | W4 surfaces to operator with proposed fix; operator decides |
| **LOW** | Observation; no action required unless operator chooses | Appended to polish notes; not surfaced |

**Categories** (controlled vocabulary — extend only when a new category is required):

- `taxonomy:kill:[FLAG_NAME]` — kill flag from `microbeat_taxonomy.md` Section 8
- `taxonomy:anti-pattern:[FLAG_NAME]` — anti-pattern from Section 9
- `craft:[principle]` — failure against `voice_style.md` §1 craft principle (specificity / cruelty / compression / witness / gesture / system / cliff / dialogue-thinness / staging-violation)
- `llm-failure:[mode]` — `voice_style.md` §6 LLM failure mode (abstract-lore / trauma-summary / flash-forward / adjective-stacking / cartoon-bully / resolution-leak / explain-after-show / repeated-humiliation / confusion-twist / drop-in-savior / texture-over-clarity / genre-muddy)
- `structural:[issue]` — structural pattern (im-desert / im-density / injustice-distribution / because-chain / cliffhanger / hook-engine / escalation / parallel-threads)
- `quality:[dimension]` — Quality Gate (Key 1) dimension failure (opening / cliffhanger / writing-style-voice / scene-quality / dialogue-ratio)
- `fidelity:[surface]` — Show Fidelity Gate (Key 2 #1) failure (surface / essence / tone-fit). A `fidelity:tone-fit` finding is always CRITICAL.
- `bofu:[check]` — Key 2 #2–#4 failure (empathy / thruline / cliffhanger-type)
- `essence:[dimension]` — show essence drift, scored inside the Fidelity Gate (identity / combat / power-source / voice-register)

**Location format:** `Beat [N]`, `Scene [N] MB[N]`, `Scene [N]`, `Whole script`, `Powerstart`, `CTA`, or `Lines [N]–[N]` for specific passages.

---

## SUB-FUNCTION 1: BEAT ANALYSIS

**Input:** W3-locked beat sheet (scenes containing tagged micro-beats with treatments + annotations), or a skeleton.
**When to use:** Default for beat sheet / skeleton analysis requests.
**Purpose:** Answer the two keys for the structure *before* W4 writes prose — and only then catch the story problems worth fixing at the cheap iteration stage.

### COLD-READ MANDATE (non-negotiable)

**Every analysis is a fresh, first-time evaluation — even if you have analyzed this exact beat sheet before in this conversation.** Do not diff against a prior pass. Do not check "did the thing I flagged get fixed?" and call the rest unchanged. Re-derive the verdict from scratch, as if you have never seen these beats and have no memory of an earlier score.

Why this is a hard rule: the writer's incentive is to revise until the tool agrees. If each pass only checks whether the last note was addressed, the score ratchets upward across iterations regardless of whether the beats actually got better — the operator games the analysis without meaning to. The defense is independence. A beat sheet that would FAIL on a clean first read in a new conversation must FAIL here, no matter how many times it has been revised or how close the last pass came to passing.

Operationally:
- Score and label every beat from the artifact in front of you, not from the delta since last time.
- If a revision fixed Beat 7 but quietly weakened Beat 12, the cold read catches Beat 12 — a diff-read would miss it.
- Never let "this is close to what we approved before" substitute for a real re-read. Approval is not inherited.
- If a fresh read lands lower than a previous pass, say so plainly and explain why. The operator needs to know the honest standing, not a number that only goes up.

State the basis explicitly in the output: this is a cold read scored on its own merits.

**For high-stakes re-checks, recommend a fresh conversation.** A prior analysis sitting in the same context can still bias the re-read; the only way to guarantee a true cold read is to re-run it in a new conversation. When the operator is re-checking something that has already been analyzed here — especially after several revisions — add one line: *"For a fully independent read, re-run this in a new conversation."*

### The two keys for a beat sheet

Beat Analysis answers the same two keys as Bucketing, applied to structure rather than execution:

- **Key 1 (CPI):** would this beat-order, competently executed, obligate an install? Opening pull, cliffhanger hunger, propulsive spine. The Phase A dimensions below are how Key 1 is scored for a beat sheet.
- **Key 2 (BOFU):** #1 is it true to the show (world, Essence, **tone-fit** — a beat sheet can bake in a tone the show doesn't support); #2 does empathy build and *peak before the cliffhanger* (injustice density + witness beats); #3 is there one aligned thruline; #4 is the cliffhanger discrete and must-resolve, not a vague world-expansion. Lead the operator-facing output with the two-key verdict, then the dimension detail.

### Priority Order — STORY FIRST, WORKFLOW SECOND

Beat Analysis is fundamentally **story analysis**. Will this beat sheet tell a story that converts? Phase A runs first and gets primary surface area in the operator-facing output. Workflow compliance (taxonomy kill flags, missing annotations, spec adherence) is real but secondary — it lives in Phase B and surfaces as polish notes after the story diagnostic.

A beat sheet with broken story logic and perfect spec compliance is a failure. A beat sheet with strong story logic and a missing `Movement:` annotation is a polish job. The analysis must reflect that priority.

### Phase A — STORY ANALYSIS (PRIMARY)

The lens here is the listener, not the spec. Does this beat sheet move a cold listener from scroll to install? Each dimension scored PASS / FLAG / FAIL with one-line evidence (beat refs required).

- **Opening pull.** Does Beat 1 make the listener CARE in 3 seconds? Does it open inside the rupture — concrete failing attempt, rupture landing on POV, or contradiction the protagonist is already inside? A clean SETUP that takes 4 beats before the rupture arrives → FLAG.
- **Protagonist agency.** Across the beat sheet, is the protagonist driving the story or being moved by it? Active choices with visible cost = PASS. Receiving information / being acted upon throughout = FAIL.
- **Antagonist effectiveness.** Is the cruelty specific (named objects/acts/words, not labels), procedural (calm common sense, not theatrical), and escalating across the arc? Multiple antagonist sources with differentiated cruelty modes = PASS. Single antagonist delivering the same kind of harm repeatedly = FLAG.
- **Empathy investment.** Has the audience earned the right to care by the climax? Are there witness/tenderness beats inside the darkness? Beat sheet that is pure injustice with zero warmth = FLAG (the listener fatigues; CTI suffers).
- **BECAUSE chain causality.** Read every scene-to-scene link cold. Does each link connect to the PRIMARY action of the previous scene, or is it just narrating sequence? Broken or weak links = CRITICAL (the spine doesn't hold).
- **Injustice density and variety.** Map every micro-beat: injustice or not. Where are the deserts? Is the variety rotating (Social / Material / Relational / Moral / Existential / Institutional / Physical)? Does each tagged INJUSTICE STRIKE pass the threshold test (knowledge gap OR virtue punished — friction ≠ injustice)? Density depleting after first third → FAIL.
- **IM map.** Every IM beat. Are they sticky (one-sentence describable), fun (exciting not just grim), distinct (different from each other)? Deserts (3+ beats without)? Tonal variety (at least 2 registers)?
- **Stakes coherence.** Are stakes felt or labeled? Specific consequences and named costs = PASS. Generic threat language ("everything is at stake") = FLAG.
- **Set-piece pacing.** Highest-stakes moments getting room to breathe? Or are High set-piece beats compressed alongside Low set-pieces in the same scene?
- **Resolution withholding.** Inhale or exhale at the final beat? Becoming, or having become? More questions at the cliffhanger than at the start of the final third? Same emotional currency as the opening? Resolution leaked → CRITICAL.
- **Cliffhanger hunger.** Does the final beat make a cold listener NEED to download? One unresolved question that compresses the engine = PASS. Multiple closed loops with one minor thread left = FAIL.

### Phase B — WORKFLOW & COMPLIANCE (SECONDARY)

Spec adherence check. W3 already ran kill-flag, review-flag, and anti-pattern scans at lock. W5 re-runs defensively. Findings here surface as polish notes and improvement opportunities — NOT as the primary diagnostic.

**Severity calibration for workflow findings:**

- A workflow violation that is **also breaking the story** (e.g., STATIC-BEAT at a load-bearing moment producing a dead spot in the listener's experience) → escalate to CRITICAL/HIGH and ALSO appears in Phase A as a story finding.
- A workflow violation that is **pure spec compliance** with no story impact (e.g., missing `Punishment type:` annotation on an INJUSTICE STRIKE beat where the cruelty modality is clear from context) → MED or LOW. Polish note for W4 readiness, not a story problem.
- **Kill flag does not automatically equal CRITICAL severity.** Severity follows story impact, not spec category.

**Kill-flag re-scan** (from `microbeat_taxonomy.md` Section 8 + W3 spec):

- `CLIFF: RESOLUTION` present → CRITICAL (story break — resolution leaked)
- First micro-beat of any scene tagged `SETUP: STATE` → HIGH (story risk — Q1 retention bleeds at scene break)
- Engine-specific required subtypes missing → HIGH (engine contract violation)
- More than 25% of total micro-beats tagged `SETUP: STATE` → HIGH (pacing dead, story drifts)
- Zero `INJUSTICE STRIKE` in first 25% of runtime → CRITICAL (no hook for empathy → CPI collapse)
- Zero `TURN` micro-beats in any 30% window → HIGH (escalation flatlines)
- `TURN: REVEAL` without named overturned assumption → MED (W3 quality, not story per se — fix at W3)
- `TURN` beat without WITHHELD/SATISFIED modifier → MED
- Any micro-beat under 80 words → MED (W4 will invent; not story-breaking but W4 quality risk)
- `STATIC-BEAT`: any micro-beat with no nameable movement → severity follows the story finding. If the static beat is at a load-bearing moment (Scene 1, Cliff, IM landing), CRITICAL. If it's a low-impact connective beat, MED.
- Missing required annotations per tag (cruelty modality, punishment type) → LOW unless ambiguity is breaking the W3→W4 handoff.

**Anti-pattern scan** (from `microbeat_taxonomy.md` Section 9): VILLAIN-MONOLOGUE-OVERRUN, MECHANIC-TOLD-NOT-SHOWN, BOND-CLAIMED-NOT-DEMONSTRATED, STAKES-VAGUED, STAKES-FABRICATED, INJUSTICE-LABELED-NOT-INFLICTED, CLIFF-RESOLVES. Each scored by story impact, not spec category.

**Other workflow checks:**
- Movement annotation completeness (any beat missing `Movement:` field).
- Tag distribution (any single tag >40% of beats → flag balance issue).
- Treatment word counts (any treatment >150 words → likely should split into two micro-beats; HIGH if it does, MED if borderline).

### Operator-Facing Output (STORY-LED)

```
## BEAT ANALYSIS: [Title]
*Cold read — scored on its own merits, independent of any prior pass.*

**VERDICT: [PASS / FAIL]**
KEY 1 — CPI: [PASS / FAIL]   |   KEY 2 — BOFU: [PASS / FAIL]
  #1 true-to-show (incl. tone): [PASS/FAIL]  · #2 empathy peaks pre-cliff: [strong/flag]  · #3 thruline aligns: [strong/flag]  · #4 discrete must-resolve cliff: [discrete/flag]

**[Score] / 100 — [LABEL]**

[2–3 sentence executive summary explaining the score in plain, critical language — what works and where it breaks. Model: "Solid, propulsive start; falls apart in the final third; samey empathy builds throughout." Lead with the two-key verdict. If a tone-fit or fidelity break fails Key 2 #1, that is the headline regardless of score.]

### STORY DIAGNOSTIC

[Phase A dimensions: PASS / FLAG / FAIL per dimension with one-line evidence and beat refs. Skip clean PASS dimensions to keep tight — only show FLAG and FAIL plus 1–2 PASS calls worth protecting.]

### Beat-by-Beat
[EVERY beat gets one of four labels:
 - **PROTECT** — load-bearing strength; do not touch in revision, a rewrite would likely make it worse. (Tag + one line on WHY it's protected, so the operator doesn't accidentally cut it.)
 - **STRONG** — works. Tag only, no note needed.
 - **WEAK** — underperforms. One-line diagnosis + a **prescriptive fix** (concrete, actionable — what to change and how, not just "make it stronger").
 - **CRITICAL** — breaks a key or the spine. One-line diagnosis + a **prescriptive fix**, and it also appears in Structured Findings at CRITICAL/HIGH.
 Notes through the STORY lens — what the beat does for the listener, not spec compliance. Every WEAK and CRITICAL beat MUST carry a prescriptive suggestion; a label without a fix is incomplete.]

### Workflow & Compliance Notes
[Phase B findings that didn't escalate to story-level. Polish notes for W3 cleanup or W4 readiness. Bullet list, one line each. Omit entirely if clean.]

### THESIS — Vital Improvements
[2–4 sentences. The overall takeaway: the few changes that actually move this from where it is to a pass (or to a higher tier). Not a list of every note — the vital ones. If it FAILS, name the one or two breaks that must be fixed before anything else is worth doing. This is the section the operator acts on first.]

## STRUCTURED FINDINGS

[Table per Output Model spec. Story findings FIRST, sorted by severity. Workflow findings AFTER, sorted by severity.]
```

### Box Office Scale (label only)

| Range | Label |
|-------|-------|
| 95–100 | BLOCKBUSTER |
| 85–94 | VIRAL |
| 75–84 | CROWD PLEASER |
| 65–74 | POPCORN FLICK |
| 55–64 | STRAIGHT TO STREAMING |
| 45–54 | B-MOVIE |
| <45 | BACK TO THE LAB |

Score by **story judgment**, not formula. Story-breaking issues (CRITICAL Phase A findings) cap at 54. Workflow-only issues do not cap the score on their own — a beat sheet with strong story logic and Phase B compliance noise scores by the story it tells, with workflow gaps treated as polish.

**The score never overrides the verdict.** The /100 is a story-quality label, not the pass/fail. A beat sheet can score well on story craft and still FAIL on Key 2 #1 — a tone or premise the show doesn't support, or Essence the beats bake in wrong. When Key 2 #1 fails, the verdict is FAIL and the score is secondary; lead with the break, not the label.

---

## SUB-FUNCTION 2: BUCKETING (SCRIPT)

**Input:** W4-locked script.
**When to use:** Default for any script analysis request. First question is "can this writer execute at the level we need?" — not "is the structure right?"

### Procedure

Run the full procedure in `bucketing.md`. Load that file at this point (not at W5 start — see tiered load). Bucketing is the script-level instrument for the two keys: the **Quality Gate** answers Key 1, the **Show Fidelity Gate** answers Key 2 #1, and the remaining Key 2 checks (#2 empathy, #3 thruline, #4 cliffhanger type) are surfaced alongside.

**Brief summary of the bucketing.md flow:**

1. Read cold.
2. Run Fast Path kill check (quality kills AND fidelity kills).
3. **Quality Gate (Key 1):** score the three primaries (Opening / Cliffhanger / Writing Style & Voice) — MIN caps the bucket; craft floors (Scene Quality / Dialogue Ratio) pull down within it. Bucket ≥ 3 = PASS.
4. **Show Fidelity Gate (Key 2 #1):** clean PASS/FAIL across surface, Essence, and tone-fit, landing-weighted.
5. Surface Key 2 #2–#4 (empathy / thruline / discrete-cliffhanger) as flags.
6. Adversarial pass (bucket ceiling 3 or 4, or borderline fidelity).
7. Write the verdict + compressed diagnostic. **Verdict = PASS only if Quality Bucket ≥ 3 AND Fidelity = PASS.**

### Integration with the broader architecture

When running Bucketing, also surface:

- Any `voice_style.md` §6 LLM failure mode flag — append as a HIGH structured finding.
- Any `voice_style.md` §7 diagnostic test failure across the script — append as MED (HIGH if it compounds with a quality trigger).
- Any per-show **Essence drift** (Identity, Combat, Power Source, Voice Register) — scored inside the Show Fidelity Gate. Surface as `essence:[dimension]` in structured findings.
- Speaker tagging issues for Multi-Cast VO — untagged narrator prose → HIGH (`craft:speaker-tag-missing`).
- **A tone-fit FAIL is always CRITICAL** — it fails Key 2 #1 and therefore the script, no matter how clean the Quality Gate.

### Operator-Facing Output

Use the readable diagnostic layout from `bucketing.md` → OUTPUT FORMAT (markdown, never a code block — verdict first, scores at a glance, each flaw as what-it-is + why-it-hurts + next action). Append the structured findings block at the end.

### When to offer Deep Dive

After bucketing:
- **Both gates pass (Bucket 4–5 AND Fidelity PASS):** offer Deep Dive. The two keys are cleared; structural analysis is now the bottleneck.
- **Fidelity FAIL:** do NOT advance, even at Bucket 5. The show-truth break is fixed or the script reassigned first — a clean bucket does not buy a pass.
- **Bucket 1–3:** do NOT auto-advance. Execution needs fixing first.
- **Operator explicit request:** always honor.

Phrasing: > "This passes both gates — ready for a structural deep dive. Want me to run it?"

---

## SUB-FUNCTION 3: DEEP DIVE

**Input:** W4-locked script (both gates passed OR operator override) OR W3-locked beat sheet on request ("go deep").
**When to use:** Triggered after both gates pass, or on explicit operator request. Burns significant tokens; only valuable once the two keys are cleared.

Deep Dive presumes the two keys have passed and stress-tests the **structure** that bucketing doesn't reach (IM density, BECAUSE chain, hook-engine contract, skeleton inheritance). It still re-confirms Key 2 at depth — show fidelity on every non-source element, injustice distribution as the empathy engine (#2), and resolution withholding / cliffhanger shape (#4) — because structural breaks can sink BOFU even when the gates passed. This sub-function also feeds W4's revision cycle, but the two-key verdict, not the loop, is the point.

### Step 1 — Structural Read

Single compact table. No prose on clean elements — only expand on problems.

| Element | Finding |
|---|---|
| Hook engine | [which one, or none] |
| Functional skeleton | [chosen skeleton from W1; honored / drifted] |
| Container | [event / journey / neither] |
| Antagonist | [named human / absent / system] |
| Stakes character | [present / absent / inconsistent] |
| BECAUSE chain | [holds / breaks at Beat X] |
| Escalation | [rising / flat / reverses at Beat X] |
| Cliffhanger | [hunger / dread / resolved] |
| Engine contract | [honored / violated] |

### Step 2 — IM Audit

Two artifacts in order: qualification log, then IM table. Do not skip the log — writing the rejects is the forcing function against inflated counts.

**Qualification Log:**

```
CANDIDATES: [n] flagged → [m] survive

EXCLUDED ([n-m]):
- Beat [X]: [candidate] — [reason]
```

**Default-NOT-IM priors** (auto-exclude unless explicitly overridden with written rationale):

- First 30 seconds / Beat 1–2 → default PREMISE. Delta Check fails when no prior understanding exists.
- Character relationship beats (breakup, betrayal, bond) → default THROUGHLINE. Character Swap Test mandatory.
- Combat executing established mechanic → default NOT IM. Clear only if new mechanic revealed.
- Degree escalation in same domain (Stage 4 → Stage 6) → first instance qualifies, subsequent do not. Fails Distinct.
- Narrator-explained discoveries → default UNREALIZED, not ALIVE.
- Pre-announced moments → default DEGRADED, never ALIVE.

**IM Table:**

```
IM Map: [count] IMs ([T1] T1 / [T2] T2) — target: [micro-beats ÷ 2]

| Beat | IM | Tier | Flavor | Stage | Issue? |
|------|----|------|--------|-------|--------|

Flags: [deserts, monotony, flavor concentration, same-domain clusters]
```

Stage (ALIVE / DEGRADED / UNREALIZED) for scripts only. UNREALIZED does not count toward total.

### Step 3 — Combined Scan

Single pass. Report violations only.

- **BECAUSE chain.** Write every scene-to-scene link. Primary action driving each?
- **Injustice distribution.** Map by type. Adjacent variation? Thirds above 50%?
- **Resolution withholding.** Exhale test, Expansion test, Threshold test (see `voice_style.md` §5 — resolution leak; CLAUDE.md → Resolution withholding).
- **Parallel threads.** Failures only. If none: "No parallel threads."
- **Show fidelity.** Fabrication litmus on every non-source element. Entry window. Essence match.
- **`voice_style.md` §7 diagnostic tests** across the full script. Per-scene scores aggregated.
- **`voice_style.md` §6 LLM failure modes** scan.
- **`microbeat_taxonomy.md` anti-patterns** — list matches with beat refs.
- **W3 staging respect (W4 scripts only).** Confirm W4 didn't override W3 physical specs or invent characters/objects/locations.

### Step 4 — Delivery

```
## DEEP DIVE: [Title]

### Top Fix
[One sentence — the single most important fix for CPI.]

### Structural Issues
[Ordered by severity. Each:]
- **[Issue]** — Beat [N]: [What's wrong. Why it hurts CPI. Fix.] (3 sentences max.)

### Protect List
[1–3 strengths to preserve, with beat refs.]

### IM Map
[Table from Step 2]

### Overall
[1–2 sentences.]

## STRUCTURED FINDINGS

[Full table per Output Model spec]
```

### Deep Dive Principles

- **Be specific.** Beat references on every issue.
- **Propose fixes.** When flagging fabrication, name the world-file swap.
- **Prioritize.** 8 issues → lead with the 2 that matter most for CPI.
- **Don't rewrite.** Diagnose only. The rewrite happens in W3/W4 when the operator routes back.

### Multi-Script Comparison

Activate only when analyzing 2+ scripts in the same request.
- Lock audit A before opening B.
- No comparative framing until all audits locked.
- Same event, different staging call, no justification → reopen that IM only.

---

## W4↔W5 AUTOMATION LOOP

W5 is the analysis half of the automation loop architecture. W4 produces a draft → W5 analyzes → structured findings feed back to W4 → W4 revises → loop until quality threshold is met or escalation surfaces to operator.

### Loop Operation

1. **W4 finishes a draft** (scene-by-scene mode: per scene; one-shot mode: full promo).
2. **W5 runs** — Beat Analysis (if beat sheet) or Bucketing → Deep Dive (if script and Bucket 4–5).
3. **Structured findings produced.**
4. **W4 consumes findings** per severity tier:
   - **CRITICAL** → W4 cannot advance. Surface to operator immediately with the finding + W5's proposed fix.
   - **HIGH** → W4 attempts auto-revision ONCE. If finding persists after revision, surface to operator.
   - **MED** → W4 surfaces to operator with proposed fix. Operator decides accept / revise / defer.
   - **LOW** → Append to polish notes; not surfaced.
5. **Revision cycle.** W4 revises, W5 re-analyzes. Loop terminates when no CRITICAL/HIGH findings remain, OR when the operator manually advances despite remaining findings, OR when W4 has attempted revision on the same finding twice and it persists.

**Closure-table verification (X-1, W5 side).** When the artifact is a revision responding to a prior W5 findings table, the next W5 pass opens by verifying the previous revision's closure table — every finding ID marked FIXED or DEFERRED, and FIXED confirmed as the defect being absent from the surrounding block (not merely one line changed) — before reading fresh. This precedes, and does not replace, the cold read.

### Loop Termination Conditions

- Clean run: zero CRITICAL, zero unresolved HIGH.
- Operator override: operator declares "ship it" or "good enough" despite remaining findings.
- Escalation: same HIGH finding persists after 2 W4 revision attempts → W5 surfaces with diagnosis that the issue may be upstream (W3 beat sheet, W2 skeleton, W1 angle).

### Loop Status Note (Architecture-Ready, Not Yet Wired)

The loop architecture is operational at the W5 spec level. The W4 consumption logic (how W4 reads structured findings and triggers revisions) needs wiring at the W4 spec level when the team is ready to enable automation. Until then, W5's structured findings are operator-facing decisions, not W4 inputs.

---

## ANTI-PATTERN CATALOG

Diagnostic labels. Names for problems — used to communicate what's wrong, not prescriptions for what to do. Most cross-link to formal flags in `microbeat_taxonomy.md` Section 9 or `voice_style.md` §6.

**Opening:** World-before-crisis. Protagonist without problem. Backstory dump. Vague stakes. Generic opening. Single-register empathy. Flat command open. *(See voice_style §6 abstract-lore-open, trauma-summary.)*

**Beat-level:** Novelistic drift. Label-as-content. Misinterpretable beat. Zero-function moment. Static beat. *(See taxonomy STATIC-BEAT.)*

**Throughline:** Tension resolution before midpoint. Micro-resolution. Throughline drift. Emotional beat repetition. Window order violation.

**Character:** Passive protagonist. Antagonist without teeth. No emotional anchor. Stakes character investment mismatch. Cartoon-bully villain. *(See voice_style §6 cartoon-bully + §1 Principle 2.)*

**Pacing:** Staccato fragmentation. Over-clever voice. Rushing the event. Summary cuts. Decorative simile. Narrator essay. Tension-release stage directions.

**Structural:** Convenience arrival. Beat overload. Mechanism repetition. IM desert. IM flavor duplication. Classic Woodward. Flat injustice beat. Parallel redundant beats. *(See taxonomy MECHANIC-TOLD-NOT-SHOWN, INJUSTICE-LABELED-NOT-INFLICTED.)*

**Resolution:** Cliffhanger closes instead of opens. Mystery collapse in CTA. Resolution cliffhanger. *(See taxonomy CLIFF-RESOLVES.)*

**Dialogue (script-only):** Dialogue thinness (volume below target). Untagged narrator paragraphs in Multi-Cast VO. W3 staging contradicted by W4 invention. *(See W4 spec Dialogue Generation + W3 Staging Source of Truth.)*

---

## ITERATION

If W5 surfaces findings and the operator chooses to revise:

- **Beat sheet revisions** → return to W3 with the structured findings as input.
- **Script revisions** → return to W4 with the structured findings as input. W4's tiered revision (auto-revise once / surface) consumes the CRITICAL/HIGH findings directly.
- **Structural revisions touching W2 skeleton** → escalate to W2 with the findings.
- **Angle-level problems** → escalate to W1.

W5 does not rewrite. Its job ends when the diagnostic is delivered.

---

## ANALYSIS LOCK

> **Analysis delivered.** [Sub-function name].

If the operator wants additional analysis (e.g., Deep Dive after Bucketing), name the next step explicitly:

> Want a structural Deep Dive? / Want me to re-bucket after revision? / Anything specific to drill into?
