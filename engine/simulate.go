package engine

import (
	"fmt"
	"math"
	"pehlivan-grand-prix/models"
)

type TeamData struct {
	Data *models.TeamData
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
	team.Data.Battery.EnergyWh = 1317.12
	team.Data.Battery.Voltage = 58.78
	team.Data.Battery.Current = 0.0
	team.Data.Battery.DischargeRate = 0.0
	team.Data.Battery.SoC = 99.7
	team.Data.Battery.Temp = 22.0
	team.Data.Battery.Health = 100
	team.Data.Battery.CellBalance = []float64{
		4.20, 4.19, 4.20, 4.18, 4.20, 4.19, 4.20,
		4.20, 4.19, 4.18, 4.20, 4.20, 4.19, 4.20,
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

func (team *TeamData) SimulateCar() {
	//Mekanik Simülasyon Sabitleri (gemini tavsiyesi :D)
	vehicleMass := 170.0     // kg
	wheelRadius := 0.20      // metre
	airDragCoeff := 0.35     // CdA, örnek değer
	airDensity := 1.225      // kg/m^3
	frontalArea := 1.0       // m^2
	rollingResCoeff := 0.015 // Crr, örnek değer
	gravity := 9.81          // m/s^2
	dt := 1.0                // saniye, zaman adımı

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
	// 0 RPM -> 75A max
	// noLoadRPM -> 0A max
	maxAvailableCurrent := maxCurrent * (1.0 - speedRatio)

	// ============================================
	// AKILLI AKAM HESABI
	// ============================================
	// Sabit hız için gereken akım
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
		throttleFactor := throttlePercent / 100.0
		requestedCurrent := maxAvailableCurrent * throttleFactor

		// En az hızı koruyacak kadar akım çek (eğer throttle > 0 ise)
		// Ama fiziksel limitin üzerine çıkamaz
		actualCurrent = math.Max(requestedCurrent, currentToMaintain)
		if actualCurrent > maxAvailableCurrent {
			actualCurrent = maxAvailableCurrent
		}
		if actualCurrent < 0 {
			actualCurrent = 0
		}
	}

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
	// VERİMLİLİK HESABI
	// ============================================
	// Motor verimliliği hız ve yüke bağlı değişir
	loadFactor := actualCurrent / maxCurrent
	speedFactor := currentRPM / maxRPM

	// Optimum verimlilik %40-80 yük ve %50-80 hızda
	baseEfficiency := 0.95
	loadPenalty := math.Abs(loadFactor-0.6) * 0.15   // Optimum yük %60
	speedPenalty := math.Abs(speedFactor-0.65) * 0.1 // Optimum hız %65 RPM

	efficiency := baseEfficiency - loadPenalty - speedPenalty
	if efficiency < 0.70 {
		efficiency = 0.70 // Minimum verimlilik %70
	}
	if efficiency > 0.96 {
		efficiency = 0.96 // Maksimum verimlilik %96
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
	// BATARYA AKIMI GÜNCELLE
	// ============================================
	team.Data.Battery.Current = actualCurrent
	team.Data.Battery.DischargeRate = actualCurrent / (2.800 * 7) // 2.8Ah * 7P = 21Ah toplam kapasite için C-rate

	fmt.Printf("%.2f km/h | %d RPM | %.2f Nm tork | %.2f A akım (max: %.1fA) | %.1f%% verimlilik | %.1f W güç\n",
		team.Data.Motor.Speed, team.Data.Motor.RPM, team.Data.Motor.Torque,
		team.Data.Motor.Current, maxAvailableCurrent, team.Data.Motor.Efficiency, team.Data.Motor.PowerW)
}
