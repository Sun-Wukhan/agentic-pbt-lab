# Finding: torch rejects *every* negative-step slice that NumPy evaluates — a deliberate, long-standing divergence, not a bug

- Target / property: pytorch/pytorch @ torch 2.15.0+cpu — property I3 as
  originally stated (`test_i3_basic_slicing_matches_numpy`): basic
  slicing along axis 0, start/stop/step drawn including negative steps,
  matches NumPy exactly. The indexing documentation presents slicing as
  following Python/NumPy semantics, which license negative steps.
- Minimal counterexample (shrunk by Hypothesis):
  - input: `torch.tensor([0])`, slice `slice(0, 0, -3)`
  - NumPy: returns an empty array. Torch: `ValueError: step must be
    greater than zero`.
  - Characterisation sweep (see `reproducers.py.txt`): 11 of 11
    negative-step slices diverge the same way, including the most basic
    reversal idiom there is — `t[::-1]` raises; NumPy returns
    `[4, 3, 2, 1, 0]` on `arange(5)`.
- Observed vs expected: this is not an edge case in an otherwise
  NumPy-compatible implementation; the rejection is *blanket*. Tensors
  do not support negative strides at all (the same wall appears in
  `torch.from_numpy`, which refuses negative-stride arrays), and
  negative-step slicing would need exactly that machinery.
- Verdict: **wrong property / intended behaviour** — a documented
  divergence, not a reportable bug. Upstream tracks it as a feature
  request (pytorch/pytorch#59786, "Support negative step sizes for
  slicing"), maintainers consistently direct users to `torch.flip`, and
  the error is raised deliberately by the indexing implementation.
  Filing it as a bug would misrepresent a known design boundary.
- Reasoning: the failure is still the most practically significant
  divergence this target found: code ported mechanically from NumPy
  that reverses with `[::-1]` breaks loudly at runtime rather than
  silently, which is the safe direction to fail — but it does break.
  The lab records it so the map is honest.
- Harness disposition: I3 now draws positive steps only (and passes at
  every check count); the divergence is pinned by characterisation test
  I3b (`test_i3b_negative_step_slicing_is_a_documented_divergence`),
  which asserts the rejection for 8 representative slices and that
  `torch.flip(t, [0])` equals NumPy's `a[::-1]`. If a future torch
  grows negative-step support, I3b fails loudly and this finding gets
  revisited.
- Upstream issue: none filed by us — pre-existing upstream feature
  request https://github.com/pytorch/pytorch/issues/59786
- Fix PR: none. Reproducer: `reproducers.py.txt`.
