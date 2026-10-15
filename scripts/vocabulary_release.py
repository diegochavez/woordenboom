#!/usr/bin/env python3
"""Build, validate, and transactionally import the pinned Dutch-English release.

Uses only Python's standard library. Import targets local Docker Compose Postgres.
No network calls, API keys, generated examples, or modifications to older batches.
"""
import argparse
import collections
import csv
import gzip
import hashlib
import io
import json
from pathlib import Path
import subprocess
import unicodedata
import urllib.parse
import xml.etree.ElementTree as ET

ROOT = Path(__file__).resolve().parents[1]
RELEASE = ROOT / 'data/content/release-9500'
SOURCE = ROOT / 'data/sources/wikdict/nld-eng-2025.11.21.tei.gz'
SOURCE_SHA = '3468f7421840028431417c437acd02a38eb29e5fd1a69484573cc61c74d66313'
SOURCE_URL = 'https://download.wikdict.com/dictionaries/tei/recommended/nld-eng.tei'
SOURCE_ID = 'wikdict-nld-eng-2025.11.21'
LICENSE = 'CC-BY-SA-3.0'
NS = {'t': 'http://www.tei-c.org/ns/1.0'}
FIELDS = ['rank', 'lemma', 'english_translation', 'topic', 'part_of_speech',
          'frequency_rank', 'zipf', 'entry_type', 'needs_topic_review', 'source',
          'source_url', 'dictionary_url', 'license', 'dutch_example', 'english_example', 'senses_json']


def normal(text):
    return unicodedata.normalize('NFC', ' '.join(text.split()))


def read_csv(path):
    with open(path, newline='', encoding='utf-8') as f:
        return list(csv.DictReader(f))


def dictionary():
    raw = gzip.decompress(SOURCE.read_bytes())
    if hashlib.sha256(raw).hexdigest() != SOURCE_SHA:
        raise ValueError('Dictionary checksum mismatch')
    result = collections.defaultdict(list)
    for entry in ET.fromstring(raw).findall('.//t:entry', NS):
        lemma = normal(entry.findtext('t:form/t:orth', default='', namespaces=NS))
        pos = entry.findtext('t:gramGrp/t:pos', default='', namespaces=NS)
        if pos in {'pn', 'prefix', 'suffix', 'abbreviation'}:
            continue
        for sense in entry.findall('t:sense', NS):
            translations = list(dict.fromkeys(normal(''.join(q.itertext())) for q in
                                sense.findall('t:cit[@type="trans"]/t:quote', NS)))
            translations = [s for s in translations if s and len(s) <= 90]
            if translations:
                result[lemma].append({'pos': pos, 'translations': translations,
                    'definitions': [normal(''.join(d.itertext())) for d in sense.findall('.//t:def', NS)]})
    return result


def display_translation(senses):
    # Represent separate senses before adding synonyms of the first sense.
    options = [s['translations'][0] for s in senses]
    options += [t for s in senses for t in s['translations'][1:]]
    return '; '.join(list(dict.fromkeys(options))[:3])


def build(candidates_path):
    entries = dictionary()
    prior = {r['lemma']: r for r in read_csv(ROOT / 'data/exports/release-2000/all.csv')}
    authored = {k: v for k, v in prior.items() if v['english_translation']}
    overrides = {r['lemma']: r for r in read_csv(RELEASE / 'overrides.csv')}
    candidates = {r['lemma']: r for r in read_csv(candidates_path)
                  if r['entry_type'] in {'base_word', 'compound', 'multiword'}
                  and r['is_learning_candidate'] == 't'}
    if not authored.keys() <= candidates.keys():
        raise ValueError('Candidate snapshot is missing previously authored words')
    eligible = [r for r in candidates.values() if r['lemma'] in entries or r['lemma'] in authored]
    eligible.sort(key=lambda r: (-float(r['zipf_score']), r['lemma']))
    chosen = set(authored)
    for r in eligible:
        if len(chosen) >= 9500:
            break
        chosen.add(r['lemma'])
    selected = sorted((candidates[k] for k in chosen), key=lambda r: (-float(r['zipf_score']), r['lemma']))
    rows = []
    for rank, candidate in enumerate(selected, 1):
        lemma = candidate['lemma']
        old = prior.get(lemma, {})
        content = authored.get(lemma, {})
        senses = entries.get(lemma, [])
        source = 'woordenboom-generated-v1' if content else SOURCE_ID
        translation = content.get('english_translation') or display_translation(senses)
        if lemma in overrides and not content:
            translation = overrides[lemma]['english_translation']
            source = 'woordenboom-editorial-v1'
        # Keep prior assignments; unclassified vocabulary has an honest general category.
        labels = candidate['topic_labels'].strip('{}').split(',')
        topic = old.get('topic') or next((t for t in labels if t and t != 'daily-basics'), 'general-vocabulary')
        rows.append(dict(zip(FIELDS, [rank, lemma, translation, topic,
            '; '.join(dict.fromkeys(s['pos'] for s in senses if s['pos'])),
            candidate['frequency_rank'], candidate['zipf_score'], candidate['entry_type'],
            'true' if topic == 'general-vocabulary' or old.get('needs_topic_review') == 'true' else 'false',
            source, SOURCE_URL if source == SOURCE_ID else ('data/content/release-2000/' if content else 'data/content/release-9500/overrides.csv'),
            'https://nl.wiktionary.org/wiki/' + urllib.parse.quote(lemma, safe=''), LICENSE,
            content.get('dutch_example', ''), content.get('english_example', ''),
            json.dumps(senses, ensure_ascii=False, separators=(',', ':'))])))
    validate(rows)
    with open(RELEASE / 'words.csv', 'w', newline='', encoding='utf-8') as f:
        writer = csv.DictWriter(f, fieldnames=FIELDS, lineterminator='\n')
        writer.writeheader()
        writer.writerows(rows)
    manifest = quality(rows)
    manifest.update({'batch': 'release-9500', 'dictionary_version': '2025.11.21',
        'dictionary_url': SOURCE_URL, 'dictionary_sha256': SOURCE_SHA,
        'csv_sha256': hashlib.sha256((RELEASE / 'words.csv').read_bytes()).hexdigest(),
        'license': LICENSE, 'review_status': 'needs_content_review',
        'eligible_dictionary_candidates': len(eligible),
        'candidate_snapshot_sha256': hashlib.sha256(Path(candidates_path).read_bytes()).hexdigest()})
    (RELEASE / 'manifest.json').write_text(json.dumps(manifest, indent=2) + '\n')
    print(json.dumps(manifest, indent=2))


