package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

type batchRow struct {
	Rank               string
	Topic              string
	TopicRank          string
	Lemma              string
	EntryType          string
	CEFR               string
	ReviewStatus       string
	NeedsTopicReview   string
	Notes              string
	EnglishTranslation string
	DutchExample       string
	EnglishExample     string
}

func main() {
	input := flag.String("input", "data/exports/release-2000/all.csv", "release batch export CSV")
	outputDir := flag.String("output-dir", "data/content/release-2000/workqueue", "directory for generated workqueue files")
	flag.Parse()

	rows, err := readBatchRows(*input)
	if err != nil {
		log.Fatalf("read batch rows: %v", err)
	}
	if len(rows) == 0 {
		log.Fatal("no rows found")
	}
	if err := os.MkdirAll(*outputDir, 0o755); err != nil {
		log.Fatalf("create output dir: %v", err)
	}

	if err := writeAllQueue(filepath.Join(*outputDir, "all-work-queue.csv"), rows); err != nil {
		log.Fatalf("write all queue: %v", err)
	}
	if err := writeTopicReview(filepath.Join(*outputDir, "topic-review.csv"), rows); err != nil {
		log.Fatalf("write topic review: %v", err)
	}
	topicCounts, err := writeTopicFiles(*outputDir, rows)
	if err != nil {
		log.Fatalf("write topic files: %v", err)
	}
	if err := writeSummary(filepath.Join(*outputDir, "SUMMARY.md"), rows, topicCounts); err != nil {
		log.Fatalf("write summary: %v", err)
	}

	log.Printf("prepared %d release rows in %s", len(rows), *outputDir)
}

func readBatchRows(path string) ([]batchRow, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.FieldsPerRecord = -1
	header, err := reader.Read()
	if err != nil {
		return nil, err
	}
	columns := map[string]int{}
	for index, column := range header {
		columns[strings.TrimSpace(column)] = index
	}
	for _, required := range []string{"rank", "topic", "topic_rank", "lemma", "entry_type", "cefr", "review_status", "needs_topic_review", "notes", "english_translation", "dutch_example", "english_example"} {
		if _, ok := columns[required]; !ok {
			return nil, fmt.Errorf("missing required column %q", required)
		}
	}

	rows := make([]batchRow, 0)
	for {
		record, err := reader.Read()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		rows = append(rows, batchRow{
			Rank:               cell(record, columns["rank"]),
			Topic:              cell(record, columns["topic"]),
			TopicRank:          cell(record, columns["topic_rank"]),
			Lemma:              cell(record, columns["lemma"]),
			EntryType:          cell(record, columns["entry_type"]),
			CEFR:               cell(record, columns["cefr"]),
			ReviewStatus:       cell(record, columns["review_status"]),
			NeedsTopicReview:   cell(record, columns["needs_topic_review"]),
			Notes:              cell(record, columns["notes"]),
			EnglishTranslation: cell(record, columns["english_translation"]),
			DutchExample:       cell(record, columns["dutch_example"]),
			EnglishExample:     cell(record, columns["english_example"]),
		})
	}
	return rows, nil
}

func writeAllQueue(path string, rows []batchRow) error {
	return writeCSV(path, []string{"rank", "topic", "topic_rank", "lemma", "entry_type", "cefr", "topic_decision", "content_status", "english_translation", "dutch_example", "english_example", "notes"}, func(writer *csv.Writer) error {
		for _, row := range rows {
			topicDecision := "accepted"
			if row.Topic == "" || row.NeedsTopicReview == "true" {
				topicDecision = "needs_topic_review"
			}
			contentStatus := "needs_translation"
			if row.EnglishTranslation != "" && row.DutchExample != "" && row.EnglishExample != "" {
				contentStatus = row.ReviewStatus
			}
			if err := writer.Write([]string{row.Rank, row.Topic, row.TopicRank, row.Lemma, row.EntryType, row.CEFR, topicDecision, contentStatus, row.EnglishTranslation, row.DutchExample, row.EnglishExample, row.Notes}); err != nil {
				return err
			}
		}
		return nil
	})
}

