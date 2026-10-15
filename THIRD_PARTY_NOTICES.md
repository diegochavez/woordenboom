# Third-party notices

## OpenTaal Dutch word list

Woordenboom includes data from the "Dutch word list" by OpenTaal.

- Creator: OpenTaal, https://opentaal.org
- Source: https://github.com/OpenTaal/opentaal-wordlist
- Version: 2.20.23
- Commit: `b250510dda431785f962019167d1415198ff3905`
- License selected by Woordenboom: Revised BSD License, also known as BSD-3-Clause

OpenTaal and its contributors do not endorse Woordenboom.

Copyright (c) 2020, OpenTaal
All rights reserved.

Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions are met:

1. Redistributions of source code must retain the above copyright notice,
   this list of conditions and the following disclaimer.
2. Redistributions in binary form must reproduce the above copyright notice,
   this list of conditions and the following disclaimer in the documentation
   and/or other materials provided with the distribution.
3. Neither the name of the copyright holder nor the names of its contributors
   may be used to endorse or promote products derived from this software
   without specific prior written permission.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS"
AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE
FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL
DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR
SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER
CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY,
OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

## wordfreq

Woordenboom may use frequency estimates generated with `wordfreq` for Dutch candidate ranking.

- Project: https://github.com/rspeer/wordfreq
- Author: Robyn Speer and contributors
- Code license: Apache-2.0
- Data notice: wordfreq data is compiled from multiple sources and includes Creative Commons Attribution-ShareAlike and other attributed sources. Preserve wordfreq notices when using generated frequency evidence.
- Citation: Robyn Speer. (2022). rspeer/wordfreq: v3.0. Zenodo. https://doi.org/10.5281/zenodo.7199437

## Dutch–English translations: WikDict / Wiktionary / DBnary

- Work: Nederlands-English FreeDict+WikDict dictionary, 40,061 headwords
- Contributors: Wiktionary contributors, DBnary, and WikDict maintainer Karl Bartel
- Version: 2025.11.21
- Download: https://download.wikdict.com/dictionaries/tei/recommended/nld-eng.tei
- Upstream: https://www.wikdict.com/ and https://nl.wiktionary.org/
- License stated in this pinned TEI file: CC BY-SA 3.0 Unported
- License: https://creativecommons.org/licenses/by-sa/3.0/legalcode
- Local source and license: `data/sources/wikdict/`
- Uncompressed SHA-256: `3468f7421840028431417c437acd02a38eb29e5fd1a69484573cc61c74d66313`

The pinned file’s license is used here even though the current WikDict website
links to a newer license. Woordenboom's release-9500 translation collection is
provided under CC BY-SA 3.0, including its editorial translation corrections and
included draft examples. This statement covers the dataset, not application code.
Preserve attribution, the license link, and notice of changes when sharing it.
No endorsement by the original contributors is implied.

Changes: matched exact Dutch lemmas to the OpenTaal candidate universe; excluded
proper-name, affix and abbreviation dictionary entries; selected 9,500 candidates
using wordfreq estimates while preserving 190 existing example pairs; shortened
translations to at most three distinct dictionary senses/synonyms; corrected 40
common translations; retained full selected senses and source-entry links for
review. These are draft learning translations, not a human-approved curriculum.

Each row in `data/content/release-9500/words.csv` identifies its translation source.
The dictionary URL identifies an entry for review, not a claim that editorial or
previously generated translations were copied from that page. Frequency evidence
continues to carry the wordfreq attribution above.

## Dutch definition overlay

`data/content/definitions-v1` adapts the Dutch definitions in the pinned WikDict
2025.11.21 source described above. Definitions are selected by English gloss,
limited to three per word, and supplemented with Woordenboom AI editorial drafts.
This adaptation is distributed under CC BY-SA 3.0. Each meaning retains source
and license metadata. See its README and editorial.json for modifications.

## Iconoir

`iconoir-react-native` 7.12.1, copyright 2021 Daniel Martin and Pascal Jufer. MIT license.
Used for mobile interface icons. Full notice: [Iconoir license](docs/licenses/iconoir-MIT.txt).
Source: https://github.com/iconoir-icons/iconoir
