import copy
import json
import unittest
import word_definitions as definitions

class DefinitionTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.original = definitions.release.read_csv(definitions.SOURCE)
        cls.editorial = json.loads((definitions.DEST / 'editorial.json').read_text())
        cls.rows = definitions.build_rows(cls.original, cls.editorial)

    def test_release_coverage_and_deterministic_build(self):
        definitions.validate(self.rows, self.original)
        self.assertEqual(self.rows, definitions.release.read_csv(definitions.DEST / 'words.csv'))
        self.assertGreaterEqual(sum(bool(json.loads(r['meanings_json'])) for r in self.rows), 7600)

    def test_frequent_and_authored_lessons_have_definitions(self):
        self.assertTrue(all(json.loads(r['meanings_json']) for r in self.rows[:300]))
        for row, old in zip(self.rows, self.original):
            if old['source'] == 'woordenboom-generated-v1':
                self.assertTrue(json.loads(row['meanings_json']), row['lemma'])

    def test_does_not_borrow_a_nonmatching_gloss(self):
        raw = {r['lemma']: r for r in self.original}
        for lemma in ['bij', 'was', 'kom', 'mee']:
            with self.subTest(lemma=lemma):
                meanings, issues = definitions.extract(raw[lemma])
                self.assertEqual(meanings, [])
                self.assertIn('unmatched_translation', issues)
        actual = {r['lemma']: r for r in self.rows}
        for lemma, expected in [('bij', 'at; near'), ('kan', 'can'), ('woord', 'word'), ('kom', 'come')]:
            self.assertEqual(actual[lemma]['translation'], expected)
            self.assertTrue(json.loads(actual[lemma]['meanings_json']))

    def test_multiple_meanings_stay_paired_and_in_order(self):
        row = next(r for r in self.rows if r['lemma'] == 'leren')
        meanings = json.loads(row['meanings_json'])
        self.assertEqual([m['english'] for m in meanings], ['learn', 'teach'])
        self.assertTrue(meanings[1]['dutch'].startswith('Iemand helpen'))
        self.assertNotEqual(meanings[0]['dutch'], meanings[1]['dutch'])

    def test_rejects_missing_provenance_wrong_gloss_or_markup(self):
        for key, value in [('english', 'unrelated'), ('license', ''), ('dutch', '<script>')]:
            rows = copy.deepcopy(self.rows)
            meanings = json.loads(rows[0]['meanings_json'])
            meanings[0][key] = value
            rows[0]['meanings_json'] = json.dumps(meanings)
            with self.subTest(key=key), self.assertRaises(ValueError):
                definitions.validate(rows, self.original)
        with self.assertRaises(ValueError):
            definitions.validate(self.rows[:-1], self.original)

    def test_retains_unmatched_words_without_fabricated_definitions(self):
        row = dict(self.original[0], english_translation='unknown sense', senses_json='[]')
        meanings, issues = definitions.extract(row)
        self.assertEqual(meanings, [])
        self.assertIn('missing_matched_definition', issues)

if __name__ == '__main__':
    unittest.main()
