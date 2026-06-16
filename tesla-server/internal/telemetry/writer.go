package telemetry

import (
	"log"
	"sync"
	"tesla-server/internal/database"
	"tesla-server/models"
	"time"
)

const (
	batchSize     = 50
	flushInterval = 3 * time.Second
)

// 遥测数据分类
const (
	CategoryRealtime = "realtime"
	CategoryState    = "state"
	CategoryMedia    = "media"
	CategoryRaw      = "raw"
)

// rawRecord 待写入的原始记录
type rawRecord struct {
	VIN     string
	Data    models.JSONMap
	Topic   string
	Txid    string
	RawData []byte
}

// TelemetryWriter 异步批量写入遥测原始数据到 MySQL
type TelemetryWriter struct {
	mu      sync.Mutex
	buffers map[string][]rawRecord // key: category
	stopCh  chan struct{}
	doneCh  chan struct{}
}

var writer *TelemetryWriter

// InitWriter 初始化遥测数据写入器
func InitWriter() {
	writer = &TelemetryWriter{
		buffers: make(map[string][]rawRecord),
		stopCh:  make(chan struct{}),
		doneCh:  make(chan struct{}),
	}
	go writer.run()
	log.Printf("[TelemetryWriter] Started (batch=%d, flush=%v)", batchSize, flushInterval)
}

// StopWriter 停止写入器，刷出剩余数据
func StopWriter() {
	if writer == nil {
		return
	}
	close(writer.stopCh)
	<-writer.doneCh
}

// RecordRealtime 记录实时数据（遥测推送，只记录实际推送的字段）
func RecordRealtime(vin string, fields map[string]interface{}) {
	if writer == nil {
		return
	}
	m := models.JSONMap{}
	for k, v := range fields {
		m[k] = v
	}
	writer.enqueue(CategoryRealtime, rawRecord{VIN: vin, Data: m})
}

// RecordState 记录车辆状态数据（遥测推送）
func RecordState(vin string, fields map[string]interface{}) {
	if writer == nil {
		return
	}
	m := models.JSONMap{}
	for k, v := range fields {
		m[k] = v
	}
	writer.enqueue(CategoryState, rawRecord{VIN: vin, Data: m})
}

// RecordMedia 记录媒体数据（遥测推送，只记录实际推送的字段）
func RecordMedia(vin string, fields map[string]interface{}) {
	if writer == nil {
		return
	}
	m := models.JSONMap{}
	for k, v := range fields {
		m[k] = v
	}
	writer.enqueue(CategoryMedia, rawRecord{VIN: vin, Data: m})
}

// RecordRaw 记录原始二进制数据（用于事后分析）
func RecordRaw(vin, topic, txid string, rawData []byte) {
	if writer == nil {
		return
	}
	writer.enqueue(CategoryRaw, rawRecord{VIN: vin, Topic: topic, Txid: txid, RawData: rawData})
}

func (w *TelemetryWriter) enqueue(category string, record rawRecord) {
	w.mu.Lock()
	w.buffers[category] = append(w.buffers[category], record)
	w.mu.Unlock()
}

func (w *TelemetryWriter) run() {
	ticker := time.NewTicker(flushInterval)
	defer ticker.Stop()
	defer close(w.doneCh)

	for {
		select {
		case <-ticker.C:
			w.flush()
		case <-w.stopCh:
			w.flush()
			return
		}
	}
}

func (w *TelemetryWriter) flush() {
	w.mu.Lock()
	snapshot := w.buffers
	w.buffers = make(map[string][]rawRecord)
	w.mu.Unlock()

	if len(snapshot) == 0 {
		return
	}

	totalRecords := 0
	for _, records := range snapshot {
		totalRecords += len(records)
	}
	log.Printf("[TelemetryWriter] Flushing %d records across %d categories", totalRecords, len(snapshot))

	for category, records := range snapshot {
		if len(records) == 0 {
			continue
		}
		w.writeBatch(category, records)
	}
}

func (w *TelemetryWriter) writeBatch(category string, records []rawRecord) {
	db := database.GetDB()
	if db == nil {
		log.Printf("[TelemetryWriter] DB is nil, skipping %d %s records", len(records), category)
		return
	}

	now := time.Now()

	switch category {
	case CategoryRealtime:
		batch := make([]models.TelemetryRealtime, 0, len(records))
		for _, r := range records {
			batch = append(batch, models.TelemetryRealtime{
				VIN:       r.VIN,
				Data:      r.Data,
				CreatedAt: now,
			})
		}
		if err := db.CreateInBatches(batch, batchSize).Error; err != nil {
			log.Printf("[TelemetryWriter] Failed to write %d realtime records: %v", len(batch), err)
		} else {
			log.Printf("[TelemetryWriter] Wrote %d realtime records to DB", len(batch))
		}

	case CategoryState:
		batch := make([]models.TelemetryState, 0, len(records))
		for _, r := range records {
			batch = append(batch, models.TelemetryState{
				VIN:       r.VIN,
				Data:      r.Data,
				CreatedAt: now,
			})
		}
		if err := db.CreateInBatches(batch, batchSize).Error; err != nil {
			log.Printf("[TelemetryWriter] Failed to write %d state records: %v", len(batch), err)
		} else {
			log.Printf("[TelemetryWriter] Wrote %d state records to DB", len(batch))
		}

	case CategoryMedia:
		batch := make([]models.TelemetryMedia, 0, len(records))
		for _, r := range records {
			batch = append(batch, models.TelemetryMedia{
				VIN:       r.VIN,
				Data:      r.Data,
				CreatedAt: now,
			})
		}
		if err := db.CreateInBatches(batch, batchSize).Error; err != nil {
			log.Printf("[TelemetryWriter] Failed to write %d media records: %v", len(batch), err)
		} else {
			log.Printf("[TelemetryWriter] Wrote %d media records to DB", len(batch))
		}

	case CategoryRaw:
		batch := make([]models.TelemetryRaw, 0, len(records))
		for _, r := range records {
			batch = append(batch, models.TelemetryRaw{
				VIN:       r.VIN,
				Topic:     r.Topic,
				Txid:      r.Txid,
				RawData:   r.RawData,
				CreatedAt: now,
			})
		}
		if err := db.CreateInBatches(batch, batchSize).Error; err != nil {
			log.Printf("[TelemetryWriter] Failed to write %d raw records: %v", len(batch), err)
		} else {
			log.Printf("[TelemetryWriter] Wrote %d raw records to DB", len(batch))
		}
	}
}
