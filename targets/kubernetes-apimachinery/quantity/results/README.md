# Wave-2 Quantity runs — results

Harness: this directory's parent (`../run.sh [CHECKS]`), rapid v1.2.0,
separate module replacing `k8s.io/apimachinery` with `../../apimachinery`
(`pkg/api/resource` byte-identical to pin `b26d0136…`; clone unmodified).

| Log | Checks per property | Outcome |
|---|---|---|
| `run-100-checks.log` | 100 | 16/17 properties pass; `TestPBTQuantityExponentRange` FAILS (finding 001) |
| `run-2000-checks.log` | 2,000 | identical: 16/17 pass; R15 fails with shrunk input `1e4294967296` |
| `repro-001.log` | directed | `TestReproExponentWraparound` fails on all 7 wrapped-exponent cases; boundary controls (MaxInt32 parses; MinInt32-band rejects) pass |

Totals: 17 properties × 2,100 rapid checks each ≈ 35,700 generated cases,
plus the directed reproducer. Two further raw failures during bring-up
were triaged to documented behaviour and the properties corrected with
explicit preconditions: the parse nano floor (finding 002) and the
BinarySI parse cap / format loss (finding 003). One confirmed bug:
finding 001.
