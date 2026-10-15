import json
import unittest
import category_release as categories

class CategoryTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.original, cls.rows, cls.rules, cls.assignments = categories.build()
        cls.by_topic = {(a['lemma'], a['topic']): a for a in cls.assignments}

    def test_exact_sense_prevents_ambiguous_false_matches(self):
        self.assertNotIn(('kam', 'public-transport'), self.by_topic)
        self.assertNotIn(('pupil', 'school-childcare'), self.by_topic)
        self.assertNotIn(('hozen', 'legal-contracts'), self.by_topic)
        self.assertNotIn(('beslist', 'work-office'), self.by_topic)
        self.assertNotIn(('dragend', 'engineering-technical'), self.by_topic)
        self.assertNotIn(('kut', 'healthcare'), self.by_topic)
        pupil = self.by_topic[('pupil', 'healthcare')]
        self.assertEqual(pupil['translation'], 'pupil')
        self.assertTrue(all('(anatomie)' in m['dutch'] for m in json.loads(pupil['meanings_json'])))

    def test_matched_category_sense_not_entire_dictionary_entry(self):
        frost = self.by_topic[('vorst', 'weather-small-talk')]
        monarch = self.by_topic[('vorst', 'politics-news')]
        self.assertEqual(frost['translation'], 'frost')
        self.assertEqual(monarch['translation'], 'monarch')
        self.assertNotEqual(frost['meanings_json'], monarch['meanings_json'])

    def test_cap_preserves_originals_and_assignment_provenance(self):
        original = {r['lemma']: r for r in self.original}
        counts = {}
        for a in self.assignments:
            self.assertNotEqual(a['topic'], original[a['lemma']]['topic'])
            self.assertTrue(a['evidence'])
            self.assertTrue(json.loads(a['meanings_json']))
            counts[a['lemma']] = counts.get(a['lemma'], 0) + 1
        for lemma, count in counts.items():
            self.assertLessEqual(count + (original[lemma]['topic'] != 'general-vocabulary'), 3)
        self.assertEqual(self.assignments, categories.release.read_csv(categories.DEST/'assignments.csv'))
        sample = categories.pilot(self.rows, self.original, self.rules)
        self.assertEqual(len({r['lemma'] for r in sample}), 500)

if __name__ == '__main__': unittest.main()
