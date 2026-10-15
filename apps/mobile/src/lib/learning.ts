export type CEFRLevel = 'A1' | 'A2' | 'B1' | 'B2' | 'C1' | 'C2';

export type CEFRFilter = {
  level: CEFRLevel;
  wordCount: number;
  isAvailable: boolean;
};

export type Topic = {
  slug: string;
  title: string;
  description: string;
  wordCount: number;
  classifiedWordCount?: number;
  startedCount: number;
  isAvailable: boolean;
};

export type LearningFilters = {
  topics: Topic[];
  cefrLevels: CEFRFilter[];
};

export type LearningWord = {
  id: string;
  topic: string;
  word: string;
  translation: string;
};
