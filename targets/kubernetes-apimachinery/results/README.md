# Results — kubernetes/apimachinery

Target commit: `b26d01364f13520dfdecb540fc63253be5b76662`
Framework: pgregory.net/rapid v1.2.0, Go (lab toolchain), runs on 2026-10-31.

- `run-100-checks.log` — full suite (24 properties, 4 packages) at
  `-rapid.checks=100`: all pass.
- `run-2000-checks-m2-failure.log` — the first 2,000-check run: 23/24
  pass; `TestPBTParseToLabelSelectorAgreesWithLabels` fails with the
  counterexample triaged as Finding 004.
- `run-2000-checks.log` — final 2,000-check run after the harness and
  property corrections recorded in `../findings/`: all 24 pass.

No compile isolation or other deviations were needed (unlike the
rclone chunker target): the four packages under test
(`pkg/api/resource`, `pkg/labels`, `pkg/util/intstr`,
`pkg/apis/meta/v1`) build and link quickly on their own.
