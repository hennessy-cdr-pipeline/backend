package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"time"

	generator "github.com/hennessy-cdr-pipeline/backend/internal/generator"
)

func main() {
	output := flag.String("out", "cdr_report.csv", "путь к выходному CDR CSV-файлу")
	count := flag.Int("count", 10, "количество создаваемых CDR-записей")
	subscribersPath := flag.String("subscribers", "data/subscribers.csv", "путь к справочнику абонентов")
	flag.Parse()
	if *count < 0 {
		log.Fatal("количество записей не может быть отрицательным")
	}
	subscribers, err := loadSubscribers(*subscribersPath)
	if err != nil {
		log.Fatal(err)
	}
	if len(subscribers) == 0 {
		log.Fatal("справочник абонентов пуст")
	}

	f, err := generator.CreateReport(*output)
	if err != nil {
		log.Fatal(err)
	}
	for sequence := 1; sequence <= *count; sequence++ {
		if err := generator.AppendRecord(f, newRecord(sequence, subscribers)); err != nil {
			_ = f.Close()
			log.Fatal(err)
		}
	}
	if err := f.Close(); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Создан CSV-файл с %d CDR-записями: %s\n", *count, *output)
}

type subscriber struct {
	msisdn  string
	imsi    string
	mcc     string
	mnc     string
	profile string
}

func loadSubscribers(path string) ([]subscriber, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("открыть справочник абонентов %q: %w", path, err)
	}
	defer f.Close()

	reader := csv.NewReader(f)
	header, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("прочитать заголовок справочника: %w", err)
	}
	columns := make(map[string]int, len(header))
	for i, name := range header {
		columns[name] = i
	}
	for _, name := range []string{"msisdn", "imsi", "mcc", "mnc", "profile"} {
		if _, ok := columns[name]; !ok {
			return nil, fmt.Errorf("в справочнике отсутствует колонка %q", name)
		}
	}

	var result []subscriber
	for line := 2; ; line++ {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("прочитать строку %d справочника: %w", line, err)
		}
		get := func(name string) string { return row[columns[name]] }
		item := subscriber{
			msisdn: get("msisdn"), imsi: get("imsi"), mcc: get("mcc"), mnc: get("mnc"), profile: get("profile"),
		}
		if item.msisdn == "" || item.imsi == "" || item.mcc == "" || item.mnc == "" || item.profile == "" {
			return nil, fmt.Errorf("пустое обязательное поле в строке %d справочника", line)
		}
		result = append(result, item)
	}
	return result, nil
}

func newRecord(sequence int, subscribers []subscriber) generator.CDRRecord {
	served := fieldSubscriber(sequence, subscribers)
	typ := fieldRecordType(sequence)
	start := fieldStartTime(sequence)
	entity := fieldRecordingEntity(sequence)
	partial := fieldPartialType()
	roamingEvent := fieldIsRoamingEvent(typ, served)
	mcc, mnc := fieldMCC(served, roamingEvent), fieldMNC(served, roamingEvent)
	roaming := fieldRoamingFlag(mcc, mnc)

	return generator.CDRRecord{
		RecordID:        fieldRecordID(sequence),
		RecordType:      typ,
		ServedIMSI:      fieldServedIMSI(served),
		ServedMSISDN:    fieldServedMSISDN(served),
		CallingNumber:   fieldCallingNumber(typ, served, sequence),
		CalledNumber:    fieldCalledNumber(typ, served, sequence),
		StartTime:       start,
		DurationSec:     fieldDurationSec(typ),
		CauseForTerm:    fieldCauseForTerm(),
		MCC:             mcc,
		MNC:             mnc,
		LAC:             fieldLAC(sequence),
		CellID:          fieldCellID(sequence),
		RecordingEntity: entity,
		ChargingID:      fieldChargingID(sequence),
		SequenceNumber:  fieldSequenceNumber(partial),
		PartialType:     partial,
		IngestTime:      fieldIngestTime(start),
		SourceFile:      fieldSourceFile(entity, start),
		RoamingFlag:     roaming,
		PartnerPLMN:     fieldPartnerPLMN(roaming),
	}
}

func fieldRecordID(sequence int) string { return fmt.Sprintf("cdr-%08d", sequence) }
func fieldRecordType(sequence int) string {
	types := [...]string{"MOC", "MTC", "SMS_MO", "SMS_MT"}
	return types[(sequence-1)%len(types)]
}
func fieldSubscriber(sequence int, subscribers []subscriber) subscriber {
	return subscribers[(sequence-1)%len(subscribers)]
}
func fieldServedIMSI(s subscriber) string   { return s.imsi }
func fieldServedMSISDN(s subscriber) string { return s.msisdn }
func fieldCallingNumber(recordType string, s subscriber, sequence int) string {
	if recordType == "MOC" || recordType == "SMS_MO" {
		return s.msisdn
	}
	return fieldOtherNumber(sequence)
}
func fieldCalledNumber(recordType string, s subscriber, sequence int) string {
	if recordType == "MTC" || recordType == "SMS_MT" {
		return s.msisdn
	}
	return fieldOtherNumber(sequence)
}
func fieldOtherNumber(sequence int) string { return fmt.Sprintf("7900%07d", sequence) }
func fieldStartTime(sequence int) time.Time {
	moscow := time.FixedZone("MSK", 3*60*60)
	return time.Date(2026, 1, 1, 0, 0, 0, 0, moscow).Add(time.Duration(sequence-1) * time.Minute)
}
func fieldDurationSec(recordType string) int {
	if recordType == "SMS_MO" || recordType == "SMS_MT" {
		return 0
	}
	return 60
}
func fieldCauseForTerm() string { return "NORMAL_RELEASE" }
func fieldIsRoamingEvent(recordType string, s subscriber) bool {
	return s.profile == "roamer" && (recordType == "MOC" || recordType == "MTC")
}
func fieldMCC(s subscriber, roaming bool) string {
	if roaming {
		return fieldPartnerMCC()
	}
	return s.mcc
}
func fieldMNC(s subscriber, roaming bool) string {
	if roaming {
		return fieldPartnerMNC()
	}
	return s.mnc
}
func fieldPartnerMCC() string         { return "262" }
func fieldPartnerMNC() string         { return "01" }
func fieldLAC(sequence int) string    { return strconv.Itoa(1000 + sequence%50000) }
func fieldCellID(sequence int) string { return fmt.Sprintf("0x%04X", 0x1000+sequence%0xEFFF) }
func fieldRecordingEntity(sequence int) string {
	entities := [...]string{"MSC-MSK-01", "MSC-MSK-02"}
	return entities[(sequence-1)%len(entities)]
}
func fieldChargingID(sequence int) string { return fmt.Sprintf("charge-%08d", sequence) }
func fieldPartialType() string            { return "NONE" }
func fieldSequenceNumber(partialType string) int {
	if partialType == "NONE" {
		return 1
	}
	return 1
}
func fieldIngestTime(start time.Time) time.Time { return start.Add(5 * time.Minute) }
func fieldSourceFile(entity string, start time.Time) string {
	return fmt.Sprintf("%s_%s.csv", entity, start.Format("20060102_15"))
}
func fieldRoamingFlag(mcc, mnc string) bool { return mcc != "250" || mnc != "01" }
func fieldPartnerPLMN(roaming bool) string {
	if !roaming {
		return ""
	}
	return fieldPartnerMCC() + fieldPartnerMNC()
}
