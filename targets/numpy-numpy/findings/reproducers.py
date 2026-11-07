# Plain-Python triage reproducer for Finding 001 (property P3).
# No Hypothesis: shows the minimal non-associative triple, the pairwise
# steps that produce it, and exhaustive counts over the lab's dtype set.
# Run: ~/workspace/.venv-pbt/bin/python findings/reproducers.py
import itertools

import numpy as np

print("numpy", np.__version__)
print("docstring contract:", np.promote_types.__doc__.splitlines()[7].strip())

a, b, c = (np.dtype("int8"), np.dtype("uint8"), np.dtype("float16"))
print(f"\nminimal triple: ({a}, {b}, {c})")
print(f"  promote(promote({a},{b}) = {np.promote_types(a, b)}, {c})"
      f" = {np.promote_types(np.promote_types(a, b), c)}")
print(f"  promote({a}, promote({b},{c}) = {np.promote_types(b, c)})"
      f" = {np.promote_types(a, np.promote_types(b, c))}")

codes = ["?", "b", "B", "h", "H", "i", "I", "q", "Q", "e", "f", "d", "F", "D"]
dts = [np.dtype(x) for x in codes]
nonassoc = [(x, y, z) for x, y, z in itertools.product(dts, repeat=3)
            if np.promote_types(np.promote_types(x, y), z)
            != np.promote_types(x, np.promote_types(y, z))]
noncomm = [(x, y) for x, y in itertools.product(dts, repeat=2)
           if np.promote_types(x, y) != np.promote_types(y, x)]
print(f"\nexhaustive over {len(dts)} dtypes: "
      f"{len(nonassoc)} of {len(dts) ** 3} ordered triples non-associative; "
      f"{len(noncomm)} of {len(dts) ** 2} ordered pairs non-commutative")
print("first non-associative triples:")
for t in nonassoc[:5]:
    print("  ", t)
