package engine

import (
	"math"
)

// AIDriver akıllı sürücü yapısı
type AIDriver struct {
	Enabled         bool
	TargetSpeed     float64 // Hedef hız (km/h)
	TargetThrottle  float64 // Hesaplanan throttle (0-100)
	Mode            string  // "efficiency", "performance", "balanced"
	LookAheadMeters float64 // Viraj için ileri bakış mesafesi

	// PID kontrolcü parametreleri
	Kp float64 // Proportional gain
	Ki float64 // Integral gain
	Kd float64 // Derivative gain

	// PID iç değişkenleri
	lastError   float64
	integral    float64
	lastSpeed   float64
	smoothSpeed float64

	// Verimlilik takibi
	TotalEnergy    float64 // Toplam harcanan enerji (Wh)
	TotalDistance  float64 // Toplam mesafe (m)
	AvgEfficiency  float64 // Ortalama verimlilik
	InstEfficiency float64 // Anlık verimlilik (Wh/km)

	// Strateji
	BrakingDistance  float64 // Frenleme mesafesi (m)
	AccelSmoothness  float64 // Hızlanma yumuşaklığı (0-1)
	CoastingDistance float64 // Coasting mesafesi
}

// Singleton AI Driver instance
var aiDriver *AIDriver

// GetAIDriver singleton AI driver döndürür
func GetAIDriver() *AIDriver {
	if aiDriver == nil {
		aiDriver = &AIDriver{
			Enabled:          false,
			Mode:             "efficiency",
			LookAheadMeters:  150.0, // 150m ileri bak
			Kp:               2.5,
			Ki:               0.1,
			Kd:               0.8,
			AccelSmoothness:  0.7,
			BrakingDistance:  80.0,
			CoastingDistance: 50.0,
		}
	}
	return aiDriver
}

// SetEnabled AI sürücüyü etkinleştirir/devre dışı bırakır
func (ai *AIDriver) SetEnabled(enabled bool) {
	ai.Enabled = enabled
	if enabled {
		// Reset PID
		ai.lastError = 0
		ai.integral = 0
		ai.TotalEnergy = 0
		ai.TotalDistance = 0
	}
}

// SetMode AI sürüş modunu ayarlar
func (ai *AIDriver) SetMode(mode string) {
	ai.Mode = mode
	switch mode {
	case "efficiency":
		ai.Kp = 2.0
		ai.Ki = 0.05
		ai.Kd = 1.0
		ai.AccelSmoothness = 0.8
		ai.LookAheadMeters = 200.0
		ai.BrakingDistance = 100.0
		ai.CoastingDistance = 80.0
	case "performance":
		ai.Kp = 4.0
		ai.Ki = 0.2
		ai.Kd = 0.5
		ai.AccelSmoothness = 0.4
		ai.LookAheadMeters = 100.0
		ai.BrakingDistance = 50.0
		ai.CoastingDistance = 20.0
	case "balanced":
		ai.Kp = 3.0
		ai.Ki = 0.1
		ai.Kd = 0.7
		ai.AccelSmoothness = 0.6
		ai.LookAheadMeters = 150.0
		ai.BrakingDistance = 70.0
		ai.CoastingDistance = 40.0
	}
}

