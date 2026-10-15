#!/usr/bin/env python3
"""Export Dutch wordfreq Zipf scores for lemmas read from stdin.

Usage:
  docker compose exec -T postgres psql -U woordenboom -d woordenboom -tA -c "SELECT lemma FROM words" \
    | python3 scripts/export-wordfreq-nl.py > data/wordfreq-nl.tsv

Requires:
  pip install wordfreq
"""

import sys

from wordfreq import zipf_frequency


def main() -> None:
    seen: set[str] = set()
    for raw in sys.stdin:
        lemma = raw.strip()
        if not lemma or lemma in seen:
            continue
        seen.add(lemma)
        score = zipf_frequency(lemma, "nl", wordlist="best")
        if score > 0:
            print(f"{lemma}\t{score:.2f}")


if __name__ == "__main__":
    main()
