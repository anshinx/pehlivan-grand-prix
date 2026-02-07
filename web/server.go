package web

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // CORS için
	},
}

// Tüm bağlı client'ları tutuyoruz
var clients = make(map[*websocket.Conn]bool)
var clientsMu sync.Mutex

// Throttle değerini global olarak tutuyoruz (engine paketi okuyacak)
var CurrentThrottle float64 = 0
var throttleMu sync.RWMutex

// Simülasyon hız çarpanı (1x, 2x, 5x, 10x vs.)
var TimeMultiplier float64 = 1.0
var timeMu sync.RWMutex

// Logging callback'leri (main.go'dan bağlanacak)
var StartLoggingCallback func()
var StopLoggingCallback func()

// AI Driver callback'leri
var SetAIEnabledCallback func(bool)
var SetAIModeCallback func(string)

// Cell Variation seviyesi (min, medium, high)
var CellVariationLevel string = "medium"
var variationMu sync.RWMutex

// Reset callback
var ResetSimulationCallback func()

// SimData websocket üzerinden gönderilecek veri yapısı
type SimData struct {
	Speed      float64 `json:"speed"`
	RPM        int     `json:"rpm"`
	Torque     float64 `json:"torque"`
	Current    float64 `json:"current"`
	MaxCurrent float64 `json:"maxCurrent"`
	Efficiency float64 `json:"efficiency"`
	Power      float64 `json:"power"`
	Throttle   float64 `json:"throttle"`
	// Batarya verileri
	BatterySOC       float64   `json:"batterySoc"`
	BatteryVoltage   float64   `json:"batteryVoltage"`
	BatteryCurrent   float64   `json:"batteryCurrent"`
	BatteryTemp      float64   `json:"batteryTemp"`
	BatteryPower     float64   `json:"batteryPower"`
	BatteryCRate     float64   `json:"batteryCRate"`
	BatteryEnergy    float64   `json:"batteryEnergy"`
	BatteryCellMin   float64   `json:"batteryCellMin"`
	BatteryCellMax   float64   `json:"batteryCellMax"`
	BatteryCells     []float64 `json:"batteryCells"`
	BatteryCellDelta float64   `json:"batteryCellDelta"` // Max-Min voltaj farkı
	BalanceStatus    string    `json:"balanceStatus"`    // "good", "warning", "critical"
	BalanceEffect    float64   `json:"balanceEffect"`    // Performans etkisi %
	// Motor sıcaklığı
	MotorTemp float64 `json:"motorTemp"`
	// Pist verileri
	Lap           int     `json:"lap"`
	LapPosition   float64 `json:"lapPosition"`   // Metre cinsinden tur içi konum
	TotalDistance float64 `json:"totalDistance"` // Toplam gidilen mesafe
	TrackLength   float64 `json:"trackLength"`   // Pist uzunluğu
	// Simülasyon hızı
	TimeMultiplier float64 `json:"timeMultiplier"` // Simülasyon hız çarpanı
	// Logging durumu
	IsLogging   bool   `json:"isLogging"`
	LogFilename string `json:"logFilename"`
	RecordCount int    `json:"recordCount"`
	// AI Driver durumu
	AIEnabled        bool    `json:"aiEnabled"`
	AIMode           string  `json:"aiMode"`
	AITargetSpeed    float64 `json:"aiTargetSpeed"`
	AITargetThrottle float64 `json:"aiTargetThrottle"`
	AIAvgEfficiency  float64 `json:"aiAvgEfficiency"`
	AIInstEfficiency float64 `json:"aiInstEfficiency"`
	// Pist segment bilgisi
	CurrentSegment  string  `json:"currentSegment"`
	SegmentType     string  `json:"segmentType"`
	SegmentMaxSpeed float64 `json:"segmentMaxSpeed"`
}

// Command client'dan gelen komut yapısı
type Command struct {
	Type   string  `json:"type"`
	Value  float64 `json:"value"`
	StrVal string  `json:"strVal"` // AI mode için string değer
}

func GetThrottle() float64 {
	throttleMu.RLock()
	defer throttleMu.RUnlock()
	return CurrentThrottle
}