func writeTopicReview(path string, rows []batchRow) error {
	return writeCSV(path, []string{"rank", "lemma", "entry_type", "cefr", "suggested_topic", "topic_decision", "notes"}, func(writer *csv.Writer) error {
		for _, row := range rows {
			if row.Topic != "" && row.NeedsTopicReview != "true" {
				continue
			}
			if err := writer.Write([]string{row.Rank, row.Lemma, row.EntryType, row.CEFR, suggestedTopic(row), "needs_topic_review", row.Notes}); err != nil {
				return err
			}
		}
		return nil
	})
}

func writeTopicFiles(outputDir string, rows []batchRow) (map[string]int, error) {
	byTopic := map[string][]batchRow{}
	for _, row := range rows {
		if row.Topic == "" || row.NeedsTopicReview == "true" {
			continue
		}
		byTopic[row.Topic] = append(byTopic[row.Topic], row)
	}
	topics := make([]string, 0, len(byTopic))
	for topic := range byTopic {
		topics = append(topics, topic)
	}
	sort.Strings(topics)

	counts := map[string]int{}
	for _, topic := range topics {
		rows := byTopic[topic]
		counts[topic] = len(rows)
		path := filepath.Join(outputDir, topic+".csv")
		if err := writeCSV(path, []string{"lemma", "english_translation", "dutch_example", "english_example"}, func(writer *csv.Writer) error {
			for _, row := range rows {
				if err := writer.Write([]string{row.Lemma, row.EnglishTranslation, row.DutchExample, row.EnglishExample}); err != nil {
					return err
				}
			}
			return nil
		}); err != nil {
			return nil, err
		}
	}
	return counts, nil
}

func writeSummary(path string, rows []batchRow, topicCounts map[string]int) error {
	total := len(rows)
	needsTopic := 0
	withContent := 0
	for _, row := range rows {
		if row.Topic == "" || row.NeedsTopicReview == "true" {
			needsTopic++
		}
		if row.EnglishTranslation != "" && row.DutchExample != "" && row.EnglishExample != "" {
			withContent++
		}
	}
	topics := make([]string, 0, len(topicCounts))
	for topic := range topicCounts {
		topics = append(topics, topic)
	}
	sort.Strings(topics)

	var builder strings.Builder
	builder.WriteString("# Release 2,000 Content Workqueue\n\n")
	builder.WriteString("Generated from `data/exports/release-2000/all.csv`.\n\n")
	builder.WriteString("## Counts\n\n")
	builder.WriteString(fmt.Sprintf("- Total reserved words: %s\n", strconv.Itoa(total)))
	builder.WriteString(fmt.Sprintf("- Words with imported content: %s\n", strconv.Itoa(withContent)))
	builder.WriteString(fmt.Sprintf("- Words needing topic review: %s\n", strconv.Itoa(needsTopic)))
	builder.WriteString(fmt.Sprintf("- Words ready for translation by topic: %s\n", strconv.Itoa(total-needsTopic)))
	builder.WriteString("\n## Files\n\n")
	builder.WriteString("- `all-work-queue.csv`: full 2,000-word release workspace.\n")
	builder.WriteString("- `topic-review.csv`: words that need topic assignment or rejection before translation.\n")
	builder.WriteString("- `<topic>.csv`: import-ready content files for assigned topics.\n")
	builder.WriteString("\n## Assigned Topic Counts\n\n")
	for _, topic := range topics {
		builder.WriteString(fmt.Sprintf("- `%s`: %d\n", topic, topicCounts[topic]))
	}
	builder.WriteString("\n## Review Rule\n\n")
	builder.WriteString("Generated or drafted translations should be imported with `--status needs_content_review` until checked by a Dutch speaker or reviewer.\n")
	return os.WriteFile(path, []byte(builder.String()), 0o644)
}

func suggestedTopic(row batchRow) string {
	const prefix = "candidate topics:"
	index := strings.Index(row.Notes, prefix)
	if index == -1 {
		return ""
	}
	candidates := strings.TrimSpace(row.Notes[index+len(prefix):])
	if comma := strings.Index(candidates, ","); comma != -1 {
		return strings.TrimSpace(candidates[:comma])
	}
	return candidates
}

func writeCSV(path string, header []string, writeRows func(*csv.Writer) error) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	writer := csv.NewWriter(file)
	if err := writer.Write(header); err != nil {
		return err
	}
	if err := writeRows(writer); err != nil {
		return err
	}
	writer.Flush()
	return writer.Error()
}

func cell(record []string, index int) string {
	if index >= len(record) {
		return ""
	}
	return strings.TrimSpace(record[index])
}
