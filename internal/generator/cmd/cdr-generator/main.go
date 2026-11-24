package main

import (
	"flag"
	"fmt"
	"log"
	"time"

	generator "cdr_generator"
)

func main() {
	output := flag.String("out", "cdr_report.csv", "путь к выходному CDR CSV-файлу")
	count := flag.Int("count", 10, "количество создаваемых CDR-записей")
	flag.Parse()

	if *count < 0 {
		log.Fatal("количество записей не может быть отрицательным")
	}

	f, err := generator.CreateReport(*output)
	if err != nil {
		log.Fatal(err)
	}
	for sequence := 1; sequence <= *count; sequence++ {
		record := newRecord(sequence)
		if err := generator.AppendRecord(f, record); err != nil {
			_ = f.Close()
			log.Fatal(err)
		}
	}
	if err := f.Close(); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Создан CSV-файл с %d CDR-записями: %s\n", *count, *output)
}

// newRecord — болванка записи; значения полей будут заполнены логикой генерации.
func newRecord(sequence int) generator.CDRRecord {
	return generator.CDRRecord{
		RecordID:        "",
		RecordType:      "",
		ServedIMSI:      "",
		ServedMSISDN:    "",
		CallingNumber:   "",
		CalledNumber:    "",
		StartTime:       time.Time{},
		DurationSec:     0,
		CauseForTerm:    "",
		MCC:             "",
		MNC:             "",
		LAC:             "",
		CellID:          "",
		RecordingEntity: "",
		ChargingID:      "",
		SequenceNumber:  sequence,
		PartialType:     "",
		IngestTime:      time.Time{},
		SourceFile:      "",
		RoamingFlag:     false,
		PartnerPLMN:     "",
	}
}
