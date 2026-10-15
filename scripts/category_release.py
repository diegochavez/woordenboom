#!/usr/bin/env python3
"""Build reproducible, sense-specific category memberships without changing the corpus.

Rules match an exact English gloss or a Dutch dictionary domain on the SAME
meaning. Ambiguous glosses require a domain. No confidence score or human approval
is invented. Imports replace only draft assignments from this classifier.
"""
import argparse, collections, csv, hashlib, io, json, re, subprocess
from pathlib import Path
import vocabulary_release as release

ROOT = Path(__file__).resolve().parents[1]
DEST = ROOT / 'data/content/categories-v1'
SOURCE = ROOT / 'data/content/definitions-v1/words.csv'
FIELDS = ['rank', 'lemma', 'topic', 'translation', 'meanings_json', 'evidence', 'review_status']

def digest(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()

def classify(row, original, rules):
    memberships = []
    if row['lemma'] in rules.get('excluded_lemmas', []):
        return memberships
    all_domains = set(rules['context_domains']) | {d for r in rules['topics'].values() for d in r['domains']}
    ambiguous = set(rules['ambiguous_glosses'])
    for topic, rule in rules['topics'].items():
        if topic == original['topic']:
            continue
        matched, evidence = [], []
        for meaning in json.loads(row['meanings_json']):
            glosses = {g.strip().lower() for g in meaning['english'].split(';')}
            domains = set(re.findall(r'\(([^()]+)\)', meaning['dutch'].lower()))
            # Figurative senses do not inherit a literal topic from their gloss.
            if domains.intersection(rules['excluded_domains']) or '...' in meaning['dutch'] or '…' in meaning['dutch']:
                continue
            domain_hits = domains.intersection(rule['domains'])
            gloss_hits = glosses.intersection(rule['glosses'].lower().split('|')) - ambiguous
            if domains.intersection(all_domains) and not domain_hits:
                gloss_hits = set()
            if meaning['partOfSpeech'] not in {'n','noun','v','verb','adj','adjective','interjection'}:
                gloss_hits = set()
            if not domain_hits and not gloss_hits:
                continue
            matched.append(meaning)
            evidence.extend('domain:' + d for d in sorted(domain_hits))
            evidence.extend('gloss:' + g for g in sorted(gloss_hits))
        if matched:
            memberships.append(dict(rank=row['rank'], lemma=row['lemma'], topic=topic,
                translation='; '.join(dict.fromkeys(m['english'] for m in matched)),
                meanings_json=json.dumps(matched, ensure_ascii=False, separators=(',', ':')),
                evidence='; '.join(dict.fromkeys(evidence)), review_status='needs_content_review'))
    limit = 3 if original['topic'] == 'general-vocabulary' else 2
    return memberships[:limit]

def pilot(rows, originals, rules):
    """500 distinct words: frequent, ambiguous, and frequency/POS strata."""
    selected = {r['lemma']: r for r, o in zip(rows[:200], originals[:200]) if o['topic'] == 'general-vocabulary'}
    ambiguous = set(rules['ambiguous_glosses'])
    candidates = [r for r in rows if len(json.loads(r['meanings_json'])) > 1 or set(r['translation'].lower().split('; ')) & ambiguous]
    for r in sorted(candidates, key=lambda r: hashlib.sha256(r['lemma'].encode()).hexdigest())[:125]:
        selected[r['lemma']] = r
    strata = collections.defaultdict(list)
    for r, o in zip(rows, originals):
        if r['lemma'] not in selected:
            strata[(int(r['rank']) // 1000, o['part_of_speech'])].append(r)
    for group in strata.values():
        group.sort(key=lambda r: hashlib.sha256(r['lemma'].encode()).hexdigest())
    while len(selected) < 500:
        for key in sorted(strata):
            if strata[key] and len(selected) < 500:
                r = strata[key].pop(0); selected[r['lemma']] = r
    return sorted(selected.values(), key=lambda r: int(r['rank']))

def write_csv(path, fields, rows):
    with path.open('w', newline='', encoding='utf-8') as f:
        writer = csv.DictWriter(f, fieldnames=fields, lineterminator='\n')
        writer.writeheader(); writer.writerows(rows)

def build():
    originals = release.read_csv(release.RELEASE / 'words.csv')
    rows = release.read_csv(SOURCE)
    rules = json.loads((DEST/'rules.json').read_text())
    if digest(SOURCE) != json.loads((SOURCE.parent/'manifest.json').read_text())['csv_sha256']:
        raise ValueError('Definition overlay checksum mismatch')
    if digest(release.RELEASE/'words.csv') != json.loads((release.RELEASE/'manifest.json').read_text())['csv_sha256']:
        raise ValueError('Original corpus checksum mismatch')
    assert len(rows) == len(originals) == 9500
    assert [(r['rank'], r['lemma']) for r in rows] == [(r['rank'], r['lemma']) for r in originals]
    by_lemma = {o['lemma']: o for o in originals}
    assignments = [m for r in rows for m in classify(r, by_lemma[r['lemma']], rules)]
    return originals, rows, rules, assignments

def import_sql(assignments, revision, content_revision):
    stream = io.StringIO(); writer = csv.writer(stream, lineterminator='\n')
    writer.writerows([[r[f] for f in FIELDS] for r in assignments])
    return rf"""\set ON_ERROR_STOP on
BEGIN;
SELECT pg_advisory_xact_lock(hashtext('woordenboom-vocabulary-import'));
CREATE TEMP TABLE category_incoming ({','.join(f'{f} text' for f in FIELDS)}) ON COMMIT DROP;
COPY category_incoming FROM STDIN WITH (FORMAT csv);
{stream.getvalue()}\.
DO $$ BEGIN
 IF (SELECT count(*) FROM category_incoming i JOIN words w ON w.lemma=i.lemma AND w.language_code='nl'
 JOIN vocabulary_batch_words bw ON bw.word_id=w.id AND bw.batch_rank=i.rank::int
 JOIN vocabulary_batches b ON b.id=bw.batch_id AND b.slug='release-9500'
 JOIN word_learning_content lc ON lc.batch_id=b.id AND lc.word_id=w.id AND lc.revision='{content_revision}')
 <> (SELECT count(*) FROM category_incoming) THEN RAISE EXCEPTION 'Category content revision/membership mismatch'; END IF;
END $$;
DELETE FROM vocabulary_word_topics m USING vocabulary_batches b WHERE b.id=m.batch_id AND b.slug='release-9500'
 AND m.source='category-senses-v1' AND m.review_status<>'approved' AND NOT EXISTS
 (SELECT 1 FROM category_incoming i JOIN words w ON w.lemma=i.lemma AND w.language_code='nl' WHERE w.id=m.word_id AND i.topic=m.topic_slug);
INSERT INTO topics(slug,title) SELECT DISTINCT i.topic,initcap(replace(i.topic,'-',' ')) FROM category_incoming i
ON CONFLICT(slug) DO NOTHING;
INSERT INTO vocabulary_word_topics(batch_id,word_id,topic_slug,translation,meanings,evidence,source,review_status,revision,content_revision)
SELECT b.id,w.id,i.topic,i.translation,i.meanings_json::jsonb,i.evidence,'category-senses-v1',i.review_status,'{revision}','{content_revision}'
FROM category_incoming i JOIN words w ON w.lemma=i.lemma AND w.language_code='nl'
JOIN vocabulary_batch_words bw ON bw.word_id=w.id JOIN vocabulary_batches b ON b.id=bw.batch_id AND b.slug='release-9500'
ON CONFLICT(batch_id,word_id,topic_slug) DO UPDATE SET translation=EXCLUDED.translation,meanings=EXCLUDED.meanings,
 evidence=EXCLUDED.evidence,revision=EXCLUDED.revision,content_revision=EXCLUDED.content_revision,updated_at=now()
WHERE vocabulary_word_topics.source='category-senses-v1' AND vocabulary_word_topics.review_status<>'approved'
 AND (vocabulary_word_topics.revision<>EXCLUDED.revision OR vocabulary_word_topics.content_revision<>EXCLUDED.content_revision);
COMMIT;
"""

def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('command', choices=['pilot', 'build', 'validate', 'import'])
    args = parser.parse_args()
    original, rows, rules, assignments = build()
    if args.command == 'pilot':
        sample = pilot(rows, original, rules)
        mapped = collections.defaultdict(list)
        for a in assignments: mapped[a['lemma']].append(a)
        write_csv(DEST/'pilot.csv', ['rank','lemma','translation','meanings_json','proposed_topics','evidence'],
            [dict(rank=r['rank'], lemma=r['lemma'], translation=r['translation'], meanings_json=r['meanings_json'],
            proposed_topics=';'.join(a['topic'] for a in mapped[r['lemma']]), evidence=' | '.join(a['evidence'] for a in mapped[r['lemma']])) for r in sample])
        print(json.dumps({'pilot_words':len(sample),'with_proposals':sum(bool(mapped[r['lemma']]) for r in sample)})); return
    before = collections.Counter(r['topic'] for r in original)
    members = {(r['lemma'],r['topic']) for r in original if r['topic']!='general-vocabulary'} | {(a['lemma'],a['topic']) for a in assignments}
    after = collections.Counter(t for _,t in members); after['general-vocabulary'] = 9500
    report = dict(batch='release-9500', classifier=rules['version'], new_memberships=len(assignments),
        specifically_categorized_words=len({w for w,_ in members}), general_only_words=9500-len({w for w,_ in members}),
        before=dict(before), after=dict(sorted(after.items())), source_sha256=digest(SOURCE), rules_sha256=digest(DEST/'rules.json'),
        review_status='needs_content_review', human_verified_assignments=0)
    if args.command=='build':
        write_csv(DEST/'assignments.csv', FIELDS, assignments)
        report['csv_sha256']=digest(DEST/'assignments.csv')
        (DEST/'manifest.json').write_text(json.dumps(report,indent=2)+'\n')
    actual = release.read_csv(DEST/'assignments.csv'); manifest=json.loads((DEST/'manifest.json').read_text())
    report['csv_sha256']=digest(DEST/'assignments.csv')
    if actual!=assignments or report!=manifest: raise ValueError('Rebuild category assignments: data or checksum mismatch')
    if args.command=='import':
        subprocess.run(['docker','compose','exec','-T','postgres','psql','-X','-U','woordenboom','-d','woordenboom'], cwd=ROOT,
            input=import_sql(assignments,report['csv_sha256'],report['source_sha256']),text=True,check=True)
    print(json.dumps(report,indent=2))

if __name__=='__main__': main()