// CalculateThrottle ana AI hesaplama fonksiyonu
func (ai *AIDriver) CalculateThrottle(currentSpeed, position float64, dt float64) float64 {
	if !ai.Enabled {
		return -1 // AI kapalı, manuel kontrol
	}

	// ============================================
	// 1. HEDEF HIZ HESAPLAMA
	// ============================================
	targetSpeed := ai.calculateTargetSpeed(currentSpeed, position)
	ai.TargetSpeed = targetSpeed

	// ============================================
	// 2. HIZLANMA / FRENLEME STRATEJİSİ
	// ============================================
	// Hız yumuşatma (gürültü filtreleme)
	ai.smoothSpeed = ai.smoothSpeed*0.9 + currentSpeed*0.1
	smoothError := targetSpeed - ai.smoothSpeed

	// PID kontrolcü
	ai.integral += smoothError * dt
	// Anti-windup
	if ai.integral > 50 {
		ai.integral = 50
	}
	if ai.integral < -50 {
		ai.integral = -50
	}

	derivative := (smoothError - ai.lastError) / dt
	ai.lastError = smoothError

	// PID çıktısı
	pidOutput := ai.Kp*smoothError + ai.Ki*ai.integral + ai.Kd*derivative

	// ============================================
	// 3. VERİMLİLİK OPTİMİZASYONU
	// ============================================
	var throttle float64

	switch ai.Mode {
	case "efficiency":
		throttle = ai.efficiencyOptimizedThrottle(currentSpeed, targetSpeed, pidOutput)
	case "performance":
		throttle = ai.performanceThrottle(pidOutput)
	default:
		throttle = ai.balancedThrottle(currentSpeed, targetSpeed, pidOutput)
	}

	// Throttle limitlerini uygula
	if throttle > 100 {
		throttle = 100
	}
	if throttle < 0 {
		throttle = 0
	}

	// Yumuşak geçiş (ani throttle değişimlerini önle)
	maxChange := (1.0 - ai.AccelSmoothness) * 50 * dt * 20 // Max değişim/tick
	if maxChange < 2 {
		maxChange = 2
	}
	if throttle-ai.TargetThrottle > maxChange {
		throttle = ai.TargetThrottle + maxChange
	}
	if ai.TargetThrottle-throttle > maxChange {
		throttle = ai.TargetThrottle - maxChange
	}

	ai.TargetThrottle = throttle
	ai.lastSpeed = currentSpeed

	return throttle
}

// calculateTargetSpeed viraj ve pist durumuna göre hedef hız hesaplar
func (ai *AIDriver) calculateTargetSpeed(currentSpeed, position float64) float64 {
	// Mevcut segment
	currentSeg := GetSegmentAtPosition(position)
	currentMaxSpeed := currentSeg.MaxSpeed

	// İleriyi tara
	upcomingMinSpeed := GetMinSpeedAhead(position, ai.LookAheadMeters)

	// Frenleme mesafesi hesabı
	// v² = v₀² - 2as => s = (v₀² - v²) / (2a)
	// Tipik fren ivmesi: 2-3 m/s² (konforlu frenleme)
	brakingDecel := 2.5 // m/s²

	currentSpeedMS := currentSpeed / 3.6
	upcomingSpeedMS := upcomingMinSpeed / 3.6

	if currentSpeedMS > upcomingSpeedMS {
		// Frenleme mesafesi hesapla
		brakingDist := (currentSpeedMS*currentSpeedMS - upcomingSpeedMS*upcomingSpeedMS) / (2 * brakingDecel)

		// Segment mesafesini bul
		distToCorner := ai.distanceToNextCorner(position, upcomingMinSpeed)

		// Coasting + braking mesafesi
		totalStoppingDist := brakingDist + ai.CoastingDistance

		if distToCorner <= totalStoppingDist {
			// Yavaşlamaya başla
			// Kademeli hedef hız azaltma
			progressRatio := 1.0 - (distToCorner / totalStoppingDist)
			if progressRatio < 0 {
				progressRatio = 0
			}
			if progressRatio > 1 {
				progressRatio = 1
			}

			targetSpeed := currentSpeed - (currentSpeed-upcomingMinSpeed)*progressRatio
			return targetSpeed
		}
	}

	// Mevcut segment hızında devam et
	return currentMaxSpeed
}

// distanceToNextCorner önümüzdeki viraja olan mesafe
func (ai *AIDriver) distanceToNextCorner(position float64, targetMaxSpeed float64) float64 {
	pos := position
	for pos >= TrackLength {
		pos -= TrackLength
	}

	for i := 0; i < len(SilesiaRingTrack)*2; i++ { // 2 tur tara (wrap-around için)
		idx := i % len(SilesiaRingTrack)
		seg := SilesiaRingTrack[idx]

		segStart := seg.StartPos + float64(i/len(SilesiaRingTrack))*TrackLength

		if segStart > pos {
			if seg.MaxSpeed <= targetMaxSpeed && seg.Type != "straight" {
				return segStart - pos
			}
		}
		if i > len(SilesiaRingTrack) {
			break
		}
	}
	return ai.LookAheadMeters // Viraj bulunamadı
}