def validate(rows):
    if len(rows) != 9500:
        raise ValueError(f'Expected 9500 words, found {len(rows)}')
    lemmas = set()
    topics = {r['topic'] for r in read_csv(ROOT / 'data/exports/release-2000/all.csv') if r['topic']}
    topics.add('general-vocabulary')
    for rank, row in enumerate(rows, 1):
        lemma = row['lemma']
        if int(row['rank']) != rank or not lemma or lemma != normal(lemma) or lemma.casefold() in lemmas:
            raise ValueError(f'Duplicate, non-normalized lemma, or invalid rank at {rank}: {lemma}')
        lemmas.add(lemma.casefold())
        if not row['english_translation'].strip() or len(row['english_translation']) > 280:
            raise ValueError(f'Missing or excessive translation: {lemma}')
        if row['topic'] not in topics or row['needs_topic_review'] not in {'true', 'false'}:
            raise ValueError(f'Invalid topic metadata: {lemma}')
        if row['source'] not in {SOURCE_ID, 'woordenboom-generated-v1', 'woordenboom-editorial-v1'} or row['license'] != LICENSE:
            raise ValueError(f'Invalid provenance: {lemma}')
        if bool(row['dutch_example']) != bool(row['english_example']):
            raise ValueError(f'Incomplete example pair: {lemma}')
        if row['source'] == SOURCE_ID and not json.loads(row['senses_json']):
            raise ValueError(f'Missing dictionary senses: {lemma}')
        if float(row['zipf']) <= 0 or int(row['frequency_rank']) <= 0:
            raise ValueError(f'Invalid frequency evidence: {lemma}')
        for value in row.values():
            if any(c in str(value) for c in ('\x00', '\r', '\n')):
                raise ValueError(f'Unexpected control character: {lemma}')


def quality(rows):
    return {'words': len(rows), 'unique_lemmas': len({r['lemma'] for r in rows}),
        'english_translations': sum(bool(r['english_translation']) for r in rows),
        'example_pairs': sum(bool(r['dutch_example'] and r['english_example']) for r in rows),
        'needs_topic_review': sum(r['needs_topic_review'] == 'true' for r in rows),
        'min_zipf': min(float(r['zipf']) for r in rows),
        'sources': dict(collections.Counter(r['source'] for r in rows)),
        'topics': dict(sorted(collections.Counter(r['topic'] for r in rows).items())),
        'multiple_dictionary_senses': sum(len(json.loads(r['senses_json'])) > 1 for r in rows)}


