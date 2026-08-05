# Finish handoff

When the pipeline is done (W4 script assembled and saved), run the finish tool with the paths where you saved each artifact.

**Do not call any HTTP API.** Run `promo-finish` only.

## Inputs (key → file you wrote)

| Key | What you produced | Where you saved it |
|-----|-------------------|---------------------|
| `map` | W2 map | `working files/beat skeletons/[Series]_[Angle]_Map_v[X].md` |
| `briefs` | W3 briefs | `working files/beat sheets/[Series]_[Angle]_Briefs_v[X].md` |
| `script` | W4 script | `working files/w4 scripts/[Series]_[Angle]_v[X].md` |
| `receipt` | W4 receipts | `working files/documentation/[Series]_[Angle]_W4_Receipts_v[X].md` |

Use your actual filenames. Keys stay the same every run.

## Command

```bash
promo-finish finish \
  map="working files/beat skeletons/RBM_Disowned_Map_v1.md" \
  briefs="working files/beat sheets/RBM_Disowned_Briefs_v1.md" \
  script="working files/w4 scripts/RBM_Disowned_v1.md" \
  receipt="working files/documentation/RBM_Disowned_W4_Receipts_v1.md"
```

Stop when you see `PROMO_FINISH_OK`.
