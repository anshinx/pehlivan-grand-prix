package engine

import (
	"fmt"
	"math"
	"pehlivan-grand-prix/models"
	"pehlivan-grand-prix/web"
)

type TeamData struct {
	Data *models.TeamData
}

// Hücre varyasyon parametreleri
type CellVariationParams struct {
	CapacityVariation   float64 // Kapasite varyasyonu (0.008 = %0.8 per step)
	ResistanceVariation float64 // İç direnç varyasyonu (%)
	RandomVariation     float64 // Rastgele voltaj varyasyonu
}

// Varyasyon seviyelerine göre parametreler
func GetVariationParams() CellVariationParams {
	level := web.GetCellVariationLevel()
	switch level {
	case "min":
		return CellVariationParams{
			CapacityVariation:   0.002,  // %0-1 kapasite farkı
			ResistanceVariation: 0.05,   // ±5% R varyasyonu
			RandomVariation:     0.0002, // ±0.02%
		}
	case "high":
		return CellVariationParams{
			CapacityVariation:   0.016, // %0-8 kapasite farkı
			ResistanceVariation: 0.25,  // ±25% R varyasyonu
			RandomVariation:     0.002, // ±0.2%
		}
	default: // medium
		return CellVariationParams{
			CapacityVariation:   0.008, // %0-3.2 kapasite farkı
			ResistanceVariation: 0.10,  // ±10% R varyasyonu
			RandomVariation:     0.001, // ±0.1%
		}
	}
}

// Tüm veriyi önden her takım için giriyoruz ki başlangıç noktasında sıkıntı çıkmasın.
// 0 olanlar zorunlu değil ama  ¯\_(ツ)_/¯ meeh,
func (team *TeamData) InitializeSimulation() {
	if team.Data.IsInitialzied {
		return
	}
	team.Data.Stats.ActionCount = 0
	team.Data.Stats.TotalPackets = 0
	team.Data.IsInitialzied = true

	// Battery initialization (14S7P Battery Pack)
	// 14S = 14 hücre seri, 7P = 7 hücre paralel
	// Hücre: 18650/21700 Li-ion, 3Ah nominal
	cellCapacity := 3.0                                    // Ah (tek hücre)
	parallelCells := 7                                     // 7P
	seriesCells := 14                                      // 14S
	totalCapacity := cellCapacity * float64(parallelCells) // 21Ah

	// Voltaj aralığı (Li-ion)
	cellVoltageMax := 4.20 // V (tam şarj)
	cellVoltageNom := 3.70 // V (nominal)
	_ = 3.00               // cellVoltageMin - V (minimum güvenli), simülasyonda kullanılıyor

	// Başlangıç: %100 şarjlı
	initialCellVoltage := cellVoltageMax
	packVoltage := initialCellVoltage * float64(seriesCells) // 58.8V

	// Enerji hesabı
	nominalEnergy := cellVoltageNom * float64(seriesCells) * totalCapacity // ~1087 Wh

	team.Data.Battery.TotalCapacity = totalCapacity
	team.Data.Battery.UsedCapacity = 0.0
	team.Data.Battery.EnergyWh = nominalEnergy
	team.Data.Battery.Voltage = packVoltage
	team.Data.Battery.Current = 0.0
	team.Data.Battery.DischargeRate = 0.0
	team.Data.Battery.SoC = 100.0
	team.Data.Battery.Temp = 25.0
	team.Data.Battery.Health = 100
	team.Data.Battery.InternalRes = 50.0 // mOhm (paket toplam)
	team.Data.Battery.CellVoltageMin = initialCellVoltage
	team.Data.Battery.CellVoltageMax = initialCellVoltage

	// 14 hücre voltajları (başlangıçta dengeli)
	team.Data.Battery.CellBalance = make([]float64, seriesCells)
	for i := 0; i < seriesCells; i++ {
		// Hafif dengesizlik simülasyonu (±0.02V)
		offset := (float64(i%3) - 1) * 0.01
		team.Data.Battery.CellBalance[i] = initialCellVoltage + offset
	}

	// Motor initialization
	team.Data.Motor.MaxSpeed = 37.7
	team.Data.Motor.Speed = 0.0
	team.Data.Motor.RPM = 0
	team.Data.Motor.Torque = 0.0
	team.Data.Motor.Current = 0.0
	team.Data.Motor.Voltage = 0.0
	team.Data.Motor.PowerW = 0.0
	team.Data.Motor.Efficiency = 95.0
	team.Data.Motor.Temp = 22.0
	team.Data.Motor.Health = 100

	fmt.Printf("%s has been initialized with data of: \n", team.Data.TeamName)
	fmt.Printf(" Motor Health %d \n", team.Data.Motor.Health)
	fmt.Printf(" Battery Health %d\n", team.Data.Battery.Health)
	fmt.Printf(" Action Count %d\n", team.Data.Stats.ActionCount)
	fmt.Printf(" Total Packets Sent %d\n", team.Data.Stats.TotalPackets)

}

