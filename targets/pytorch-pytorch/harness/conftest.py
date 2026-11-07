# Hypothesis profiles for the pytorch-pytorch PBT lab.
#
#   default — 200 examples per property (first pass, mirrors the lab's
#             "default checks" runs in other targets)
#   long    — 2,000 examples per property (the confirmation run)
#
# Select with HYPOTHESIS_PROFILE=long (run.sh passes this through).
# database=None keeps runs hermetic: counterexamples are reported in the
# pytest output and transcribed into findings/, never replayed from a
# local example database.
import os
import warnings

import numpy as np  # noqa: F401  (imported so overflow warnings exist as a class)
from hypothesis import settings

# NumPy integer overflow wraps with a RuntimeWarning; torch wraps
# silently. The wrap VALUES are what the properties compare.
warnings.filterwarnings("ignore", category=RuntimeWarning)

settings.register_profile(
    "default",
    max_examples=200,
    deadline=None,
    database=None,
)
settings.register_profile(
    "long",
    max_examples=2000,
    deadline=None,
    database=None,
)
settings.load_profile(os.environ.get("HYPOTHESIS_PROFILE", "default"))
