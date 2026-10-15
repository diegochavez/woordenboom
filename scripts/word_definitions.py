#!/usr/bin/env python3
"""Build, audit, validate and import sense-matched Dutch definitions. No network calls.

Original release files, word IDs, examples, progress, and saved session questions
are never rewritten. Output is an attributed, batch-scoped editorial overlay.
"""
import argparse
import collections
import csv
import hashlib
import io
import json
import re
import subprocess
from pathlib import Path
import vocabulary_release as release

ROOT = release.ROOT
DEST = ROOT / 'data/content/definitions-v1'
SOURCE = release.RELEASE / 'words.csv'
FIELDS = ['rank', 'lemma', 'translation', 'meanings_json', 'review_status', 'issues']

def digest(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()

def aliases(text):
    return {re.sub(r'^to ', '', s.strip().casefold()) for s in re.split(r';|\s+or\s+', text) if s.strip()}

def usable_definition(text):
    return bool(text.strip()) and len(text) <= 500 and not any(x in text for x in ('\\', 'Zie ook', '<', '>', 'http', '{', '}'))

def extract(row):
    """Match glosses, never join a homograph's unrelated sense by lemma alone."""
    senses = json.loads(row['senses_json'])
    meanings, issues, used = [], [], set()
    for gloss in re.split(r';|\s+or\s+', row['english_translation']):
        matching = [s for s in senses if any(aliases(gloss) & aliases(t) for t in s['translations'])]
        if not matching:
            issues.append('unmatched_translation')
        found = False
        for sense in matching:
            definition = next((release.normal(d) for d in sense['definitions'] if usable_definition(d)), None)
            if not definition:
                continue
            found = True
            if definition in used:
                continue
            used.add(definition)
            meanings.append({'english': gloss.strip(), 'dutch': definition,
                'partOfSpeech': sense['pos'], 'source': release.SOURCE_ID,
                'sourceUrl': row['dictionary_url'], 'license': release.LICENSE})
        if not found:
            issues.append('missing_matched_definition')
    if len(meanings) > 3:
        issues.append('additional_senses_not_shown')
    if len(senses) > 1:
        issues.append('multiple_senses_need_review')
    if any(not usable_definition(d) for s in senses for d in s['definitions']):
        issues.append('source_text_needs_cleanup')
    return meanings[:3], sorted(set(issues))

def build_rows(original, editorial):
    if set(editorial) - {r['lemma'] for r in original}:
        raise ValueError('Editorial lemma is absent from the release')
    rows = []
    for row in original:
        meanings, issues = extract(row)
        translation = row['english_translation']
        if row['lemma'] in editorial:
            edit = editorial[row['lemma']]
            if not edit['reason']:
                raise ValueError('Editorial reason required')
            meanings = [dict(m, source='woordenboom-definitions-editorial-v1',
                sourceUrl='data/content/definitions-v1/editorial.json', license=release.LICENSE) for m in edit['meanings']]
            translation = '; '.join(dict.fromkeys(m['english'] for m in meanings))
            issues = ['editorial_draft_needs_language_review']
        rows.append(dict(zip(FIELDS, [row['rank'], row['lemma'], translation,
            json.dumps(meanings, ensure_ascii=False, separators=(',', ':')), 'needs_content_review', ';'.join(issues)])))
    return rows

def validate(rows, original):
    if len(rows) != 9500 or [(r['rank'], r['lemma']) for r in rows] != [(r['rank'], r['lemma']) for r in original]:
        raise ValueError('Release membership/ranking changed')
    for row in rows:
        if not row['translation'].strip() or len(row['translation']) > 280 or row['review_status'] != 'needs_content_review':
            raise ValueError('Invalid translation or review status: ' + row['lemma'])
        meanings = json.loads(row['meanings_json'])
        if not isinstance(meanings, list) or len(meanings) > 3:
            raise ValueError('Invalid meanings: ' + row['lemma'])
        for meaning in meanings:
            if not usable_definition(meaning['dutch']) or not aliases(meaning['english']) <= aliases(row['translation']):
                raise ValueError('Definition/translation mismatch: ' + row['lemma'])
            if meaning['license'] != release.LICENSE or not meaning['sourceUrl'] or meaning['source'] not in {release.SOURCE_ID, 'woordenboom-definitions-editorial-v1'}:
                raise ValueError('Missing definition provenance')
        if any('\n' in str(v) or '\r' in str(v) or '\x00' in str(v) for v in row.values()):
            raise ValueError('Unexpected control character')

def quality(rows, original, editorial):
    return {'words': len(rows), 'with_dutch_definitions': sum(bool(json.loads(r['meanings_json'])) for r in rows),
        'without_matched_definition': sum(not json.loads(r['meanings_json']) for r in rows),
        'definition_pairs': sum(len(json.loads(r['meanings_json'])) for r in rows),
        'editorial_words': len(editorial),
        'changed_translations': sum(r['translation'] != o['english_translation'] for r, o in zip(rows, original)),
        'changed_translation_glosses': sum(aliases(r['translation']) != aliases(o['english_translation']) for r, o in zip(rows, original)),
        'first_300_with_definitions': sum(bool(json.loads(r['meanings_json'])) for r in rows[:300]),
        'authored_lessons_with_definitions': sum(bool(json.loads(r['meanings_json'])) for r, o in zip(rows, original) if o['source']=='woordenboom-generated-v1'),
        'issues': dict(sorted(collections.Counter(issue for r in rows for issue in r['issues'].split(';') if issue).items())),
        'review_status': 'needs_content_review', 'human_verified_words': 0}

def import_sql(rows, revision):
    if not re.fullmatch('[0-9a-f]{64}', revision):
        raise ValueError('Invalid revision')
    stream = io.StringIO()
    writer = csv.writer(stream, lineterminator='\n')
    writer.writerows([[r[f] for f in FIELDS] for r in rows])
    return rf"""\set ON_ERROR_STOP on
BEGIN;
SELECT pg_advisory_xact_lock(hashtext('woordenboom-vocabulary-import'));
CREATE TEMP TABLE definitions_incoming ({','.join(f'{f} text' for f in FIELDS)}) ON COMMIT DROP;
COPY definitions_incoming FROM STDIN WITH (FORMAT csv);
{stream.getvalue()}\.
DO $$ BEGIN
 IF (SELECT count(*) FROM definitions_incoming i JOIN words w ON w.lemma=i.lemma AND w.language_code='nl'
 JOIN vocabulary_batch_words bw ON bw.word_id=w.id AND bw.batch_rank=i.rank::int
 JOIN vocabulary_batches b ON b.id=bw.batch_id AND b.slug='release-9500') <> 9500
 THEN RAISE EXCEPTION 'Definition membership/ranking mismatch'; END IF;
END $$;
INSERT INTO word_learning_content(batch_id,word_id,translation,meanings,review_status,revision)
SELECT b.id,w.id,i.translation,i.meanings_json::jsonb,i.review_status,'{revision}'
FROM definitions_incoming i JOIN words w ON w.lemma=i.lemma AND w.language_code='nl'
JOIN vocabulary_batch_words bw ON bw.word_id=w.id
JOIN vocabulary_batches b ON b.id=bw.batch_id AND b.slug='release-9500'
ON CONFLICT(batch_id,word_id) DO UPDATE SET translation=EXCLUDED.translation,meanings=EXCLUDED.meanings,
 review_status=EXCLUDED.review_status,revision=EXCLUDED.revision,updated_at=now()
WHERE word_learning_content.revision<>EXCLUDED.revision AND word_learning_content.review_status<>'approved';
COMMIT;
"""

def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('command', choices=['build', 'validate', 'import'])
    args = parser.parse_args()
    original = release.read_csv(SOURCE)
    release.validate(original)
    if digest(SOURCE) != json.loads((release.RELEASE / 'manifest.json').read_text())['csv_sha256']:
        raise ValueError('Original release checksum mismatch')
    editorial = json.loads((DEST / 'editorial.json').read_text())
    expected = build_rows(original, editorial)
    validate(expected, original)
    if args.command == 'build':
        with (DEST / 'words.csv').open('w', newline='', encoding='utf-8') as f:
            writer = csv.DictWriter(f, fieldnames=FIELDS, lineterminator='\n')
            writer.writeheader(); writer.writerows(expected)
        report = quality(expected, original, editorial)
        report.update({'source_csv_sha256': digest(SOURCE), 'editorial_sha256': digest(DEST / 'editorial.json'),
            'csv_sha256': digest(DEST / 'words.csv'), 'dictionary_source': release.SOURCE_ID,
            'license': release.LICENSE, 'batch': 'release-9500'})
        (DEST / 'manifest.json').write_text(json.dumps(report, indent=2)+'\n')
        with (DEST / 'translation-changes.csv').open('w', newline='', encoding='utf-8') as f:
            writer = csv.writer(f, lineterminator='\n')
            writer.writerow(['rank', 'lemma', 'previous_translation', 'translation', 'change_type'])
            for row, old in zip(expected, original):
                if row['translation'] != old['english_translation']:
                    writer.writerow([row['rank'], row['lemma'], old['english_translation'], row['translation'],
                        'gloss_change' if aliases(row['translation']) != aliases(old['english_translation']) else 'formatting'])
        with (DEST / 'review-queue.csv').open('w', newline='', encoding='utf-8') as f:
            writer = csv.writer(f, lineterminator='\n')
            writer.writerow(['rank', 'lemma', 'translation', 'issues', 'dictionary_url'])
            for row, old in zip(expected, original):
                if row['issues']:
                    writer.writerow([row['rank'],row['lemma'],row['translation'],row['issues'],old['dictionary_url']])
    rows = release.read_csv(DEST / 'words.csv')
    manifest = json.loads((DEST / 'manifest.json').read_text())
    if rows != expected or manifest['csv_sha256'] != digest(DEST / 'words.csv') or manifest['editorial_sha256'] != digest(DEST / 'editorial.json') or manifest['source_csv_sha256'] != digest(SOURCE):
        raise ValueError('Content/checksum mismatch; rebuild the overlay')
    if args.command == 'import':
        subprocess.run(['docker','compose','exec','-T','postgres','psql','-X','-U','woordenboom','-d','woordenboom'],
            cwd=ROOT, input=import_sql(rows, manifest['csv_sha256']), text=True, check=True)
    print(json.dumps(manifest, indent=2))

if __name__ == '__main__':
    main()