// Throttle değerini 0-100 arasında tutar (yüzde olarak)
// Gerçek akım SimulateCar'da dinamik olarak hesaplanır
var throttlePercent float64 = 0

// Mevcut hızda çekilebilecek maksimum akım (web arayüzü için export)
var maxAvailableCurrent float64 = 40.0

func GetMaxAvailableCurrent() float64 {
	return maxAvailableCurrent
}

func (team *TeamData) ProcessActions(action string, payload int) {
	var err error

	switch action {
	case "accelerate":
		team.Data.Motor.RPM += payload
	case "decelerate":
		team.Data.Motor.RPM -= payload
		if team.Data.Motor.RPM < 0 {
			team.Data.Motor.RPM = 0
		}

	case "set_throttle":
		// Throttle sadece yüzde olarak saklanır (0-100)
		// Gerçek akım hıza göre SimulateCar'da hesaplanacak
		throttlePercent = float64(payload)
		if throttlePercent > 100 {
			throttlePercent = 100
		}
		if throttlePercent < 0 {
			throttlePercent = 0
		}

	default:
		err = fmt.Errorf("unknown action: %s", action)

	}
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}

}

// TimeMultiplier getter fonksiyonu (web paketinden alınacak)
var GetTimeMultiplier func() float64

func (team *TeamData) SimulateCar() {
	// Simülasyon hız çarpanı
	timeMultiplier := 1.0
	if GetTimeMultiplier != nil {
		timeMultiplier = GetTimeMultiplier()
	}

	//Mekanik Simülasyon Sabitleri (Solar/Efficiency Araç)
	vehicleMass := 170.0          // kg (araç + sürücü)
	wheelRadius := 0.20           // metre (16" jant + lastik)
	airDragCoeff := 0.10          // Cd (solar araç - çok aerodinamik)
	airDensity := 1.225           // kg/m^3
	frontalArea := 0.5            // m^2 (dar, alçak solar araç gövdesi)
	rollingResCoeff := 0.005      // Crr (Michelin solar tire - çok düşük)
	gravity := 9.81               // m/s^2
	baseDt := 0.05                // saniye, temel zaman adımı (50ms tick)
	dt := baseDt * timeMultiplier // Hızlandırılmış zaman adımı

	// Motor sabitleri (Mitsuba M2096-III)
	const Kt = 0.48         // Nm/A, tork sabiti
	const Ke = 0.48         // V/(rad/s), back-EMF sabiti (genelde Kt = Ke)
	const maxCurrent = 40.0 // A, kontrolcü akım limiti
	maxRPM := 518.0         // Mitsuba Max RPM
	maxTorque := 30.0       // Nm, maksimum tork
	noLoadRPM := 600.0      // Yüksüz maksimum RPM (back-EMF = batarya gerilimi olduğunda)

	// Mevcut hız (m/s)
	speedMS := (team.Data.Motor.Speed * 1000) / 3600

	// RPM'den açısal hız (rad/s)
	wheelCirc := 2 * math.Pi * wheelRadius
	currentRPM := (speedMS / wheelCirc) * 60
	angularVelocity := (currentRPM * 2 * math.Pi) / 60 // rad/s

	// ============================================
	// GEREKEN TORK HESABI (Dirençlere göre)
	// ============================================
	// Önce mevcut dirençleri hesapla
	rollingResistance := rollingResCoeff * vehicleMass * gravity
	airDrag := 0.5 * airDragCoeff * airDensity * frontalArea * speedMS * speedMS
	totalResistance := rollingResistance + airDrag

	// Sabit hız için gereken tork (sadece dirençleri yenmek için)
	torqueToMaintain := totalResistance * wheelRadius

	// ============================================
	// HIZ-TORK KARAKTERİSTİĞİ (Lineer Model)
	// ============================================
	// DC/BLDC motorlarda klasik tork-hız eğrisi:
	// - 0 RPM'de maksimum tork (stall torque)
	// - Maksimum RPM'de 0 tork (no-load speed)
	// Bu lineer ilişki: T = T_max * (1 - RPM/RPM_noload)
	//
	// Akım için: I = I_max * (1 - RPM/RPM_noload)
	// Yani hız arttıkça çekilebilecek maksimum akım DÜŞER

	speedRatio := currentRPM / noLoadRPM
	if speedRatio > 1.0 {
		speedRatio = 1.0
	}
	if speedRatio < 0 {
		speedRatio = 0
	}

	// Hıza göre maksimum çekilebilecek akım (lineer düşüş)
	// 0 RPM -> 40A max
	// noLoadRPM -> 0A max
	localMaxCurrent := maxCurrent * (1.0 - speedRatio)
	maxAvailableCurrent = localMaxCurrent // Global değişkene ata (web için)

	// ============================================
	// AKILLI AKAM HESABI
	// ============================================
	// Sabit hız için gereken akım (bilgi amaçlı)
	currentToMaintain := torqueToMaintain / Kt
	if currentToMaintain < 0 {
		currentToMaintain = 0
	}

	var actualCurrent float64

	if throttlePercent <= 0 {
		// Throttle kapalı - motor freni / coasting
		actualCurrent = 0
	} else {
		// Throttle yüzdesine göre istenen akım
		// %100 throttle = mevcut hızda çekilebilecek maksimum akım
		// Düşük throttle = düşük akım = yavaşlama mümkün
		throttleFactor := throttlePercent / 100.0
		actualCurrent = localMaxCurrent * throttleFactor

		// Fiziksel limitler
		if actualCurrent > localMaxCurrent {
			actualCurrent = localMaxCurrent
		}
		if actualCurrent < 0 {
			actualCurrent = 0
		}
	}

	// Hızı korumak için gereken akım yüzdesi (UI için bilgi)
	maintainThrottlePercent := 0.0
	if localMaxCurrent > 0 {
		maintainThrottlePercent = (currentToMaintain / localMaxCurrent) * 100
	}
	_ = maintainThrottlePercent // TODO: WebSocket ile gönder

	// Back-EMF hesabı
	backEMF := Ke * angularVelocity

	// ============================================
	// TORK HESABI
	// ============================================
	torque := Kt * actualCurrent
	if torque > maxTorque {
		torque = maxTorque
	}

	team.Data.Motor.Torque = torque
	team.Data.Motor.Current = actualCurrent

	// ============================================
	// VERİMLİLİK HESABI (Mitsuba M2096-III)
	// ============================================
	// Mitsuba hub motorlar %90-95 verimlilik aralığında çalışır
	// Verimlilik eğrisi: düşük yükte düşük, orta yükte maksimum, yüksek yükte hafif düşüş
	loadFactor := actualCurrent / maxCurrent
	speedFactor := currentRPM / maxRPM

	// Mitsuba verimlilik modeli:
	// - Çok düşük yük (<10%): ~88%
	// - Düşük yük (10-30%): ~90-92%
	// - Orta yük (30-70%): ~93-95% (optimum bölge)
	// - Yüksek yük (>70%): ~91-93%
	var efficiency float64

	if loadFactor < 0.05 {
		// Neredeyse yüksüz - düşük verim
		efficiency = 0.85
	} else if loadFactor < 0.15 {
		// Çok düşük yük
		efficiency = 0.88 + loadFactor*0.2
	} else if loadFactor < 0.30 {
		// Düşük yük - verim artıyor
		efficiency = 0.90 + (loadFactor-0.15)*0.2
	} else if loadFactor < 0.70 {
		// Optimum bölge - %93-95
		efficiency = 0.93 + (0.5-math.Abs(loadFactor-0.5))*0.04
	} else {
		// Yüksek yük - hafif düşüş
		efficiency = 0.93 - (loadFactor-0.70)*0.1
	}

	// Hız faktörü - çok düşük RPM'de verim biraz düşer
	if speedFactor < 0.2 {
		efficiency *= (0.95 + speedFactor*0.25) // %95-100 arası çarpan
	}

	// Sınırlar
	if efficiency < 0.85 {
		efficiency = 0.85
	}
	if efficiency > 0.95 {
		efficiency = 0.95
	}
	team.Data.Motor.Efficiency = efficiency * 100

	// ============================================
	// GÜÇ HESABI
	// ============================================
	// Mekanik güç = Tork * Açısal hız
	mechanicalPower := torque * angularVelocity
	// Elektriksel güç = Mekanik güç / Verimlilik
	electricalPower := mechanicalPower / efficiency
	team.Data.Motor.PowerW = electricalPower
	team.Data.Motor.Voltage = backEMF + (actualCurrent * 0.1) // Yaklaşık motor gerilimi

	// ============================================
	// HAREKET FİZİĞİ
	// ============================================
	// Torktan kuvvet
	force := torque / wheelRadius // N

	// Net kuvvet
	netForce := force - totalResistance

	// İvme
	acceleration := netForce / vehicleMass

	// Hız güncelle
	speedMS += acceleration * dt

	// Negatif hız olmasın
	if speedMS < 0 {
		speedMS = 0
	}

	// Maksimum hız limiti
	maxSpeedMS := team.Data.Motor.MaxSpeed * 1000 / 3600
	if speedMS > maxSpeedMS {
		speedMS = maxSpeedMS
	}

	// km/h'ya çevir
	team.Data.Motor.Speed = speedMS * 3.6

	// RPM güncelle
	team.Data.Motor.RPM = int((speedMS / wheelCirc) * 60)

	// ============================================
	// BATARYA SİMÜLASYONU (14S7P)
	// ============================================
	totalCapacityAh := 21.0 // 7P × 3Ah = 21Ah
	seriesCells := 14
	cellVoltageMax := 4.20                                   // V
	cellVoltageMin := 3.00                                   // V
	internalResOhm := team.Data.Battery.InternalRes / 1000.0 // mOhm -> Ohm

	// Akımı kaydet
	team.Data.Battery.Current = actualCurrent

	// C-rate hesapla (akım / kapasite)
	cRate := actualCurrent / totalCapacityAh
	team.Data.Battery.DischargeRate = cRate

	// Kullanılan kapasite güncelle (Ah)
	// current (A) × dt (s) / 3600 (s/h) = Ah
	usedAh := actualCurrent * dt / 3600.0
	team.Data.Battery.UsedCapacity += usedAh

	// Kalan kapasite
	remainingCapacity := totalCapacityAh - team.Data.Battery.UsedCapacity
	if remainingCapacity < 0 {
		remainingCapacity = 0
	}

	// SoC hesapla (%)
	team.Data.Battery.SoC = (remainingCapacity / totalCapacityAh) * 100.0

	// Hücre voltajı hesapla (SoC'a göre lineer yaklaşım)
	// Gerçekte discharge curve non-linear, basitleştirilmiş model
	socFraction := team.Data.Battery.SoC / 100.0

	// Lineer: V = Vmin + (Vmax - Vmin) × SoC
	openCircuitVoltage := cellVoltageMin + (cellVoltageMax-cellVoltageMin)*socFraction

	// Yük altında voltaj düşümü (İç direnç etkisi)
	// V_load = V_oc - I × R
	voltageDrop := actualCurrent * internalResOhm
	loadVoltage := openCircuitVoltage*float64(seriesCells) - voltageDrop

	if loadVoltage < cellVoltageMin*float64(seriesCells) {
		loadVoltage = cellVoltageMin * float64(seriesCells)
	}

	team.Data.Battery.Voltage = loadVoltage

	// Enerji güncelle (Wh)
	// Kullanılan enerji: P × t = V × I × t
	usedEnergy := loadVoltage * actualCurrent * dt / 3600.0
	team.Data.Battery.EnergyWh -= usedEnergy
	if team.Data.Battery.EnergyWh < 0 {
		team.Data.Battery.EnergyWh = 0
	}

	// Varyasyon parametrelerini al (min/medium/high)
	varParams := GetVariationParams()

	// Hücre voltajları güncelle (gerçekçi dengesizlik modeli)
	if len(team.Data.Battery.CellBalance) == seriesCells {
		minCellV := 9999.0
		maxCellV := 0.0

		for i := 0; i < seriesCells; i++ {
			// Her hücrenin farklı kapasitesi var (üretim toleransı + yaşlanma)
			// Bazı hücreler daha hızlı deşarj olur
			cellCapacityFactor := 1.0 - float64(i%5)*varParams.CapacityVariation

			// Hücre voltajı = Genel voltaj × kapasite faktörü × rastgele varyasyon
			randomVariation := 1.0 + (float64((i*7)%11)-5.0)*varParams.RandomVariation
			cellV := openCircuitVoltage * cellCapacityFactor * randomVariation

			// Yük altında farklı iç direnç etkisi (zayıf hücreler daha çok düşer)
			cellInternalRes := internalResOhm / float64(seriesCells) * (1.0 + float64(i%4)*varParams.ResistanceVariation)
			cellVoltageDrop := actualCurrent * cellInternalRes
			cellV -= cellVoltageDrop

			if cellV < cellVoltageMin {
				cellV = cellVoltageMin
			}
			if cellV > cellVoltageMax {
				cellV = cellVoltageMax
			}
			team.Data.Battery.CellBalance[i] = cellV

			if cellV < minCellV {
				minCellV = cellV
			}
			if cellV > maxCellV {
				maxCellV = cellV
			}
		}
		team.Data.Battery.CellVoltageMin = minCellV
		team.Data.Battery.CellVoltageMax = maxCellV

		// ============================================
		// BALANS ETKİSİ: Performans ve Verim
		// ============================================
		cellDelta := maxCellV - minCellV // Hücreler arası voltaj farkı

		// 1. Akım Sınırlaması (BMS koruma)
		// Delta > 0.1V ise BMS akımı sınırlamaya başlar
		// Delta > 0.3V ise ciddi sınırlama
		if cellDelta > 0.1 {
			// Akım sınırlama faktörü: 0.1V delta = %100, 0.3V delta = %50
			balanceCurrentLimit := 1.0 - (cellDelta-0.1)*2.5
			if balanceCurrentLimit < 0.5 {
				balanceCurrentLimit = 0.5 // Minimum %50 akım izni
			}
			if balanceCurrentLimit > 1.0 {
				balanceCurrentLimit = 1.0
			}
			// Global max current'ı etkile (bir sonraki tick'te)
			maxAvailableCurrent *= balanceCurrentLimit
		}

		// 2. Verim Düşüşü (dengesiz akım dağılımı)
		// Her 0.05V delta için %1 verim kaybı
		balanceEfficiencyLoss := cellDelta * 20 // 0.05V = %1 kayıp
		if balanceEfficiencyLoss > 10 {
			balanceEfficiencyLoss = 10 // Max %10 kayıp
		}
		team.Data.Motor.Efficiency -= balanceEfficiencyLoss
		if team.Data.Motor.Efficiency < 75 {
			team.Data.Motor.Efficiency = 75
		}

		// 3. Kullanılabilir Kapasite (en zayıf hücre belirler)
		// Min hücre voltajı 3.0V'a düştüğünde paket boş sayılır
		if minCellV <= cellVoltageMin+0.1 {
			// Batarya neredeyse boş - güç sınırla
			lowCellFactor := (minCellV - cellVoltageMin) / 0.1
			if lowCellFactor < 0 {
				lowCellFactor = 0
			}
			maxAvailableCurrent *= lowCellFactor
		}
	}

	// Sıcaklık simülasyonu
	// Isı üretimi: P = I² × R
	heatGenerated := actualCurrent * actualCurrent * internalResOhm * dt // Joule
	// Soğuma (ortama ısı transferi)
	ambientTemp := 25.0
	coolingRate := 0.001 // Basit soğuma katsayısı
	cooling := (team.Data.Battery.Temp - ambientTemp) * coolingRate * dt

	// Sıcaklık değişimi (basit termal model)
	// Batarya termal kapasitesi ~1000 J/K (yaklaşık)
	thermalMass := 1000.0
	tempChange := heatGenerated/thermalMass - cooling
	team.Data.Battery.Temp += tempChange

	// Sıcaklık limitleri
	if team.Data.Battery.Temp < ambientTemp {
		team.Data.Battery.Temp = ambientTemp
	}
	if team.Data.Battery.Temp > 60.0 {
		team.Data.Battery.Temp = 60.0 // Max güvenli sıcaklık
	}

	// ============================================
	// PİST KONUM TAKİBİ (Silesia Ring - 4200m)
	// ============================================
	trackLength := 4200.0 // Silesia Ring pist uzunluğu (metre)

	// Gidilen mesafeyi güncelle (m/s * dt saniye)
	distanceTraveled := speedMS * dt
	team.Data.RaceStatus.Position += distanceTraveled

	// Tur tamamlandı mı kontrol et
	if team.Data.RaceStatus.Position >= trackLength {
		team.Data.RaceStatus.Lap++
		team.Data.RaceStatus.Position -= trackLength // Kalan mesafeyi yeni tura aktar
	}

	fmt.Printf("%.2f km/h | %d RPM | %.2f Nm tork | %.2f A akım (max: %.1fA) | Tur %d - %.0fm\n",
		team.Data.Motor.Speed, team.Data.Motor.RPM, team.Data.Motor.Torque,
		team.Data.Motor.Current, maxAvailableCurrent, team.Data.RaceStatus.Lap+1, team.Data.RaceStatus.Position)
}