func SetThrottle(value float64) {
	throttleMu.Lock()
	defer throttleMu.Unlock()
	if value < 0 {
		value = 0
	}
	if value > 100 {
		value = 100
	}
	CurrentThrottle = value
}

func GetTimeMultiplier() float64 {
	timeMu.RLock()
	defer timeMu.RUnlock()
	return TimeMultiplier
}

func SetTimeMultiplier(value float64) {
	timeMu.Lock()
	defer timeMu.Unlock()
	if value < 0.1 {
		value = 0.1
	}
	if value > 100 {
		value = 100
	}
	TimeMultiplier = value
}

func GetCellVariationLevel() string {
	variationMu.RLock()
	defer variationMu.RUnlock()
	return CellVariationLevel
}

func SetCellVariationLevel(level string) {
	variationMu.Lock()
	defer variationMu.Unlock()
	if level == "min" || level == "medium" || level == "high" {
		CellVariationLevel = level
	}
}

// WebSocket handler
func HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}
	defer conn.Close()

	// Client'ı listeye ekle
	clientsMu.Lock()
	clients[conn] = true
	clientsMu.Unlock()

	log.Println("Yeni client bağlandı")

	// Client'dan gelen mesajları dinle
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Printf("Read error: %v", err)
			clientsMu.Lock()
			delete(clients, conn)
			clientsMu.Unlock()
			break
		}

		var cmd Command
		if err := json.Unmarshal(message, &cmd); err != nil {
			log.Printf("JSON parse error: %v", err)
			continue
		}

		switch cmd.Type {
		case "throttle":
			SetThrottle(cmd.Value)
			log.Printf("Throttle: %.1f%%", cmd.Value)
		case "speed":
			SetTimeMultiplier(cmd.Value)
			log.Printf("Simülasyon hızı: %.1fx", cmd.Value)
		case "logging":
			if cmd.Value > 0 {
				// Logging başlat
				if StartLoggingCallback != nil {
					StartLoggingCallback()
				}
				log.Println("📝 CSV kaydı başladı")
			} else {
				// Logging durdur
				if StopLoggingCallback != nil {
					StopLoggingCallback()
				}
				log.Println("📝 CSV kaydı durduruldu")
			}
		case "ai_enable":
			if SetAIEnabledCallback != nil {
				enabled := cmd.Value > 0
				SetAIEnabledCallback(enabled)
				if enabled {
					log.Println("🤖 AI Sürücü aktif")
				} else {
					log.Println("🤖 AI Sürücü devre dışı")
				}
			}
		case "ai_mode":
			if SetAIModeCallback != nil && cmd.StrVal != "" {
				SetAIModeCallback(cmd.StrVal)
				log.Printf("🤖 AI Mod: %s", cmd.StrVal)
			}
		case "cell_variation":
			if cmd.StrVal != "" {
				SetCellVariationLevel(cmd.StrVal)
				log.Printf("🔋 Hücre varyasyonu: %s", cmd.StrVal)
			}
		case "reset":
			if ResetSimulationCallback != nil {
				ResetSimulationCallback()
				log.Println("🔄 Simülasyon sıfırlandı")
			}
		}
	}
}

// Tüm client'lara veri gönder
func BroadcastSimData(data SimData) {
	clientsMu.Lock()
	defer clientsMu.Unlock()

	message, err := json.Marshal(data)
	if err != nil {
		log.Printf("JSON marshal error: %v", err)
		return
	}

	for conn := range clients {
		err := conn.WriteMessage(websocket.TextMessage, message)
		if err != nil {
			log.Printf("Write error: %v", err)
			conn.Close()
			delete(clients, conn)
		}
	}
}

// HTTP sunucusunu başlat
func StartServer(port string) {
	// WebSocket endpoint
	http.HandleFunc("/ws", HandleWebSocket)

	// Statik dosyalar için
	http.Handle("/", http.FileServer(http.Dir("./web/static")))

	log.Printf("Web sunucusu başlatılıyor: http://localhost%s", port)
	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
