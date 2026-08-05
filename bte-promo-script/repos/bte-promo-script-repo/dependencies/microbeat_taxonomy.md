# Microbeat Taxonomy

**Status:** Locked
**Load:** Auto-loaded with W3 (Beat Sheet), W4 (Script), W5 (Analysis). Cross-referenced from `CLAUDE.md`.
**Purpose:** Standard taxonomy for tagging every micro-beat in a promo so W3 specifies function precisely, W4 generates against function-specific calibration, and W5 evaluates against the same rubric.
**Companion:** `microbeat_worked_examples.md` (per-engine worked examples; grows over time).
**Calibration:** `scene_exemplars.md` (execution-level rubric per subtype).
**Engine logic:** `hook_engine_library.md` (engine-specific structural rules).

This file defines the structure (5 main tags + subtypes + per-beat modifiers + anti-pattern flags). Execution-level calibration and engine logic live elsewhere. This file is the connective tissue.

---

## 1. The Five Main Tags

### `SETUP`
Establishes character position, world position, stakes architecture, or institutional context. Audience exits a SETUP beat with new information about the world or the protagonist's situation. Foundation, not propulsion — should be compressed. Excessive SETUP indicates the script is explaining instead of showing.

### `INJUSTICE STRIKE`
The protagonist (or the protagonist's bond object / stakes character / a third party the protagonist morally cares about) is harmed in a way that registers as unfair to the audience. Audience exits an INJUSTICE STRIKE wanting the wrong to be answered. Primary driver of Q1 retention and primary fuel of the BECAUSE chain.

### `ESCALATION`
Pressure on the protagonist increases. Stakes raise, antagonist gains, world tightens, or the protagonist's choice worsens their position. ESCALATION beats compound prior beats — they require something to escalate FROM.

### `TURN`
The protagonist's situation, capability, or the audience's model of the world pivots. TURN beats are the IM beats: moments where understanding shifts. Every TURN beat carries a required modifier: `WITHHELD` (planted hook for later) or `SATISFIED` (audience exhales). SATISFIED before CLIFF triggers a flag.

### `CLIFF`
The promo ends mid-trajectory with a structural pull forward into the show. Not "endings" — interruptions calibrated to drive install. A beat that delivers resolution does not pass as CLIFF.

---

## 2. Required Per-Beat Fields

Every micro-beat in W3 carries:

1. **Tag** (one of the five)
2. **Subtype** (from the catalog below; CLIFF beats may carry primary + secondary)
3. **Set-piece strength: Low / Medium / High** — distinguishes structurally-correct beats from trailer-promotable ones. A `TURN: REVEAL` via phone call is structurally identical to one via public bow but trailer-wise incomparable. Promos need an average set-piece strength above Medium; runs of Low beats kill trailer energy regardless of structural soundness.

   **Tie-breaker:** when uncertain between two levels, default to the lower. A slap is Medium (not High); a pulled knife is Medium (not High) unless it's used in front of a crowd or the blade catches a defining visual. Defaulting low keeps the spec honest — operators tend to over-call High.

   **Genre-aware anchors** (High examples by genre):
   - *Fantasy / action / system-based*: monster transformation, weapon reveal, public bow, crowd reversal, system alert, beast evolution, magical eruption, arena reversal
   - *Romance / romantasy*: public confession, wedding interruption, ballroom reveal in front of court, mate-bond manifestation, kiss-as-power-activation, royal court humiliation
   - *Contemporary / social drama*: family-dinner confrontation, courtroom verdict, press conference reveal, public arrest, boardroom reversal, paternity reveal, walk-out scene with crowd watching
   - *Returnee / identity-inversion*: convoy arrival, public identity reveal, formal salute / bow from authority figures, character switching context (e.g. drifter to billionaire), tribunal reversal

   Medium across all genres: intimate room confrontations, two-character physical exchanges, one-on-one threats without crowd. Low across all genres: phone calls, voiceover, off-screen action, characters describing events rather than witnessing them.
4. **TURN beats only:** WITHHELD / SATISFIED modifier
5. **TURN: REVEAL beats only:** named overturned assumption (what audience belief is being inverted — if you cannot name it, the beat is exposition)
6. **INJUSTICE STRIKE beats only:** specific cruelty modality (which object/act carries the wrong — generic abstractions fail)
7. **One-line beat description** (what the audience sees/hears)
8. **Pressure** — the throughline: why this beat matters structurally and where it points. The home for analysis that must NOT appear in the W3 prose body. Inherited and refined from the W2 brief's THROUGHLINE PRESSURE.
9. **Dialogue plan** — who carries the beat *in voice* and the key exchange, not just the single load-bearing line. Mark the beat **`carry: dialogue`** (the move plays primarily through spoken exchange), **`carry: narration`** (primarily narrator / interior-monologue prose), or **`carry: mixed`**, and name the load-bearing line(s) plus the reactive lines that frame them (antagonist jab, crowd reaction, witness aside, authority ruling). This is the field that lets W4 *render* dialogue instead of inventing it or narrating around its absence. A beat with no exchange named defaults to narration — and a sheet weighted to `carry: narration` produces a script that misses the W4 Dialogue-Ratio Gate (see Section 8 dialogue-balance review flag). This does not relax field 7's leak rule: the prose body still renders event, not meaning; the *exchange* is specified here and in the load-bearing lines.

---

## 3. Subtype Catalog

### SETUP subtypes

| Subtype | Definition | Anchor example |
|---|---|---|
| `STATE` | Establishes character or world state without specific stakes consequence. Default subtype; should not exceed 25% of total micro-beats; **may not be the first beat** of the promo. | Tristan chained in throne hall (GU Love Bite, $2.64 CPI) |
| `STAKES-ANCHOR` | Material consequence specification with **countable nouns + hard deadline**. Generic stakes ("rent due") fail this subtype. | "$900 by sundown / changing the locks myself" (GU9750 Pawn Shop, $4.20 CPI); "sold the house for 500K crystals / three months back rent / lose the kitchen" (GU9892 Pearl Crystal, $3.19 CPI) |
| `BOND-DEMO` | Protagonist + bond object proving the bond through small physical action. Not the verbal claim of bond — the *action* that earns audience investment. Required in Self-Sacrifice engines. | Brian making noodles for Stella and Harper at midnight (GU TBB Supermarket, $2.95 CPI); ~~Cloud nuzzling Ren in the dorm (GU9892 Pearl Crystal)~~ ⚠ **off-essence — "Cloud" is not a WBT character (HARD BLOCK 8); for WBT the bond object is Ren's parents, not the spore** |
| `INSTITUTIONAL-SETUP` | The world's ranking/ceremony/exam system is shown in operation. Distinct from `TURN: INSTITUTIONAL-DEPTH` — this is the scene that *establishes* the institution; the TURN is where the institution reveals something unexpected. | Weapon Selection ceremony opening (GU MVS Weapons); Hatching Chamber egg ceremony (GU WBT Spore Cloud); Aristocracy Exam registration (GU SoaW) |

### INJUSTICE STRIKE subtypes

| Subtype | Definition | Anchor example |
|---|---|---|
| `PHYSICAL` | Bodily harm to protagonist or bond object. | Niall plucks feathers from Midas one by one (GU RBM Daddy Issues, $4.57 CPI) |
| `SOCIAL` | Public humiliation, identity erasure, named-cruelty in front of an audience. | "She kept the menu. She kept it. To laugh at." (GU9750 Pawn Shop) |
| `INSTITUTIONAL` | Cruelty delivered through institutional language or formal ritual. The world's *rules* are the cruelty. | The wedding officiant's scroll listing every right stripped: "She will bear his name. She will not speak in council. She will not hold property." (GU Princess Ebony, $4.56 CPI) |
| `NEAR-EXPOSURE` | Physical danger AND detection danger present in the same beat. Engine-specific to Secret Identity. | Loop's accusation "He used an ability!" during Quinn's spear catch (GU MVS Weapons) |
| `INTIMATE-BETRAYAL` | Betrayal revealed during the act of physical violence. The information lands at the moment of maximum bodily impact. | "Wyatt sends his regards" delivered immediately before the bite (GU Love Bite, $2.64 CPI); Lydia kissing Niall while plucking Midas's feathers (GU RBM Daddy Issues) |
| `RELATIONAL` | A loved one (parent, bond object, stakes character) is targeted by the antagonist **because of** the protagonist's relationship to them. The antagonist's calibration is the protagonist's wound — the third party is the surface victim but the protagonist is the intended pain recipient. Operator test: would the antagonist still do this if the protagonist weren't present / wouldn't know? If no → RELATIONAL. If yes → COLLATERAL-STRIKE. | Niall killing Midas to wound Jasper through the bond (GU RBM Daddy Issues, $4.57 CPI); Wei threatening to extract Cloud for materials if Ren fails (GU9892 Pearl Crystal); "You sold your house for that? Your son has failed you" — hypothetical cruelty directed at parents but targeting Ren |
| `COLLATERAL-STRIKE` | Injustice against a third party caused by or radiating from the protagonist's position or actions, **without** the antagonist specifically targeting the protagonist through them. The third party is caught in the splash, not used as a weapon. Distinct from RELATIONAL (above) and from `ESCALATION: COLLATERAL-COST` (cumulative cost, not discrete strike). | The bruised woman: "My husband beat me because of YOU! Because of what you did at the ball!" (GU Princess Ebony) — her husband isn't trying to hurt Ebony, he's exercising power on his wife given social permission by Ebony's defiance |

### ESCALATION subtypes

| Subtype | Definition | Anchor example |
|---|---|---|
| `EXTERNAL` | Antagonist gains capability, advantage, or scale. World tightens around the protagonist. | Chris Jones's bodyguards arriving in formation (GU9750 Pawn Shop); Kruger receiving Dragonite pill |
| `INTERNAL-REFUSAL` | The protagonist's choice worsens their own position. Each refusal of an easier exit costs more. Engine-specific to Self-Sacrifice. **Engine tie-breaker:** in non-Self-Sacrifice engines, protagonist defiance tags as `TURN: AGENCY-CLAIM` instead. | Karl turning down the WilderBear, then getting banned from the event, then running back for Rae (GU FLBM Dragon Battle, $3.64 CPI); Nathan refusing wardship from House Raven (GU SoaW) |
| `COLLATERAL-COST` | Compounding price paid by the bond object or stakes character as a consequence of the protagonist's ongoing choices. The *cumulative* harm to the bond — not the discrete strike. Does not count toward injustice density. | ~~Cloud's wounds compounding from each new Stalker/Spire encounter (GU9892 Pearl Crystal)~~ ⚠ **off-essence — the spore is not a wound-taking companion "Cloud" (HARD BLOCK 8/1)**; Hawk and Rae taking damage across multiple trial events |

### TURN subtypes

Each TURN beat carries a required `WITHHELD` / `SATISFIED` modifier. SATISFIED TURNs that close the BECAUSE chain emotionally before the CLIFF trigger a flag for review.

| Subtype | Definition | Maps to existing IM type |
|---|---|---|
| `REVEAL` | New information overturns a specific named audience assumption. Required field: which assumption is being inverted. If author cannot name the assumption, the beat is exposition and gets retagged or cut. Catch-all of last resort — use one of the five named IM types below if it fits. | — |
| `POWER-ACTIVATION` | The protagonist's ability activates for the first time. Distinct from STATUS-INVERSION because it is character-state-change, not just audience-model-update. Must be defamiliarized/paradoxical to protect Q2 retention. | — |
| `AGENCY-CLAIM` | The protagonist defies, refuses, or claims voice **without power activation**. Distinct from POWER-ACTIVATION (no ability activates) and from `ESCALATION: INTERNAL-REFUSAL` (which worsens the protagonist's position — AGENCY-CLAIM claims it). The beat that converts a passive-victim opening into an active protagonist. **Engine tie-breaker:** in Self-Sacrifice engines this beat is unavailable — use `ESCALATION: INTERNAL-REFUSAL` instead. In all other engines, AGENCY-CLAIM is the correct tag for protagonist defiance/voice-claiming. | — |
| `STATUS-INVERSION` | The protagonist's social or power position visibly flips. Audience sees who they WERE vs who they ARE. | Status Inversion IM |
| `SYSTEM-DISCOVERY` | An artifact, system, or mechanic reveals its rules to the audience through use. | System/Artifact Discovery IM |
| `POWER-CEILING` | The scale of what's possible in the world expands in front of the audience. | Power-Ceiling Reveal IM |
| `INSTITUTIONAL-DEPTH` | An institution's depth or hidden architecture is revealed during use. Distinct from `SETUP: INSTITUTIONAL-SETUP` — that establishes the institution; this exposes what was hidden inside it. | Institutional Depth IM |
| `WORLD-BORDER` | The world's scope expands — new realm, new system, new category of being. | World Border IM |

### CLIFF subtypes

CLIFF beats carry **primary + at most one secondary** subtype. Secondary must be the *second-strongest pull*, not every applicable one. If a CLIFF could plausibly carry three subtypes, the third drops off. This prevents tagging inflation — every CLIFF appearing to "do everything" collapses discrimination.

Example: "Who else knows you opened it?" is `QUESTION-MULTIPLIER (primary) + DREAD-FORWARD (secondary)` — opens new questions AND seeds inescapable threat. The HUNGER-FORWARD reading (audience installs to see the Sterling Dynasty payoff) is the third-strongest pull and drops off.

| Subtype | Definition | Performance pattern |
|---|---|---|
| `QUESTION-MULTIPLIER` | Opens 2+ new questions the promo never raised. Strongest Q4 hold + strongest CTR. | GU9892 Pearl Crystal "ancient knowledge erupts"; GU RBM Daddy Issues red-eye reveal; GU9686 SS Eviction shadow army vision |
| `HUNGER-FORWARD` | Anticipates a payoff visible in the show. Audience installs to see the reward. | GU FLBM Dragon Battle royal-blood reveal; GU WBT Spore Cloud blueprint mode |
| `DREAD-FORWARD` | Anticipates an inescapable danger. Audience installs to see how the protagonist survives. | GU MVS Weapons (full vampiric reveal, both bullies watching) |
| `RESOLUTION` | **KILL FLAG.** Delivers the central payoff inside the promo. Q4 collapses; CTR may rise from curiosity but CTI suffers. | GU FLBM Trading Up "Hawk and Rae are extraordinary"; GU TS Tragic Trillionaire eyes-flash-gold reveal |

---

## 4. Universal vs Engine-Specific Subtypes

| Subtype | Universal | Self-Sacrifice required | Secret Identity required | Underdog Awakening required |
|---|:---:|:---:|:---:|:---:|
| SETUP: STATE | ✓ | | | |
| SETUP: STAKES-ANCHOR | ✓ | | | |
| SETUP: BOND-DEMO | | **required** | | |
| SETUP: INSTITUTIONAL-SETUP | ✓ | | | |
| INJUSTICE STRIKE: PHYSICAL | ✓ | | | |
| INJUSTICE STRIKE: SOCIAL | ✓ | | | |
| INJUSTICE STRIKE: INSTITUTIONAL | ✓ | | | |
| INJUSTICE STRIKE: NEAR-EXPOSURE | | | **required** | |
| INJUSTICE STRIKE: INTIMATE-BETRAYAL | ✓ | | | |
| INJUSTICE STRIKE: RELATIONAL | ✓ | | | |
| INJUSTICE STRIKE: COLLATERAL-STRIKE | ✓ | | | |
| ESCALATION: EXTERNAL | ✓ | | | |
| ESCALATION: INTERNAL-REFUSAL | | **required** | | |
| ESCALATION: COLLATERAL-COST | ✓ | | | |
| TURN: REVEAL | ✓ | | | |
| TURN: POWER-ACTIVATION | ✓ | | | **delayed past 40%** |
| TURN: AGENCY-CLAIM | ✓ | | | |
| TURN: STATUS-INVERSION | ✓ | | | |
| TURN: SYSTEM-DISCOVERY | ✓ | | | |
| TURN: POWER-CEILING | ✓ | | | |
| TURN: INSTITUTIONAL-DEPTH | ✓ | | | |
| TURN: WORLD-BORDER | ✓ | | | |
| CLIFF: any non-RESOLUTION | ✓ | | | |

---

## 5. Worked Example A — GU9750 Pawn Shop (Identity Inversion / Hidden Hero — $4.20 CPI / 14.4% CTI)

Companion iteration GU9587 ($3.38 CPI / 22.3% CTI) is the cleaner ship version of the same body script — same tagging applies.

| # | Tag · Subtype | Modifier | Set-piece | Beat |
|---|---|---|---|---|
| 1 | SETUP · INSTITUTIONAL-SETUP | — | Med | Pawnshop opening + "Sterling blood is back" (planted institutional mystery) |
| 2 | SETUP · STATE | — | Low | Apartment hallway, $43 in wallet, landlord shouting through door |
| 3 | SETUP · STAKES-ANCHOR | — | Low | "Twenty-four hours to come up with the money or you're getting evicted" |
| 4 | SETUP · STATE | — | Med | Flashback to orphanage at age 5, nun, locket pressed into hand |
| 5 | TURN · REVEAL | WITHHELD | Med | Shopkeeper pulls gun, calls someone: "The Sterling blood is back" (assumption overturned: this locket is institutionally tracked) |
| 6 | INJUSTICE STRIKE · INTIMATE-BETRAYAL | — | Med | Damian's call: Jenifer at Blueprint Hotel |
| 7 | INJUSTICE STRIKE · INTIMATE-BETRAYAL | — | High | Jed walks into Room 109, finds Jenifer in bed with Chris Jones |
| 8 | INJUSTICE STRIKE · SOCIAL | — | Med | Chris's "kept the menu" monologue (8 specific cruelties in 70 words) |
| 9 | INJUSTICE STRIKE · PHYSICAL | — | High | Bodyguards beat Jed; Chris narrates "By next Tuesday nobody will remember your name" |
| 10 | ESCALATION · EXTERNAL | — | High | Body dumped in SUV, thrown in alley dumpster |
| 11 | TURN · POWER-ACTIVATION | WITHHELD | High | Locket opens, eagle vision floods Jed, healing |
| 12 | TURN · WORLD-BORDER | WITHHELD | Low | Phone call from Sylvia Sterling (audio-only, but scope expands to "Sterling Dynasty") |
| 13 | INJUSTICE STRIKE · INSTITUTIONAL | — | Med | Landlord still threatening locks despite everything |
| 14 | SETUP · STATE | — | Low | Shower, frayed white button-down, locket inside collar |
| 15 | INJUSTICE STRIKE · SOCIAL | — | Med | Crystal Hollow hostess: "Shall I call the homeless shelter?" |
| 16 | TURN · STATUS-INVERSION | SATISFIED | High | Hostess sees Sterling name on screen, color leaves cheeks, bows (trips SATISFIED-risk review; clears per function-based rule — see modifier check below) |
| 17 | TURN · INSTITUTIONAL-DEPTH | WITHHELD | Med | Black access card embossed with the same eagle as the locket |
| 18 | SETUP · STATE | — | Low | Private elevator rises, Jed alone in Celestial Suite |
| 19 | ESCALATION · EXTERNAL | — | High | Chris arrives with surveillance photos, splint on nose, manager flanking |
| 20 | INJUSTICE STRIKE · SOCIAL | — | High | Chris's "do you know who my father is" monologue + photos |
| 21 | TURN · REVEAL | WITHHELD | High | Doors open from corridor side; six grey-suited men walk in (assumption overturned: Chris is not the most powerful person in this room) |
| 22 | TURN · STATUS-INVERSION | WITHHELD | High | Sylvia Sterling stops in front of Jed, drops to a bow; six guards bow in unison |
| 23 | CLIFF · QUESTION-MULTIPLIER + DREAD-FORWARD | — | High | "My Lord Sterling. Who else knows you opened it?" |

**Tag distribution check:** 6 SETUP / 7 INJUSTICE STRIKE / 2 ESCALATION / 7 TURN / 1 CLIFF = 23 beats. ✓

**Subtype check:**
- STATE: 4/23 = **17.4%** (within 25% cap) ✓
- TURN: 7/23 = **30.4%** (Identity-Inversion engine, expected ratio range) ✓
- No 30%-of-runtime window without a TURN beat ✓
- First beat is `SETUP: INSTITUTIONAL-SETUP` with planted hook (not STATE) ✓

**Set-piece check:**
- Average set-piece strength: ~Med-High (Pawn Shop runs on cinematic specificity — beats 7, 9, 10, 11, 19, 20, 21, 22, 23 are all High) ✓
- Low-spectacle runs: beats 2–4 (3 consecutive Low/Med opening beats — flag for review, but holds because beat 5 lands a High reveal immediately after) ✓

**Modifier check:** All TURN beats are WITHHELD except beat 16 (hostess bow), which is SATISFIED and trips the review flag. Per the function-based rule (Section 8): the next non-SETUP destabilizing beat is beat 19 (ESCALATION:EXTERNAL — Chris arrives with surveillance photos), which arrives before the next TURN (beat 21). Flag clears. ✓

---

## 6. Worked Example B — GU9892 Pearl Crystal (Self-Sacrifice — $3.19 CPI / 19.1% CTI)

(Same beat sheet as GU9672, $6.37 CPI — production iteration GU9892 represents tighter execution of identical structure. Tagging applies to both.)

Included to address Identity-Inversion bias in the Pawn Shop example. Self-Sacrifice engines distribute beats differently: heavier on ESCALATION:INTERNAL-REFUSAL, lighter on TURN density, anchored by BOND-DEMO early.

> ⚠️ **OFF-ESSENCE BEAT SHEET — TAGGING ONLY, NOT A CANON MODEL (WBT HARD BLOCK 8 + 1 + 3).** This Pearl Crystal beat sheet is retained to illustrate *micro-beat tagging* for a Self-Sacrifice engine. Its **content is off-canon**: it treats Ren's spore as a sentient companion "Cloud" that nuzzles, absorbs hits, gets thrown/caught, bleeds, shields, and fires a "Decay" attack — violating HARD BLOCK 8 (no character named Cloud), HARD BLOCK 1 (spore is +10% strength only, never a combat power), and HARD BLOCK 3 (no beast-vs-beast / beast-as-weapon). The same beat sheet is dismantled in `working files/documentation/WBT_SummitClimb_v1.md`. **Read it for the tag/subtype distribution; do not reuse any "Cloud" beat as WBT content.**

| # | Tag · Subtype | Modifier | Set-piece | Beat |
|---|---|---|---|---|
| 1 | SETUP · INSTITUTIONAL-SETUP | — | High | Assessment Day arena, ranking system on display, ceremony stage |
| 2 | SETUP · BOND-DEMO | — | Med | Flashback: Cloud nuzzles Ren in the dorm, mushrooms grow on his arms |
| 3 | INJUSTICE STRIKE · PHYSICAL | — | Med | Jin punches Ren at assessment, Cloud absorbs hits |
| 4 | INJUSTICE STRIKE · SOCIAL | — | High | Wei's "class disappointment" + "Fungus Freak!" stadium chant, parents in crowd |
| 5 | INJUSTICE STRIKE · INSTITUTIONAL | — | High | Iron Rank verdict: mines for life, Cloud extracted for materials |
| 6 | ESCALATION · EXTERNAL | — | Med | Wei's terms: Gold artifact by sunset or expulsion + extraction |
| 7 | SETUP · STAKES-ANCHOR | — | Med | Parents arrive: "sold the house / 500K crystals / 3 months back rent / lose the kitchen" |
| 8 | INJUSTICE STRIKE · SOCIAL | — | Med | Jin steals pendant: "If you're so special, then you don't need it!" |
| 9 | SETUP · STATE | — | High | Race begins, bioluminescent forest, students charging forward |
| 10 | TURN · SYSTEM-DISCOVERY | WITHHELD | High | Lurker barrier + Cloud's mana signature too faint (overturned: weakness as advantage) |
| 11 | INJUSTICE STRIKE · PHYSICAL | — | High | Jin grabs Cloud, throws him at Lurker |
| 12 | ESCALATION · INTERNAL-REFUSAL | — | High | Ren catches Cloud mid-air AND grabs pendant from Jin's neck — refuses to flee |
| 13 | TURN · WORLD-BORDER | WITHHELD | High | Mushrooms erupt across Ren's body — he belongs to this ancient ecosystem (overturned: Cloud is not isolated; world has hidden architecture) |
| 14 | ESCALATION · EXTERNAL | — | High | Gold level fog, Shadow Stalker emerges |
| 15 | INJUSTICE STRIKE · COLLATERAL-STRIKE | — | High | Stalker rakes Cloud; Cloud bleeds for Ren's choice |
| 16 | ESCALATION · INTERNAL-REFUSAL | — | High | Ren gives the Gold artifact to Taro instead of taking the easy win |
| 17 | TURN · WORLD-BORDER | WITHHELD | High | Summit platform reveals: ancient ruins above the clouds, Pearl Crystal on altar, dormant Crystalline Spires (audience's model expands: this place is older and bigger than the Academy knows) |
| 18 | INJUSTICE STRIKE · PHYSICAL | — | High | Jin punches Ren into the blast zone; Spires fire |
| 19 | ESCALATION · COLLATERAL-COST | — | High | Cloud shields Ren, takes the Spire blasts, fragments piece by piece |
| 20 | ESCALATION · INTERNAL-REFUSAL | — | High | Ren throws himself in front of Cloud; reciprocal sacrifice |
| 21 | TURN · STATUS-INVERSION | WITHHELD | High | Cloud's "Decay" snaps the necklace string Jin used to hold Ren; Jin flies into blast zone |
| 22 | TURN · POWER-ACTIVATION | WITHHELD | High | Ren touches Crystal; mushrooms blaze across body; knowledge floods through him |
| 23 | TURN · REVEAL | WITHHELD | High | "This isn't just a—" (overturned: Pearl Crystal is not a power artifact, it is something ancient the Academy never knew existed) |
| 24 | CLIFF · QUESTION-MULTIPLIER + HUNGER-FORWARD | — | High | "The light swallowed everything" + CTA: what did Ren see, why did it answer him, how does an Iron-rank know things no professor does |

**Tag distribution check:** 4 SETUP / 7 INJUSTICE STRIKE / 6 ESCALATION / 6 TURN / 1 CLIFF = 24 beats. ✓

**Subtype check:**
- STATE: 1/24 = **4.2%** (well within cap) ✓
- TURN: 6/24 = **25.0%** (Self-Sacrifice engine, upper end of typical range — unusually IM-dense for this engine) ✓
- INTERNAL-REFUSAL: 3 occurrences (engine-required, present) ✓
- BOND-DEMO: present (beat 2) ✓
- COLLATERAL-STRIKE + COLLATERAL-COST both present, distinguished cleanly ✓
- First beat is `SETUP: INSTITUTIONAL-SETUP` with the ranking ceremony visible (not STATE) ✓

**Set-piece check:** ~85% of beats run High. The single Low/Med opening sequence (beats 1–8) carries because beat 1 is High institutional spectacle. ✓

**Cross-script comparison:** Pawn Shop runs ~30% TURN beats / ~9% ESCALATION beats (Identity-Inversion engine drives dramatic-irony reveals). Pearl Crystal runs ~25% TURN beats / ~25% ESCALATION beats (Self-Sacrifice engine drives escalating-cost refusals, but Pearl Crystal sits at the high end of TURN ratio for the engine because the script unusually dense in world-discovery moments). Same CPI-tier outputs; different distributions. Engine determines the distribution shape, not a universal target ratio.

---

## 7. Integration Touchpoints

- **W1 Ideation:** Engine selection commits the script to required subtypes (per Section 4 matrix); W1 exit gate confirms the angle supports them.
- **W2 Skeleton:** Each high-level beat declares expected micro-beat count + dominant tag(s).
- **W3 Beat Sheet:** Each micro-beat carries all required fields per Section 2.
- **W4 Script:** Scene Gate reads tag + subtype + fields, runs calibration tests from `scene_exemplars.md`.
- **W5 Analysis:** Beat Analysis grades each micro-beat (Critical / Weak / Strong) with subtype-calibrated prescriptive suggestions; runs anti-pattern flag scan (Section 9).
- **CLAUDE.md:** Add `microbeat_taxonomy.md` to Reference files; auto-load with W3, W4, W5.

---

## 8. Kill Flags and Review Flags

**Kill flags** (mechanical structural failures — fail by definition):

- `CLIFF: RESOLUTION` → fails.
- First beat tagged `SETUP: STATE` → fails. Promo openings must be charged: INJUSTICE STRIKE (any subtype), TURN (any subtype), SETUP: INSTITUTIONAL-SETUP with planted hook, or SETUP: STAKES-ANCHOR with deadline visible in the first beat.
- Underdog Awakening engine with `TURN: POWER-ACTIVATION` before 40% of runtime → fails delayed-activation rule.
- Self-Sacrifice engine with zero `SETUP: BOND-DEMO` micro-beats in the first 30% of runtime → fails tenderness rule.
- Self-Sacrifice engine with zero `ESCALATION: INTERNAL-REFUSAL` micro-beats → fails escalating-cost principle.
- Secret Identity engine with zero `INJUSTICE STRIKE: NEAR-EXPOSURE` micro-beats → fails dual-layer engine rule.
- More than 25% of total micro-beats tagged `SETUP: STATE` → fails specificity (too much undifferentiated exposition).
- Zero `INJUSTICE STRIKE` micro-beats (across all subtypes including COLLATERAL-STRIKE) in the first 25% of runtime → fails injustice density rule.
- Zero `TURN` micro-beats in any 30%-of-runtime window → fails IM-density / no-deserts rule.
- `TURN: REVEAL` without named overturned assumption → fails (the beat is exposition).
- `TURN` beat without WITHHELD/SATISFIED modifier specified → fails (incomplete tag).

**Review flags** (do not auto-kill; surface for operator decision):

- Any `TURN` beat marked `SATISFIED` before the CLIFF → flag. Function-based rule: the SATISFIED beat is safe only if **the next destabilizing beat (ESCALATION or INJUSTICE STRIKE) arrives before the next TURN or CLIFF.** If the script's next non-SETUP beat after the SATISFIED TURN is another TURN or the CLIFF, the satisfaction has settled — the SATISFIED TURN promotes to kill. The Pawn Shop beat 16 (hostess bow, SATISFIED) clears the flag because the next non-SETUP beat is beat 19 (ESCALATION:EXTERNAL — Chris arrives with photos), which arrives before the next TURN (beat 21). Operators are not permitted to insert filler SETUP beats to game the gap.
- Three or more consecutive beats without either a High set-piece OR a sharp audience-model shift (INJUSTICE STRIKE or TURN succession) → flag. Trailer energy bleeds out when neither visual spectacle nor narrative escalation is doing the work. Three Medium beats can be fine if they are volatile; three merely functional beats are the problem.
- Tag distribution skew (>40% of beats in any single tag) → flag. Indicates the script is mono-functional (all setup, all escalation, etc.).
- A `CLIFF` or `TURN: POWER-ACTIVATION` whose payoff has no earlier plant in the sheet → flag (broken plant spine). The converting payoffs — a climax that detonates a planted detail, an activation the audience was seeded for — require an earlier seed; an unplanted payoff lands weaker and leaves a hole in the Plant/Payoff Map.
- **Light-dialogue sheet:** more than ~40% of non-SETUP micro-beats marked `carry: narration` (field 9), or dialogue/mixed-carry beats whose Dialogue plan names no actual exchanges → fix before lock, don't just flag. A light-dialogue sheet hands W4 a blueprint it can only render below the Dialogue-Ratio Gate hard floor (workflow4: 50% Multi-Cast VO / 45% Single-Cast), which blocks delivery and forces a script-level Deficit Pass. W3 rebalances the convertible beats to `carry: dialogue` / `carry: mixed` and names their exchanges before lock; if the imbalance is structural, escalate to operator (see workflow3 light-dialogue gate).
- **Untraced canon:** any set-piece, world-mechanic, lore claim, or quantity (tenure/count/date/duration) in the sheet that cannot be pointed to a world-bible line (or an approved premise note) → flag for operator approval (fires `CANON-FABRICATED`, Section 9). New *staging* of a committed move is fine; new *world behavior / lore / quantity* is not.

**On engine-dependent target ratios:**

Universal floor: no 30%-of-runtime TURN-desert.
Engine-specific target ranges are emerging from the data but not yet locked:
- Identity Inversion / Secret Identity engines: ~25–35% TURN beats (high reveal density)
- Self-Sacrifice engines: ~15–25% TURN beats (compensated by ESCALATION:INTERNAL-REFUSAL density)
- Underdog Awakening engines: ~20–28% TURN beats (with POWER-ACTIVATION delayed)

These ranges will tighten as more paired data accumulates. For now, deviations outside these ranges flag for operator review rather than auto-kill.

---

## 9. Anti-Pattern Flags

Distinct from tags (what beats *are*) and kill flags (mechanical structural failures). Anti-pattern flags identify what beats *fail at being* — surface failure modes within otherwise tag-compliant beats. Run by W5 as a separate scan; raised as revision requests.

| Flag | Definition | Counter-pattern (winner) |
|---|---|---|
| `VILLAIN-MONOLOGUE-OVERRUN` | Antagonist speech exceeds 30 words without intervening physical action or protagonist reaction. The cruelty becomes lecture; embeddedness (per `scene_exemplars.md`) is lost. | Chris Jones's monologue in GU9750 Pawn Shop is 70 words but broken across 8 distinct cruelties + Jed's reactions + the slap-cheek interruptions — the speech is paced through the action, not delivered as a block. |
| `MECHANIC-TOLD-NOT-SHOWN` | A world rule, system mechanic, or institutional logic is *named* in dialogue or narration without being *demonstrated* in the same beat. The audience learns it as exposition instead of discovering it through use. | `TURN: SYSTEM-DISCOVERY` in GU9892 Pearl Crystal — the audience learns "Cloud's mana signature is below detection threshold" because they *see* the Lurker ignore Cloud, not because someone explains the rule. *(⚠ craft mechanism is valid; the "Cloud" artifact is off-essence — HARD BLOCK 8 — restage as the spore/Ren, no companion.)* |
| `BOND-CLAIMED-NOT-DEMONSTRATED` | The protagonist (or narrator) states the strength of the bond verbally without a `SETUP: BOND-DEMO` beat earning the audience's belief through small physical action. The bond is asserted; the audience does not feel it. | Cloud nuzzling Ren in the dorm + mushrooms growing on Ren's arms during punches (GU9892 Pearl Crystal) — the bond is demonstrated in two specific physical actions before the script ever names it. *(⚠ off-essence example — "Cloud" is not a WBT character, HARD BLOCK 8; for WBT, demonstrate the bond through Ren ↔ parents instead.)* |
| `STAKES-VAGUED` | A `SETUP: STAKES-ANCHOR` beat fails to name countable nouns + hard deadline. Generic stakes ("rent due," "expensive treatment," "could lose everything") fire this flag. | GU9892 Pearl Crystal's "sold the house for 500K crystals / three months back rent / lose the kitchen if Iron when semester ends" — every variable is countable and clocked. |
| `STAKES-FABRICATED` | A `SETUP: STAKES-ANCHOR` beat carries specific countables but the countables are *not grounded in the source show*. The numbers are invented by the promo to clear the STAKES-VAGUED flag — but the listener who installs to find them will not. **W5 must cross-check stakes specifics against the world bible, source episodes relevant to the promo window, or an approved premise note.** If countables cannot be traced to source material, the flag fires and the beat reverts to STAKES-VAGUED (or the angle reworks to use show-grounded countables). | Pearl Crystal's parental-debt numbers are show-grounded (the source establishes the family's poverty and reliance on the academy). Conversely, a promo that invents a "500-crystal medicinal-herb fee" with no source warrant would fire STAKES-FABRICATED. |
| `INJUSTICE-LABELED-NOT-INFLICTED` | An INJUSTICE STRIKE beat relies on labels (pathetic, worthless, loser, freak) rather than specific cruelty modality (objects, numbers, named acts, physical detail). Fails Test 1 of antagonist cruelty calibration in `scene_exemplars.md`. | Chris Jones: "She kept the menu" — specific object doing the cruelty work. |
| `CLIFF-RESOLVES` | A beat tagged CLIFF lands with closure of the central question. Sister-flag to the RESOLUTION kill, but catches the *softer* version: the CLIFF that delivers emotional satisfaction even if the literal narrative is interrupted. | GU FLBM Trading Up's "Rita laughs — beasts are extraordinary" tags CLIFF structurally but the audience experiences closure; this flag fires; the CLIFF promotes to RESOLUTION → kill. |
| `PROSE-EDITORIALIZES` | The micro-beat's prose states its own meaning instead of rendering the event — thesis/aphorism tails ("being seen is the first step toward being discovered"), named emotion ("he had never felt more alone"), or a STATE/stakes explanation occupying the body of the beat in place of a filmable event. The structural reading belongs in the Pressure/Movement fields, not the prose. **Test:** delete the annotations — if the prose still tells you what the beat MEANS, the meaning has leaked in. | The board-glitch beat (W3 exemplar): prose renders the stutter and the "one hundred percent confidence" re-select; the meaning ("the system insists") lives only in Pressure/Movement. |
| `CANON-FABRICATED` | A beat invents a world-mechanic, device behavior, lore claim, or quantity (tenure, count, date, duration) that is not in the world bible — broader than `STAKES-FABRICATED` (which is scoped to STAKES-ANCHOR countables). The realization step is licensed to invent the *staging* of a move the skeleton already committed (the board glitch dramatizes "the system flags him"); it is NOT licensed to invent new *rules* (a wristwatch that stamps or color-codes squad placements), new *lore* ("a thousand-year war on vampires"), or new *quantities* ("the way it has run for two centuries"). **Test:** can each mechanic / lore claim / quantity be pointed to a bible line? If not it fires — re-ground it in a bible-true mechanic, tag it `INVENTION` for operator sign-off, or fall back to the committed move. The place to stop fabrication is HERE, upstream, before W4 blesses it as "in the sheet." | MVS: the wristwatch *reads and displays a power level* (bible) — dramatizing the draft as the watch chiming the level is grounded; inventing green/gold squad-stamps and "two centuries of clean drafts" is not. |
| `TERMINOLOGY-DRIFT` | A beat re-voices a locked canonical term instead of inheriting it verbatim — a synonym, nickname, or invented label standing in for the bible's system noun, in narration or in the system's own labels. **Test:** does every system noun match the world bible's Canonical Terms lock? In-character insults inside dialogue are exempt; narration and system labels are not. | MVS: levels are "Level 1–8 / Level One" and the watch displays a *number* — "one-star," "stars," or a star-stamped watch face is drift; "one-star runt" is allowed only as an occasional in-character jab, never as narration's default noun. |

Anti-pattern list grows over time. New entries require empirical evidence (a failure case from the audit data) before being added.

---

## 10. Maintenance

Single source of truth for the tag system. New subtypes require empirical evidence from at least one shipped script at target CPI before being added. Open questions raised during use should be appended below.