def import_sql(rows, checksum):
    staging = io.StringIO()
    writer = csv.writer(staging, lineterminator='\n')
    for row in rows:
        writer.writerow([row[f] for f in FIELDS])
    # Data goes through COPY, never through SQL interpolation. Checksum is a hex digest.
    schema = ', '.join(f'{name} text' for name in FIELDS)
    sql = rf"""\set ON_ERROR_STOP on
BEGIN;
SELECT pg_advisory_xact_lock(hashtext('woordenboom-vocabulary-import'));
CREATE TEMP TABLE incoming ({schema}) ON COMMIT DROP;
COPY incoming FROM STDIN WITH (FORMAT csv);
{staging.getvalue()}\.
DO $$ BEGIN
 IF (SELECT count(*) FROM incoming i JOIN words w ON w.lemma=i.lemma AND w.language_code='nl') <> 9500
 THEN RAISE EXCEPTION 'All 9500 lemmas must exist in the Dutch word universe'; END IF;
 IF EXISTS (SELECT 1 FROM vocabulary_batches WHERE slug='release-9500' AND notes <> 'CSV SHA256 {checksum}')
 THEN RAISE EXCEPTION 'Existing release differs: create a new version instead of overwriting it'; END IF;
END $$;
INSERT INTO lexical_sources (id,title,creator,source_url,version,license,attribution,requires_attribution,share_alike_risk,notes)
VALUES ('{SOURCE_ID}','Dutch-English FreeDict+WikDict dictionary','Wiktionary contributors; Karl Bartel; DBnary',
'{SOURCE_URL}','2025.11.21','CC-BY-SA-3.0','Wiktionary via DBnary and WikDict, maintained by Karl Bartel',true,true,
'License from the pinned TEI header. Selected, shortened and frequency-ranked by Woordenboom. Draft senses need human review.'),
('woordenboom-editorial-v1','Woordenboom translation corrections','Woordenboom', '', '1','CC-BY-SA-3.0',
'Woordenboom corrections to common dictionary senses',true,true,'Draft content; see overrides.csv')
ON CONFLICT (id) DO NOTHING;
INSERT INTO vocabulary_batches (slug,title,target_size,status,notes)
VALUES ('release-9500','9,500 Dutch words',9500,'reviewing','CSV SHA256 {checksum}') ON CONFLICT (slug) DO NOTHING;
INSERT INTO word_translations (word_id,language_code,translation,sort_order,source)
SELECT w.id,'en',i.english_translation,0,i.source FROM incoming i JOIN words w ON w.lemma=i.lemma
WHERE NOT EXISTS (SELECT 1 FROM word_translations t WHERE t.word_id=w.id AND t.language_code='en' AND t.source=i.source);
INSERT INTO word_examples (word_id,dutch_sentence,english_sentence,source,sort_order)
SELECT w.id,i.dutch_example,i.english_example,i.source,0 FROM incoming i JOIN words w ON w.lemma=i.lemma
WHERE i.dutch_example IS NOT NULL AND i.dutch_example<>''
AND NOT EXISTS (SELECT 1 FROM word_examples e WHERE e.word_id=w.id AND e.source=i.source);
INSERT INTO word_dictionary_entries (word_id,source_id,source_url,part_of_speech,senses)
SELECT w.id,'{SOURCE_ID}',i.dictionary_url,COALESCE(i.part_of_speech,''),i.senses_json::jsonb
FROM incoming i JOIN words w ON w.lemma=i.lemma WHERE i.senses_json::jsonb <> '[]'::jsonb
ON CONFLICT (word_id,source_id) DO NOTHING;
INSERT INTO vocabulary_batch_words (batch_id,word_id,batch_rank,topic_slug,topic_rank,candidate_score,frequency_rank,zipf_score,entry_type,review_status,needs_topic_review,notes)
SELECT b.id,w.id,i.rank::int,i.topic,row_number() OVER (PARTITION BY i.topic ORDER BY i.rank::int),
0,i.frequency_rank::int,i.zipf::float,i.entry_type,'needs_content_review',i.needs_topic_review::boolean,
'English translation available. CEFR and dictionary senses not human-verified; examples optional.'
FROM incoming i JOIN words w ON w.lemma=i.lemma CROSS JOIN vocabulary_batches b WHERE b.slug='release-9500'
ON CONFLICT (batch_id,word_id) DO NOTHING;
DO $$ BEGIN
 IF (SELECT count(*) FROM vocabulary_batch_words bw JOIN vocabulary_batches b ON b.id=bw.batch_id WHERE b.slug='release-9500') <> 9500
 THEN RAISE EXCEPTION 'Release does not contain exactly 9500 words'; END IF;
 IF EXISTS (SELECT 1 FROM incoming i JOIN words w ON w.lemma=i.lemma WHERE NOT EXISTS (
 SELECT 1 FROM word_translations t WHERE t.word_id=w.id AND t.language_code='en' AND btrim(t.translation)<>''))
 THEN RAISE EXCEPTION 'Missing English translations'; END IF;
END $$;
COMMIT;
"""
    return sql


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('command', choices=['build', 'validate', 'import'])
    parser.add_argument('--candidates', help='CSV exported from candidate_word_scores; only needed for build')
    args = parser.parse_args()
    if args.command == 'build':
        if not args.candidates:
            parser.error('--candidates is required to build')
        build(args.candidates)
        return
    rows = read_csv(RELEASE / 'words.csv')
    validate(rows)
    manifest = json.loads((RELEASE / 'manifest.json').read_text())
    checksum = hashlib.sha256((RELEASE / 'words.csv').read_bytes()).hexdigest()
    if checksum != manifest['csv_sha256']:
        raise ValueError('Release checksum mismatch; rebuild the manifest before importing changes')
    if args.command == 'import':
        subprocess.run(['docker', 'compose', 'exec', '-T', 'postgres', 'psql', '-X',
                        '-U', 'woordenboom', '-d', 'woordenboom'], cwd=ROOT,
                       input=import_sql(rows, checksum), text=True, check=True)
    print(json.dumps(quality(rows), indent=2))


if __name__ == '__main__':
    main()
