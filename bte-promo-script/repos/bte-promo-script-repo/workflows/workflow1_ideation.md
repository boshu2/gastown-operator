# WORKFLOW 1: IDEATION
# Develops governing elements for a new promo through collaborative shaping with the operator.

---

## REQUIRED INPUTS

**Default posture: build off a validated functional skeleton.** Ideation is primarily a *transplant* job — take a proven beat-order that has already converted and pour a new show's cast, container, and flavor into it. Inventing structure from scratch is the exception, not the rule, because unproven structure carries untested CPI risk.

Four entry types, ranked by how much proven structure they inherit:

| Entry | Skeleton source | What's inherited | What's new |
|-------|-----------------|------------------|------------|
| **Pinned Skeleton** *(PRIMARY)* — choose a functional skeleton from `functional_skeletons.md` and transplant it onto a target show | The library skeleton owns ORDER + engine + emotional arc | Beat functions (injustice / reversal / reveal / inversion), engine, cliffhanger shape | Cast, container, antagonist, combat, power source, IM flavor — poured into the inherited functions |
| **BSV** — same-show reuse of a proven script (itself that show's functional skeleton) | The proven script's own structure | Structure, engine, emotional arc, cliffhanger shape | Protagonist situation, container, antagonist, IM flavors |
| **Adaptation** — cross-show transplant of a proven script's skeleton | Source script structure + engine | Structure, engine, emotional arc | Everything the world change touches — cast, container, stakes, IMs |
| **Fresh Idea** *(rare — discouraged)* — concept, source material, or vague prompt with no skeleton | Nothing — structure is unproven | Nothing | Everything. Building without a validated skeleton means Gate 4 (below) cannot pass on inheritance — the operator must accept the CPI risk explicitly before proceeding. |

The first three are all forms of functional-skeleton inheritance — they differ only in where the skeleton comes from. **Reach for Pinned Skeleton first.** Fresh Idea is a fallback for material that genuinely has no transplantable precedent.

If unclear, ask: "Which functional skeleton are we building off — a library skeleton (pinned), a same-show proven script (BSV), or a cross-show transplant (adaptation)? Or is this a fresh idea with no skeleton?"

---

## COLLABORATION PRINCIPLE

Ideation is a shaping session, not a solo rigor pass. Claude proposes options, operator picks or redirects, Claude pressure-tests the result. If Claude is building alone and handing the operator a finished product to rubber-stamp — stop and return to shaping mode.

Rigor happens at skeleton. Ideation is fast, conversational, rapid-fire. This should feel like bouncing ideas, not filling out a form.

---

## PRE-LOAD (silent, before any operator-facing work)

- **Functional skeletons** (`functional_skeletons.md`) — the candidate beat-orders to transplant. This is the primary structural input; load it first.
- HARD BLOCKS from world file
- Essence block (protagonist identity, setting, voice samples, drift tripwires)
- Entry window data
- Prior promo performance for this IP (from performance_benchmarks.md and hook_engine_library.md)
- Available engines (from series_worlds.md)

If no world file is attached and the IP is unfamiliar, ask the operator to attach it before starting.

---

## CONCEPT SCREEN — FIVE HARD GATES (run on every concept direction, before shaping)

Before a concept earns a shaping session, it must clear five gates. **Each is binary — pass or fail. A concept that fails any one gate is killed or reshaped before it advances.** No "close enough," no "we'll fix it at skeleton." These run the moment a concept direction emerges, ahead of the per-decision LOOP below. Only ideate things that pass all five.

**Gate 1 — Propulsive.** Does the story pull forward — each beat creating the need for the next? A cold listener should feel momentum, not a sequence of events that merely happen in order. Test: state the forward pull in one line — *"I need to know what happens next BECAUSE ___."* If the concept is a situation that sits rather than a pressure that builds, or if the spine reads as AND-THEN instead of BECAUSE → **FAIL.**

**Gate 2 — Not convoluted.** Does the story hold in (roughly) continuous time with minimal hard cuts? Excessive time-jumps, stacked flashbacks, and location-hopping fragment a cold-traffic promo and bleed Q1 retention. Test: count the hard cuts (time passing / location jumps) the concept *requires* to make sense. A single Beat 0 cold-open or one anchoring flashback is allowed. If the spine can't be told without repeated "years later" / "meanwhile" / "back when," → **FAIL or compress** before proceeding.

**Gate 3 — Key throughline.** Does the concept latch onto a universal human experience — *"everyone has had a time when X"*? This is the Relatable Thruline (see `three_pillars.md` → Relatable Thruline; carried into W2 Spine). It is the empathy nerve a cold viewer feels in the first 10 seconds. Test: state it plainly — *"everyone has been underestimated / had something to hide / been betrayed by someone they trusted / been priced at less than they're worth."* If you can't name the single X, the concept has no empathy entry point → **FAIL.**

**Gate 4 — True to a validated functional skeleton.** Is the concept built on a realized, validated functional skeleton — a library skeleton (pinned), a proven same-show script (BSV), or a proven cross-show transplant (adaptation)? Test: name the skeleton and confirm the concept's beats actually map to that skeleton's *functions in order*. Two failure modes: (a) **no skeleton named** → pin one before proceeding, or formally accept Fresh-Idea CPI risk; (b) **false inheritance** — the concept claims a skeleton but its beats don't track that skeleton's functions in order (a reskin that keeps surface flavor instead of structural function) → **FAIL.**

**Gate 5 — Right-show contract.** Does the concept advertise the same show the app delivers? A wrong-show contract sells the install on a show they won't find — the most expensive BOFU failure there is, and it hides behind good craft. Two checks, both off the world file:
- **Check 1 — HARD BLOCK alignment.** Check the concept's climax, cliffhanger, and premise frame against the show's enumerated HARD BLOCKS. Any break → **FAIL.**
- **Check 2 — Tonal alignment.** Draft one sentence — *"from this concept's climax + cliffhanger, a cold installer expects a ___ show"* — surface it for **operator confirmation**, then compare the confirmed sentence to the show's **Tone Constraint** (verbatim). Mismatch → **FAIL** (the "expecting a manhunt thriller, getting warm comedy at an academy" failure).

This is a kill, not a flag. A failure fires the BOFU HARD STOP alarm and is cleared only by the contested override below — never silently, never "fix at bucketing."

### BOFU HARD STOP — the Wrong-Show alarm + contested override

When Check 1 or Check 2 fails (here or at the EXIT GATE), surface this loudly — never a buried note:

```
🚨🚨 BOFU HARD STOP — WRONG-SHOW CONTRACT 🚨🚨
TYPE:  [HARD BLOCK BREAK | TONAL MISMATCH]
WHERE: climax / cliffhanger / premise frame
WHAT:  [the decision that broke it]

— if HARD BLOCK BREAK:  Block [#] — "[verbatim from world file]"
— if TONAL MISMATCH:    Promo promises [X]; the show is [Y]
                        (Tone Constraint: "[verbatim]")

Wrong emotional contract — the install converts on a show the app won't deliver.
This is a BAD BOFU decision. Recommended path: change the decision.
```

**Contested override — not a keyword.** The operator does not pass this with a phrase:
1. **Claude holds the line** — state why this is a bad BOFU call, citing the specific block / Tone Constraint.
2. **The operator must justify**, in writing, why it should pass anyway.
3. **Claude weighs it against the world file.** A justification holds ONLY if it contests the block's *applicability* — *"the world file is stale, canon actually allows this"* (with a pointer) or *"this is an approved, logged CPI experiment."* *"It's cooler," "needs more punch," "stronger cliffhanger"* do **not** hold.
4. **Claude can return NO-GO** and the decision must change. The operator has a say; not an automatic pass.

Accepted overrides are **logged** (what was overridden, the justification accepted) for later audit.

### Additional — Show Fidelity Flag (flag, not a kill)

Alongside the gates, raise a 🔻 **BOTTOM-FUNNEL FLAG** if the concept needs a world element, cast member, or mechanic the show doesn't support in the entry window — e.g. a load-bearing character or setting outside the first ~10–20 episodes. This is a *flag*, not an automatic kill: the operator can accept it and carry it forward, but it must be named here and is hard-checked at the EXIT GATE after lock. (Tonal-register breaks and HARD BLOCK violations are NOT flags — they route to **Gate 5** as a kill.)

### Screen outcome

- **All five gates pass, no fidelity flag** → proceed to shaping.
- **All five pass, fidelity flag raised** → surface the flag, get operator acknowledgment, proceed.
- **Any gate fails** → kill or reshape the concept. Do not advance a failed concept into shaping on the assumption skeleton will fix it. A Gate 5 failure clears only via the contested override above.

---

## THE LOOP (runs on every concept decision)

The five gates above screen the *concept* once, before shaping. This LOOP is the rapid per-decision check that runs *during* shaping — every individual choice (engine, antagonist, cliffhanger) gets tested against these four. The gates ask "is this concept worth building?"; the LOOP asks "does this specific shaping decision hold?"

1. **What is the audience installing for?** (Name 2-3 fun elements they can't get elsewhere. If the elements could describe a different show on the roster — they're not specific enough and the loop fails.)
2. **Who is driving the pressure?** (Named person or force with agency. Not "the situation." If the concept has no one actively making the protagonist's life worse, it will stall at skeleton.)
3. **Why does this story happen TODAY?** (Name the clock.)
4. **Does this exist in the show?** (If the listener can't find it in 10-20 episodes, it's a false promise.)

Four questions. Every concept decision gets tested against these four. If a concept threads generic fun elements, has no driver, has no clock, or promises something the show can't deliver — kill it before it grows.

### Fail-fast

Two consecutive concept directions fail question 1 or 4 → stop iterating on the angle. The concept doesn't have enough material. Reassess the source or try a different entry point.

If two shaping decisions in a row feel vague or interchangeable → reset the angle. The concept isn't specific enough to build on.

### Micro fail-fast

If you cannot answer all four questions in under ~5 seconds, the concept is unclear. Sharpen before continuing.

---

## SHAPING (six check-ins with the operator)

Six elements shaped in this order:

1. **Engine** — hook engine from the library, or a new one with structural justification.
2. **WHY NOW** — the clock that makes this story happen today.
3. **Goal + working virtue** — the quest (what the protagonist wants) + what they won't betray to get it.
4. **Antagonist** — the force pressuring the virtue, offering the easy exit.
5. **Cliffhanger endpoint** — the moment of maximum virtue pressure, the cut to CTA.
6. **Opening** — how to enter once everything else is shaped.

Opening comes LAST. The writer's natural sequence is quest → antagonist → cliffhanger → then opening. The opening inherits constraints from all five prior decisions, which is why it's the most efficient time to shape it.

### Check-in format

At each check-in:

**Step 1 — Ask first.** "Do you already have [element] in mind, or should I surface options?" Let operators who've thought about it skip past the option menu.

**Step 2a — If they have it.** Capture in their words. If it violates a HARD BLOCK, duplicates a base promo element (BSV), or contradicts an earlier locked element — flag immediately. Otherwise record and move on.

**Step 2b — If they want options.** 2-3 distinct shapes (not variations on one idea) + "Other" always available. If the operator's answer reads hesitant or vague, offer: "Want to talk this one through before locking it?"

**Step 3 — Capture.** Restate the pick in one sentence. Operator can redirect before the next check-in.

### Out-of-sequence entry

The order is a default, not a constraint. If the operator reveals something about a later element ("I want to work backward from this cliffhanger"), capture it out of sequence and use it to inform the intervening check-ins. Confirm out-of-sequence captures at their scheduled position.

### BSV / Adaptation annotations

At each check-in, note the `[1:1 PATH]` — what a straight carry-over would look like. Flag when a pick steps on the base promo (duplication risk, diminishing returns).

### Element-specific notes

- **Engine.** When a functional skeleton is pinned, the engine is inherited from it — confirm rather than re-pick. Otherwise default to proven engines for the IP. New engines carry untested CPI risk.
- **WHY NOW.** If no natural clock exists, flag it — skeleton will fight the concept without one.
- **Goal + working virtue.** "Working" = shaped tightly enough to drive antagonist choice, not final wording. Skeleton refines once beats exist.
- **Antagonist.** Must CONTROL something the protagonist NEEDS. A bully without structural function generates pity, not hatred. For Self-Sacrifice engines, name the stakes character here too.
- **Cliffhanger.** Dominant emotion HUNGER, not DREAD. Verb tense test: present progressive ("is becoming") passes, past simple ("became") fails. Payoff mechanics must be plantable at low stakes in earlier beats. **Mechanism fidelity (Gate 5):** both wrong-show failures live in the cliffhanger *mechanism*, never its emotion. Draft the show this cliffhanger's mechanism promises, have the operator confirm it, and check it against the HARD BLOCKS + Tone Constraint. A HUNGER-positive, verb-tense-clean cliffhanger can still be a BOFU kill if its mechanism breaks a block (a beast submits, a cosmetic element weaponizes, the secret detonates into a visible win).
- **Opening.** Fuses identity + genre signal in the first three seconds. Cold audience understands stakes in <10 seconds. Plants the engine's central question in the first 30 seconds.

### Emergent element — Show Promise

Show Promise (the 2-3 core fun elements the listener installs for) is not a separate check-in. It emerges from Engine + Cliffhanger + Opening. Claude names it once shaping is complete, as part of the Lock. Operator can adjust.

---

## TRIGGERED DIAGNOSTICS

Pull these only when a specific problem surfaces during shaping.

**Trigger: "I can't articulate the fun"**
- Do the progression beats change the KIND of thing that's possible, or just the AMOUNT? (Fire → ice → shadow = variety. Fire 1 → fire 2 → fire 3 = repetition.)
- Where does the world expand? List 5+ moments where the audience's model of what's possible changes. If you can't find 5, the concept may not support IM density.

**Trigger: "The protagonist feels thin"**
- What is their defining virtue in ONE word? Can you trace that virtue through every injustice beat? If the virtue changes between beats, the engine is unfocused.
- What do they want FOR THEMSELVES — independent of anyone else? Remove the stakes character. Does the protagonist still have something to lose?

**Trigger: "Self-sacrifice engine selected"**
- Where does the audience's empathy actually land — on the protector or the protected? If every beat is a choice made FOR the stakes character, the protagonist is a support character. They need a personal quest that the sacrifice costs them.

**Trigger: "Choosing between event-based and journey-based"**
- Does the material's strongest escalation come from phases WITHIN a single event, or from moving THROUGH increasingly dangerous territory? The answer decides the shape.

**Trigger: "Compressing source material"**
- For each kept moment: does it do a job no other moment in the promo is already doing? If two moments serve the same structural function, keep the strongest, cut the other.

**Trigger: "New/unproven engine"**
- Has this engine been proven for this show? Check series_worlds for what's worked, what's failed, and WHY it failed.

**Trigger: IM seeds are thin or flavor-duplicated**
- Use the Media Reference Tool (below) to fill gaps through shape transplants from proven references.

---

## PRESSURE TEST (internal — flags-only output)

With shaping complete, run these checks silently. Only failures surface.

- **Empathy Target.** Remove any non-protagonist character the concept leans on — does the emotional engine survive? If not, the protagonist is a support character.
- **BECAUSE Viability.** Can a causal chain trace opening → cliffhanger with BECAUSE links? If the chain requires AND THEN, the concept's spine is missing.
- **Injustice & Stakes Surface.** Does the concept produce natural injustice beats with type variety? Can stakes escalate by DOMAIN across thirds, not just volume?
- **Cliffhanger Payoff.** Does the cliffhanger sell something the show delivers in the entry window? Point to the chapter/episode range.
- **Show Fidelity:**
  - Fabrication: remove each load-bearing element — does the emotional core survive? If no, element must exist in source.
  - Entry Window: load-bearing characters appear within first ~10-20 episodes.
  - Setting: primary setting is the show's entry point or directly connected.
  - Essence: protagonist situation fits Identity. Container fits Setting Essence.

### Flag format

Surface flags conversationally — not a report. For each flag:
1. What failed (named check, specific element)
2. What risk it carries (which performance metric it threatens)
3. What can be done at skeleton to absorb it
4. Which shaping check-in to revisit if the flag is unacceptable

Use 🔻 **BOTTOM-FUNNEL FLAG** for show-fidelity failures.

Operator's response determines lock entry: accepted flags carry forward; redirects loop back to shaping.

---

## EXIT GATE (show fidelity check)

Run once, after governing elements are locked. Not during ideation — after. Every check is binary — pass or fail. No "close enough." No "we'll fix it at skeleton."

- **Fabrication litmus:** If I remove the fabricated element, does the promo still have an emotional core? If no — it IS the promise and must exist in the source. **Pass / Fail.**
- **Entry window:** Will the listener find the hook character within their first 10-20 episodes? **Pass / Fail.**
- **Identity match:** Does the protagonist's situation fit the Identity dimension of the Essence block? Is the virtue derivable from who the Essence says they are? **Pass / Fail.**
- **Combat match:** Does the concept's implied combat match Combat Behavior in the Essence block? If the concept needs the protagonist to fight in a way the source doesn't support — it fails here, not at skeleton. **Pass / Fail.**
- **Power match:** Does the concept's power usage match Power Source in the Essence block? **Pass / Fail.**
- **Setting match:** Does the container trace to Setting Essence? If the concept needs a setting the show's entry window doesn't contain — it fails here. **Pass / Fail.**
- **Right-show contract (Gate 5 re-run):** Re-run both Gate 5 checks against the locked concept's climax, cliffhanger, and premise frame. **Check 1 — HARD BLOCK alignment:** any break against the show's enumerated HARD BLOCKS. **Check 2 — Tonal alignment:** the operator-confirmed *"promo promises [X]"* sentence vs. the Tone Constraint verbatim (e.g. torture/death-collar/manhunt stakes on a warm-comedy underdog). Either failure fires the BOFU HARD STOP alarm and blocks lock — cleared only by the contested override, never silently, never "fix at bucketing." **Pass / Fail.**

Any failure = hard flag. Operator must acknowledge before skeleton work begins. This is the cheapest place to kill a bottom-funnel failure — every phase after this costs more to fix.

---

## LOCK

Seven essentials formalized. Operator confirms before advancing to W2.

1. **Functional Skeleton** — the named, validated skeleton this concept inherits (library / BSV / adaptation), or an explicit "Fresh Idea — no skeleton, CPI risk accepted." W2 builds beats 1:1 against this skeleton's order and functions.
2. **Protagonist Situation** — one sentence: who they are, what's wrong, why a cold audience cares in 10 seconds.
3. **Dramatic Container** — the structure the promo lives inside. Named event or journey.
4. **Cliffhanger Endpoint** — the exact moment the promo cuts to CTA. Dominant emotion HUNGER.
5. **Hook Engine** — from the library, or proposed new one with definition.
6. **WHY NOW** — one sentence: what makes this story happen today?
7. **Show Promise** — the 2-3 core fun elements this concept threads. One of these, stated as a universal human experience, is the Relatable Thruline that cleared Gate 3.

**Gate confirmation:** Note that all five CONCEPT SCREEN gates passed (including Gate 5 — right-show contract, with the operator-confirmed tonal one-liner) and record any accepted show-fidelity flag or logged contested override carried forward.

**Also noted (not locked — tested at skeleton):** working hypothesis for antagonist, stakes character, virtue, throughline, stakes ladder domains, injustice rotation. 3+ IM seeds. Any accepted flags from the pressure test.

**Why only these seven lock:** virtue wording, throughline wording, stakes ladder domains, injustice rotation, and stakes character specifics are structural claims only fully testable once beats exist. Locking them at ideation creates commitments skeleton inherits even when the material wants to go somewhere else. The functional skeleton locks because it is the proven structure W2 inherits 1:1 — it is the one structural commitment that *should* be fixed before beats exist.

### IM Seeds

3+ candidates, one sentence each. Quick read: Sticky / Fun / Distinct. Name the type (Power-Ceiling Reveal, System/Artifact Discovery, Institutional Depth, Status Inversion, World Border). Flavors should not duplicate.

If seeds are thin or flavor-duplicated, invoke the Media Reference Tool. Full IM qualification runs at skeleton — don't over-test here.

### Conversation Name

On lock:

> **Concept locked.** Conversation name: **[Series] · Ideation · [Angle Name] · v1**
> (BSV: **[Series] · Ideation · [Angle Name] BSV · v1**)
> (Adaptation: **[Target Series] · Ideation · [Angle Name] Adaptation from [Source Series] · v1**)

---

## MEDIA REFERENCE TOOL

Available when IM seeds are thin, flavor-duplicated, or missing a type. Not mandatory.

**Step 1 — Identify the type gap.**

| Type | What's Missing |
|------|---------------|
| Power-Ceiling Reveal | No moment where scale extends upward |
| System/Artifact Discovery | No moment where architecture beneath surface is discovered |
| Institutional Depth | No moment where institution's rules are felt through collision |
| Status Inversion | No moment where the weak protagonist does something impossible |
| World Border | No moment where the world extends beyond what's on screen |

**Step 2 — Find the reference.** Ask: "What's the best version of [missing type] you've seen in anime, film, or TV?" If none, suggest from the proven library:

- *Institutional Depth:* Sorting Hat deliberates on Harry; Chunin Exams written test; MIB table-drag.
- *Status Inversion:* Killua assassination mode; Rock Lee drops weights; Mob at 100%.
- *System/Artifact Discovery:* DBZ scouter breaking; Divergent aptitude test.
- *Power-Ceiling Reveal:* Levi's first ODM sequence; Domain Expansion; All Might entrance.
- *World Border:* Greed Island materialization; One Piece Grand Line reveal.

**Step 3 — Reverse-engineer the shape.** Extract the structural mechanic. The shape transfers; the content must come from the target series world file.

**Step 4 — Transplant.** Map the shape onto real elements from the target series.

**Step 5 — Pressure-test.** Native feel? Sticky/Fun/Distinct? If someone who knows the reference would recognize it — transplant failed.
