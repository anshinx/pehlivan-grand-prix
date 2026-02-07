package models

type TeamData struct {
	TeamName      string      `json:"name"`
	Battery       BatteryInfo `json:"bat"`
	Motor         MotorInfo   `json:"motor"`
	RaceStatus    RaceInfo    `json:"race"`
	Environment   EnvInfo     `json:"env"`
	Stats         KupaStats   `json:"stats"` // Ödül takibi için
	IsInitialzied bool
}

type BatteryInfo struct {
	SoC           float64   `json:"soc"`            // % (State of Charge)
	EnergyWh      float64   `json:"wh"`             // Kalan Watt-saat
	Voltage       float64   `json:"voltage"`        // V (Paket gerilimi)
	Current       float64   `json:"current"`        // A (Akım çekişi)
	Temp          float64   `json:"temp"`           // °C
	Health        int       `json:"battery-health"` // %
	CellBalance   []float64 `json:"balance"`        // 14 hücre voltajları
	DischargeRate float64   `json:"discharge-rate"` // C-rate (1C, 2C, vb.)
	// Yeni alanlar
	CellVoltageMin float64 `json:"cellMin"`       // En düşük hücre voltajı
	CellVoltageMax float64 `json:"cellMax"`       // En yüksek hücre voltajı
	InternalRes    float64 `json:"internalRes"`   // İç direnç (mOhm)
	TotalCapacity  float64 `json:"totalCapacity"` // Toplam kapasite (Ah)
	UsedCapacity   float64 `json:"usedCapacity"`  // Kullanılan kapasite (Ah)
}

type MotorInfo struct {
	MaxSpeed   float64
	Speed      float64 `json:"speed"`        // km/h
	RPM        int     `json:"rpm"`          // Devir/dakika
	Torque     float64 `json:"torque"`       // Nm (Newton-metre)
	Current    float64 `json:"current"`      // A (Motor akımı)
	Voltage    float64 `json:"voltage"`      // V (Motor gerilimi)
	PowerW     float64 `json:"power"`        // W (Anlık güç tüketimi)
	Efficiency float64 `json:"efficiency"`   // % (Motor verimi)
	Temp       float64 `json:"temp"`         // °C
	Health     int     `json:"motor-health"` // %
}

type RaceInfo struct {
	Lap      int     `json:"lap"` // Tamamlanan tur
	Position float64 `json:"pos"` // Tur içindeki metre (0-500)
}

type EnvInfo struct {
	Rain     bool `json:"rain"`
	Parasite bool `json:"parasite"`
}

type KupaStats struct {
	TotalPackets int64 `json:"packets"`
	ActionCount  int   `json:"actions"`
}
