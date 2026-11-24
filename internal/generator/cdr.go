package cdr_generator

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"time"
)

// CDRRecord представляет одну запись CDR.
type CDRRecord struct {
	RecordID        string
	RecordType      string // MOC, MTC, SMS, GPRS, …
	ServedIMSI      string
	ServedMSISDN    string
	CallingNumber   string
	CalledNumber    string
	StartTime       time.Time
	DurationSec     int
	CauseForTerm    string
	MCC             string
	MNC             string
	LAC             string
	CellID          string
	RecordingEntity string
	ChargingID      string
	SequenceNumber  int
	PartialType     string // NONE, FIRST, INTERMEDIATE, LAST
	IngestTime      time.Time
	SourceFile      string
	RoamingFlag     bool
	PartnerPLMN     string
}

// csvHeader — фиксированный порядок колонок, совпадает с примером.
var csvHeader = []string{
	"record_id",
	"record_type",
	"served_imsi",
	"served_msisdn",
	"calling_number",
	"called_number",
	"start_time",
	"duration_sec",
	"cause_for_term",
	"mcc",
	"mnc",
	"lac",
	"cell_id",
	"recording_entity",
	"charging_id",
	"sequence_number",
	"partial_type",
	"ingest_time",
	"source_file",
	"roaming_flag",
	"partner_plmn",
}

const timeLayout = time.RFC3339 // 2006-01-02T15:04:05Z07:00

// toRow конвертирует CDRRecord в срез строк для csv.Writer.
func (r CDRRecord) toRow() []string {
	return []string{
		r.RecordID,
		r.RecordType,
		r.ServedIMSI,
		r.ServedMSISDN,
		r.CallingNumber,
		r.CalledNumber,
		r.StartTime.Format(timeLayout),
		strconv.Itoa(r.DurationSec),
		r.CauseForTerm,
		r.MCC,
		r.MNC,
		r.LAC,
		r.CellID,
		r.RecordingEntity,
		r.ChargingID,
		strconv.Itoa(r.SequenceNumber),
		r.PartialType,
		r.IngestTime.Format(timeLayout),
		r.SourceFile,
		strconv.FormatBool(r.RoamingFlag),
		r.PartnerPLMN,
	}
}

// CreateReport создаёт новый CDR-файл по указанному пути,
// записывает CSV-заголовок и возвращает открытый *os.File.
// Вызывающий код обязан закрыть файл после использования.
func CreateReport(path string) (*os.File, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return nil, fmt.Errorf("cdr: create report %q: %w", path, err)
	}

	w := csv.NewWriter(f)
	if err := w.Write(csvHeader); err != nil {
		f.Close()
		return nil, fmt.Errorf("cdr: write header to %q: %w", path, err)
	}
	w.Flush()
	if err := w.Error(); err != nil {
		f.Close()
		return nil, fmt.Errorf("cdr: flush header to %q: %w", path, err)
	}

	return f, nil
}

// AppendRecord дописывает одну CDR-запись в конец уже открытого файла f.
// Файл должен быть открыт для записи; функция сама позиционируется в конец.
func AppendRecord(f *os.File, r CDRRecord) error {
	if _, err := f.Seek(0, 2); err != nil { // 2 == io.SeekEnd
		return fmt.Errorf("cdr: seek to end of %q: %w", f.Name(), err)
	}

	w := csv.NewWriter(f)
	if err := w.Write(r.toRow()); err != nil {
		return fmt.Errorf("cdr: write record %q to %q: %w", r.RecordID, f.Name(), err)
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return fmt.Errorf("cdr: flush record %q to %q: %w", r.RecordID, f.Name(), err)
	}

	return nil
}

// AppendRecordToPath открывает файл по пути и дописывает запись.
// Удобно для разовых вызовов, когда постоянный файловый дескриптор не нужен.
func AppendRecordToPath(path string, r CDRRecord) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return fmt.Errorf("cdr: open %q for append: %w", path, err)
	}
	defer f.Close()

	return AppendRecord(f, r)
}
