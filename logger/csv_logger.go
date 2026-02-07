package logger

import (
	"encoding/csv"
	"fmt"
	"os"
	"sync"
	"time"
)

// SimRecord bir simülasyon kaydı
type SimRecord struct {
	Timestamp      time.Time
	SimTime        float64 // Simülasyon zamanı (saniye)
	Speed          float64
	RPM            int
	Torque         float64
	Current        float64
	MaxCurrent     float64
	Power          float64
	Efficiency     float64
	Throttle       float64
	BatterySOC     float64
	BatteryVoltage float64
	BatteryCurrent float64
	BatteryTemp    float64
	BatteryPower   float64
	BatteryCRate   float64
	BatteryEnergy  float64
	MotorTemp      float64
	Lap            int
	LapPosition    float64
	TotalDistance  float64
	TimeMultiplier float64
}

// CSVLogger CSV dosyasına veri kaydeden yapı
type CSVLogger struct {
	file        *os.File
	writer      *csv.Writer
	isLogging   bool
	mu          sync.Mutex
	startTime   time.Time
	simTime     float64
	recordCount int
	filename    string
}

var (
	logger     *CSVLogger
	loggerOnce sync.Once
)

// GetLogger singleton logger instance döndürür
func GetLogger() *CSVLogger {
	loggerOnce.Do(func() {
		logger = &CSVLogger{}
	})
	return logger
}

// StartLogging yeni bir CSV dosyası oluşturur ve logging başlatır
func (l *CSVLogger) StartLogging() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.isLogging {
		return fmt.Errorf("logging zaten aktif")
	}

	// Dosya adı: sim_YYYYMMDD_HHMMSS.csv
	l.filename = fmt.Sprintf("sim_%s.csv", time.Now().Format("20060102_150405"))

	file, err := os.Create(l.filename)
	if err != nil {
		return fmt.Errorf("CSV dosyası oluşturulamadı: %v", err)
	}

	l.file = file
	l.writer = csv.NewWriter(file)
	l.startTime = time.Now()
	l.simTime = 0
	l.recordCount = 0

	// Header yaz
	header := []string{
		"Timestamp",
		"SimTime_s",
		"Speed_kmh",
		"RPM",
		"Torque_Nm",
		"Current_A",
		"MaxCurrent_A",
		"Power_W",
		"Efficiency_pct",
		"Throttle_pct",
		"Battery_SOC_pct",
		"Battery_Voltage_V",
		"Battery_Current_A",
		"Battery_Temp_C",
		"Battery_Power_W",
		"Battery_CRate",
		"Battery_Energy_Wh",
		"Motor_Temp_C",
		"Lap",
		"LapPosition_m",
		"TotalDistance_m",
		"TimeMultiplier",
	}
	if err := l.writer.Write(header); err != nil {
		l.file.Close()
		return fmt.Errorf("header yazılamadı: %v", err)
	}
	l.writer.Flush()

	l.isLogging = true
	fmt.Printf("📝 CSV logging başladı: %s\n", l.filename)
	return nil
}

// StopLogging logging'i durdurur ve dosyayı kapatır
func (l *CSVLogger) StopLogging() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if !l.isLogging {
		return fmt.Errorf("logging aktif değil")
	}

	l.writer.Flush()
	err := l.file.Close()
	l.isLogging = false

	fmt.Printf("📝 CSV logging durduruldu: %s (%d kayıt)\n", l.filename, l.recordCount)
	return err
}

// IsLogging logging durumunu döndürür
func (l *CSVLogger) IsLogging() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.isLogging
}

// GetFilename aktif dosya adını döndürür
func (l *CSVLogger) GetFilename() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.filename
}

// GetRecordCount kayıt sayısını döndürür
func (l *CSVLogger) GetRecordCount() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.recordCount
}

// WriteRecord bir kayıt yazar
func (l *CSVLogger) WriteRecord(r SimRecord) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if !l.isLogging {
		return nil // Logging aktif değilse sessizce dön
	}

	// Simülasyon zamanını güncelle
	l.simTime += 0.05 * r.TimeMultiplier // 50ms * time multiplier

	record := []string{
		time.Now().Format("2006-01-02 15:04:05.000"),
		fmt.Sprintf("%.2f", l.simTime),
		fmt.Sprintf("%.2f", r.Speed),
		fmt.Sprintf("%d", r.RPM),
		fmt.Sprintf("%.3f", r.Torque),
		fmt.Sprintf("%.2f", r.Current),
		fmt.Sprintf("%.2f", r.MaxCurrent),
		fmt.Sprintf("%.1f", r.Power),
		fmt.Sprintf("%.1f", r.Efficiency),
		fmt.Sprintf("%.1f", r.Throttle),
		fmt.Sprintf("%.2f", r.BatterySOC),
		fmt.Sprintf("%.2f", r.BatteryVoltage),
		fmt.Sprintf("%.2f", r.BatteryCurrent),
		fmt.Sprintf("%.1f", r.BatteryTemp),
		fmt.Sprintf("%.1f", r.BatteryPower),
		fmt.Sprintf("%.3f", r.BatteryCRate),
		fmt.Sprintf("%.1f", r.BatteryEnergy),
		fmt.Sprintf("%.1f", r.MotorTemp),
		fmt.Sprintf("%d", r.Lap),
		fmt.Sprintf("%.1f", r.LapPosition),
		fmt.Sprintf("%.1f", r.TotalDistance),
		fmt.Sprintf("%.1f", r.TimeMultiplier),
	}

	if err := l.writer.Write(record); err != nil {
		return err
	}

	l.recordCount++

	// Her 100 kayıtta flush yap
	if l.recordCount%100 == 0 {
		l.writer.Flush()
	}

	return nil
}
