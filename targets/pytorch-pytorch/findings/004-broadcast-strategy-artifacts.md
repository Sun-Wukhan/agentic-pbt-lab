# Finding: two harness artifacts in the lab's own broadcast strategies (0-d handling), found by B1/B3 before any real run

- Target / property: pytorch/pytorch @ torch 2.15.0+cpu — properties B1
  and B3, during harness bring-up (before the first clean full run).
- Counterexample 1 (B3): the strategy's pre-drawn "broadcast target"
  disagreed with the actual broadcast of the drawn side shapes —
  `(1,)` vs `()`. Cause: the strategy drew a target shape, then let
  *each side independently* drop leading dims; when both sides drop the
  same dim it vanishes from the true broadcast result, so the scaffolding
  target was simply wrong. Fix: the strategy now returns
  `np.broadcast_shapes(sa, sb)` of the actually drawn shapes.
- Counterexample 2 (B1/B2/B3): 0-d inputs kept changing shape or type
  between generation and comparison. Two stacked causes, both ours:
  `hypothesis.extra.numpy.arrays` with shape `()` yields a NumPy
  *scalar*, not a 0-d array; NumPy 2 returns scalars from 0-d ufunc
  results; and `np.ascontiguousarray` — the obvious normaliser —
  silently promotes 0-d arrays to shape `(1,)`. Fix: generated arrays
  are wrapped in `np.asarray`, and `to_torch` coerces with `np.asarray`
  and only calls `ascontiguousarray` on genuinely non-contiguous input.
- Verdict: **harness artifacts** (×2) — the properties, once fed the
  inputs they claimed to be fed, pass at every check count, including
  exact int/float broadcast value agreement with NumPy on 0-d, scalar-
  collapsed and dim-collapsed shapes.
- Reasoning: recorded per lab practice (cf. the rclone target's Finding
  004): 0-d is where array libraries' conventions quietly differ, and a
  PBT harness that mishandles it manufactures divergences that do not
  exist. Worth one paragraph of the public record so the next target's
  strategies start from the fixed versions.
- Upstream issue: none. Fix PR: none (lab-side fixes, in the harness).