// efficiencyOptimizedThrottle verimlilik odaklı throttle hesabı
func (ai *AIDriver) efficiencyOptimizedThrottle(currentSpeed, targetSpeed, pidOutput float64) float64 {
	speedError := targetSpeed - currentSpeed

	// Verimlilik bölgesi: %40-60 throttle optimal
	// Düşük throttle = düşük akım = yüksek verimlilik (motor verimlilik eğrisi)

	if speedError > 2 {
		// Hızlanma gerekli ama yumuşak
		// Optimal verimlilik için %50-70 arası throttle
		baseThrottle := 45.0 + speedError*3
		if baseThrottle > 70 {
			baseThrottle = 70 // Verimlilik için max %70
		}
		return baseThrottle
	} else if speedError < -2 {
		// Yavaşlama gerekli - coasting
		return 0 // Motor freni, enerji harcamıyoruz
	} else {
		// Hızı koru - minimum güç
		// Sürtünmeyi yenecek kadar throttle
		// Tipik olarak %20-35 arası sabit hız için yeterli
		maintainThrottle := 25.0 + currentSpeed*0.5
		if maintainThrottle > 40 {
			maintainThrottle = 40
		}
		return maintainThrottle
	}
}

// performanceThrottle performans odaklı throttle
func (ai *AIDriver) performanceThrottle(pidOutput float64) float64 {
	// Agresif hızlanma
	throttle := 50 + pidOutput*2
	return throttle
}

// balancedThrottle dengeli throttle
func (ai *AIDriver) balancedThrottle(currentSpeed, targetSpeed, pidOutput float64) float64 {
	speedError := targetSpeed - currentSpeed

	if speedError > 1 {
		return 40 + pidOutput*1.5
	} else if speedError < -1 {
		return math.Max(0, 20+pidOutput)
	}
	return 30 + currentSpeed*0.4
}

// UpdateStats enerji ve verimlilik istatistiklerini günceller
func (ai *AIDriver) UpdateStats(powerW, speedMS, dt float64) {
	if !ai.Enabled {
		return
	}

	// Enerji hesabı (Wh)
	energyWh := (powerW * dt) / 3600.0
	ai.TotalEnergy += energyWh

	// Mesafe hesabı
	distanceM := speedMS * dt
	ai.TotalDistance += distanceM

	// Anlık verimlilik (Wh/km)
	if speedMS > 0.5 { // Minimum hız
		distanceKm := distanceM / 1000.0
		if distanceKm > 0 {
			ai.InstEfficiency = energyWh / distanceKm
		}
	}

	// Ortalama verimlilik
	if ai.TotalDistance > 100 { // 100m sonra hesapla
		ai.AvgEfficiency = ai.TotalEnergy / (ai.TotalDistance / 1000.0)
	}
}

// Reset AI sürücü istatistiklerini sıfırlar
func (ai *AIDriver) Reset() {
	ai.TargetSpeed = 0
	ai.TargetThrottle = 0
	ai.TotalEnergy = 0
	ai.TotalDistance = 0
	ai.AvgEfficiency = 0
	ai.InstEfficiency = 0
	ai.lastError = 0
	ai.integral = 0
}

// GetStatus AI durumunu döndürür (web arayüzü için)
func (ai *AIDriver) GetStatus() map[string]interface{} {
	return map[string]interface{}{
		"enabled":        ai.Enabled,
		"mode":           ai.Mode,
		"targetSpeed":    ai.TargetSpeed,
		"targetThrottle": ai.TargetThrottle,
		"avgEfficiency":  ai.AvgEfficiency,
		"instEfficiency": ai.InstEfficiency,
		"totalEnergy":    ai.TotalEnergy,
		"totalDistance":  ai.TotalDistance,
	}
}
