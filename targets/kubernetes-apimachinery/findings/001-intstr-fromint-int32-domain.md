# Finding: `intstr` passthrough property initially ranged outside the type's documented int32 domain

- Target / property: kubernetes/apimachinery @ `b26d01364f13520dfdecb540fc63253be5b76662` — property I2 (`TestPBTIntstrIntPassthrough`): a non-percent `IntOrString` is returned unchanged by `GetScaledValueFromIntOrPercent`.
- Minimal counterexample: `FromInt(2147483648)` (2^31) → `GetScaledValueFromIntOrPercent` returns `-2147483648`.
- Observed vs expected: the harness drew full 64-bit `int` values, but `IntOrString.IntVal` is an `int32` field. `FromInt`'s doc comment states the contract explicitly: "creates an IntOrString object with an int32 value. It is your responsibility not to call this method with a value greater than int32." Out-of-range input is logged via klog (with a stack trace, which is what surfaced in the test output) and silently truncated by the `int32(val)` conversion; `FromInt32` exists as the checked-domain constructor.
- Verdict: **wrong property / harness artifact** — the property's domain was ours to state correctly, not upstream's defect.
- Reasoning: the truncation is documented behaviour of a deprecated convenience constructor, and the value cannot be represented in the type at all. Restricting I2 (and I1's int draws) to the int32 domain makes the property pass at all check counts; the passthrough contract itself held everywhere inside the domain.
- Upstream issue: none (not a bug).
- Fix PR: none. Harness corrected in place; reproducer in `reproducers_test.go.txt`.
