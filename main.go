package main

import (
	"fmt"
	"math/rand"
	"pehlivan-grand-prix/engine"
	"pehlivan-grand-prix/logger"
	"pehlivan-grand-prix/models"
	"pehlivan-grand-prix/web"
	"time"
)

func startPhysicsEngine(tickrate int, tickChan chan bool) {
	ticker := time.NewTicker(time.Duration(tickrate) * time.Millisecond)
	for range ticker.C {
		//channel'e sinyal gönder
		tickChan <- true
	}
}

// Cell balance durumunu döndür
func getBalanceStatus(delta float64) string {
	if delta < 0.05 {
		return "good"
	} else if delta < 0.15 {
		return "warning"
	}
	return "critical"
}

// Balance'ın performans etkisini hesapla (%)
func getBalanceEffect(delta float64) float64 {
	if delta <= 0.05 {
		return 0 // Etki yok
	}
	// Her 0.05V üzeri = %1 verim kaybı, max %10
	effect := (delta - 0.05) / 0.05
	if effect > 10 {
		effect = 10
	}
	return effect
}

func FlipCoin() bool {
	rand.Seed(time.Now().UnixNano())

	if flipint := rand.Intn(2); flipint == 0 {
		return true
	}
	return false
}

func main() {

	// 1. Takım listesini oluştur (Motor tipinde bir slice)
	var takimlar []*engine.TeamData

	// 2. Takımı listeye ekle
	takimlar = append(takimlar, &engine.TeamData{
		Data: &models.TeamData{
			TeamName: "Pehlivan Racing",
		},
	})

	// Web sunucusunu başlat (arka planda)
	go web.StartServer(":8081")

	// Engine'e time multiplier getter'ı bağla
	engine.GetTimeMultiplier = web.GetTimeMultiplier

	// Logger callback'lerini bağla
	csvLogger := logger.GetLogger()
	web.StartLoggingCallback = func() {
		csvLogger.StartLogging()
	}
	web.StopLoggingCallback = func() {
		csvLogger.StopLogging()
	}

	// AI Driver callback'lerini bağla
	aiDriver := engine.GetAIDriver()
	web.SetAIEnabledCallback = func(enabled bool) {
		aiDriver.SetEnabled(enabled)
	}
	web.SetAIModeCallback = func(mode string) {
		aiDriver.SetMode(mode)
	}

	// Reset callback'ini bağla
	web.ResetSimulationCallback = func() {
		for _, t := range takimlar {
			// Batarya sıfırla
			t.Data.Battery.SoC = 100.0
			t.Data.Battery.UsedCapacity = 0
			t.Data.Battery.EnergyWh = 21.0 * 14 * 3.7 // ~1087 Wh
			t.Data.Battery.Voltage = 58.8
			t.Data.Battery.Current = 0
			t.Data.Battery.Temp = 25.0
			// Hücreleri sıfırla
			for i := range t.Data.Battery.CellBalance {
				t.Data.Battery.CellBalance[i] = 4.2
			}
			t.Data.Battery.CellVoltageMin = 4.2
			t.Data.Battery.CellVoltageMax = 4.2
			// Motor sıfırla
			t.Data.Motor.Speed = 0
			t.Data.Motor.RPM = 0
			t.Data.Motor.Current = 0
			t.Data.Motor.Torque = 0
			t.Data.Motor.PowerW = 0
			t.Data.Motor.Temp = 25.0
			// Yarış durumu sıfırla
			t.Data.RaceStatus.Lap = 1
			t.Data.RaceStatus.Position = 0
			// AI sıfırla
			aiDriver.Reset()
		}
	}

	physicsTick := make(chan bool)
	go startPhysicsEngine(50, physicsTick) // 50ms = 20 FPS
	tickCount := 0

	go func() {
		for range physicsTick {
			// Her tıkta tüm listeyi gez!
			for _, t := range takimlar {
				if !t.Data.IsInitialzied {
					t.InitializeSimulation()
				}

				// AI Driver kontrolü
				dt := 0.05 * web.GetTimeMultiplier()
				aiThrottle := aiDriver.CalculateThrottle(
					t.Data.Motor.Speed,
					t.Data.RaceStatus.Position,
					dt,
				)

				var throttle int
				if aiThrottle >= 0 {
					// AI aktif
					throttle = int(aiThrottle)
				} else {
					// Manuel kontrol
					throttle = int(web.GetThrottle())
				}
				t.ProcessActions("set_throttle", throttle)

				tickCount++
				t.SimulateCar()

				// AI istatistiklerini güncelle
				speedMS := t.Data.Motor.Speed / 3.6
				aiDriver.UpdateStats(t.Data.Motor.PowerW, speedMS, dt)

				// Mevcut segment bilgisi
				currentSeg := engine.GetSegmentAtPosition(t.Data.RaceStatus.Position)

				// WebSocket üzerinden veri gönder
				web.BroadcastSimData(web.SimData{
					Speed:      t.Data.Motor.Speed,
					RPM:        t.Data.Motor.RPM,
					Torque:     t.Data.Motor.Torque,
					Current:    t.Data.Motor.Current,
					MaxCurrent: engine.GetMaxAvailableCurrent(),
					Efficiency: t.Data.Motor.Efficiency,
					Power:      t.Data.Motor.PowerW,
					Throttle:   web.GetThrottle(),
					// Batarya verileri
					BatterySOC:     t.Data.Battery.SoC,
					BatteryVoltage: t.Data.Battery.Voltage,
					BatteryCurrent: t.Data.Battery.Current,
					BatteryTemp:    t.Data.Battery.Temp,
					BatteryPower:   t.Data.Battery.Voltage * t.Data.Battery.Current,
					BatteryCRate:   t.Data.Battery.DischargeRate,
					BatteryEnergy:  t.Data.Battery.EnergyWh,
					BatteryCellMin: t.Data.Battery.CellVoltageMin,
					BatteryCellMax: t.Data.Battery.CellVoltageMax,
					BatteryCells:   t.Data.Battery.CellBalance,
					// Motor
					MotorTemp: t.Data.Motor.Temp,
					// Pist verileri
					Lap:           t.Data.RaceStatus.Lap,
					LapPosition:   t.Data.RaceStatus.Position,
					TotalDistance: float64(t.Data.RaceStatus.Lap)*4200 + t.Data.RaceStatus.Position,
					TrackLength:   4200,
					// Simülasyon hızı
					TimeMultiplier: web.GetTimeMultiplier(),
					// Logging durumu
					IsLogging:   csvLogger.IsLogging(),
					LogFilename: csvLogger.GetFilename(),
					RecordCount: csvLogger.GetRecordCount(),
					// AI Driver durumu
					AIEnabled:        aiDriver.Enabled,
					AIMode:           aiDriver.Mode,
					AITargetSpeed:    aiDriver.TargetSpeed,
					AITargetThrottle: aiDriver.TargetThrottle,
					AIAvgEfficiency:  aiDriver.AvgEfficiency,
					AIInstEfficiency: aiDriver.InstEfficiency,
					// Pist segment bilgisi
					CurrentSegment:  currentSeg.Name,
					SegmentType:     currentSeg.Type,
					SegmentMaxSpeed: currentSeg.MaxSpeed,
					// Cell balance durumu
					BatteryCellDelta: t.Data.Battery.CellVoltageMax - t.Data.Battery.CellVoltageMin,
					BalanceStatus:    getBalanceStatus(t.Data.Battery.CellVoltageMax - t.Data.Battery.CellVoltageMin),
					BalanceEffect:    getBalanceEffect(t.Data.Battery.CellVoltageMax - t.Data.Battery.CellVoltageMin),
				})

				// CSV'ye kaydet (eğer logging aktifse)
				if csvLogger.IsLogging() {
					csvLogger.WriteRecord(logger.SimRecord{
						Timestamp:      time.Now(),
						SimTime:        float64(tickCount) * 0.05 * web.GetTimeMultiplier(),
						Speed:          t.Data.Motor.Speed,
						RPM:            t.Data.Motor.RPM,
						Torque:         t.Data.Motor.Torque,
						Current:        t.Data.Motor.Current,
						Power:          t.Data.Motor.PowerW,
						Efficiency:     t.Data.Motor.Efficiency,
						Throttle:       web.GetThrottle(),
						BatterySOC:     t.Data.Battery.SoC,
						BatteryVoltage: t.Data.Battery.Voltage,
						BatteryCurrent: t.Data.Battery.Current,
						BatteryTemp:    t.Data.Battery.Temp,
						BatteryPower:   t.Data.Battery.Voltage * t.Data.Battery.Current,
						BatteryCRate:   t.Data.Battery.DischargeRate,
						BatteryEnergy:  t.Data.Battery.EnergyWh,
						MotorTemp:      t.Data.Motor.Temp,
						Lap:            t.Data.RaceStatus.Lap,
						LapPosition:    t.Data.RaceStatus.Position,
						TotalDistance:  float64(t.Data.RaceStatus.Lap)*4200 + t.Data.RaceStatus.Position,
						TimeMultiplier: web.GetTimeMultiplier(),
					})
				}
			}
		}
	}()

	fmt.Println("🏎️  Pehlivan Grand Prix Simülasyonu")
	fmt.Println("📡 Web arayüzü: http://localhost:8081")
	fmt.Println("⌨️  Klavye: W/↑ hızlan, S/↓ yavaşla, SPACE acil dur")

	select {}
}
