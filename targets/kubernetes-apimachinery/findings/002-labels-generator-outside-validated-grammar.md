# Finding: labels generator stepped outside the parser's validated grammar (negative values)

- Target / property: kubernetes/apimachinery @ `b26d01364f13520dfdecb540fc63253be5b76662` — properties L2 (`TestPBTSelectorStringRoundTrip`) and L3 (`TestPBTSelectorMatchesModel`), whose generators build supposedly-valid selector strings.
- Minimal counterexamples (generator output rejected by `labels.Parse`):
  - `app=-2` — "a valid label must be … start and end with an alphanumeric character"
  - `app > -1` — same validation applied to the Gt bound
- Observed vs expected: the harness assumed any integer-looking token is a legal label value; the parser validates every requirement value (including Gt/Lt bounds, via `NewRequirement`) against the label-value grammar, which requires values to start and end with an alphanumeric character. Negative numbers are therefore not expressible as selector values in this grammar at all, even though `Requirement.Matches` would compare them numerically if they could be constructed by hand.
- Verdict: **wrong property / harness artifact** — generated inputs were outside the grammar the properties claim to range over.
- Reasoning: parse-time validation is the documented, intended behaviour (the same `validation.LabelValue` rules used across the API). The generators now draw Gt/Lt bounds from [0, 10] and label values from the valid pool; both properties then pass at all check counts. One consequence worth recording: because negative bounds are unparseable, the Gt/Lt arm of the L3 model is exercised only with non-negative bounds and (possibly non-numeric) set values.
- Upstream issue: none (not a bug).
- Fix PR: none. Harness corrected in place; reproducer in `reproducers_test.go.txt`.
