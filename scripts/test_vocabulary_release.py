import copy
import unittest
import vocabulary_release as release


class ReleaseTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.rows = release.read_csv(release.RELEASE / 'words.csv')

    def test_complete_unique_translated_release(self):
        release.validate(self.rows)
        self.assertEqual(release.quality(self.rows)['english_translations'], 9500)

    def test_preserves_every_authored_translation_and_example(self):
        actual = {r['lemma']: r for r in self.rows}
        original = release.read_csv(release.ROOT / 'data/exports/release-2000/all.csv')
        for row in original:
            if row['english_translation']:
                for field in ['english_translation', 'dutch_example', 'english_example']:
                    self.assertEqual(row[field], actual[row['lemma']][field], row['lemma'])

    def test_rejects_partial_duplicate_missing_and_invalid_source(self):
        for field, value in [('lemma', self.rows[1]['lemma']), ('english_translation', ''),
                             ('source', 'unattributed'), ('senses_json', '[]'), ('topic', 'unknown')]:
            rows = copy.deepcopy(self.rows)
            index = next(i for i, r in enumerate(rows) if r['source'] == release.SOURCE_ID)
            rows[index][field] = value
            with self.subTest(field=field), self.assertRaises(ValueError):
                release.validate(rows)
        with self.assertRaises(ValueError):
            release.validate(self.rows[:-1])

    def test_source_matches_pinned_checksum(self):
        dictionary = release.dictionary()
        self.assertIn('know', [t for s in dictionary['kennen'] for t in s['translations']])

    def test_multiple_senses_precede_synonyms(self):
        self.assertEqual(release.display_translation([
            {'translations': ['couch', 'sofa']}, {'translations': ['bank']}]), 'couch; bank; sofa')

    def test_frequent_homographs_have_corrections(self):
        actual = {r['lemma']: r['english_translation'] for r in self.rows}
        self.assertEqual(actual['mee'], 'along; with')
        self.assertEqual(actual['hen'], 'them; hen')
        self.assertEqual(actual['week'], 'week; soft')


if __name__ == '__main__':
    unittest.main()
