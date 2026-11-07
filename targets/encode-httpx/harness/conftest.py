# Hypothesis profiles for the encode-httpx PBT lab.
#
#   default — 500 examples per property
#   long    — 2,000 examples per property (the confirmation run)
#
# Select with HYPOTHESIS_PROFILE=long (run.sh passes this through).
# database=None keeps runs hermetic: counterexamples are reported in the
# pytest output and transcribed into findings/, never replayed from a
# local example database.
import os

from hypothesis import settings

settings.register_profile(
    "default",
    max_examples=500,
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
