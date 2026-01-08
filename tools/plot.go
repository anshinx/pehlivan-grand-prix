package main

import (
	"encoding/csv"
	"fmt"
	"image/color"
	"os"
	"strconv"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
)

type PlotData struct {
	Time           []float64
	Speed          []float64
	RPM            []float64
	MotorTemp      []float64
	BatteryTemp    []float64
	BatterySoC     []float64
	MotorPower     []float64
	BatteryCurrent []float64
}

func main() {
	// CSV'den verileri oku
	data, err := readCSV("simulation_data.csv")
	if err != nil {
		fmt.Printf("CSV okunamadı: %v\n", err)
		return
	}

	fmt.Printf("✓ %d veri noktası yüklendi\n", len(data.Time))

	// Grafikleri oluştur
	if err := createSpeedPlot(data); err != nil {
		fmt.Printf("Hız grafiği hatası: %v\n", err)
	} else {
		fmt.Println("✓ speed_plot.png oluşturuldu")
	}

	if err := createTemperaturePlot(data); err != nil {
		fmt.Printf("Sıcaklık grafiği hatası: %v\n", err)
	} else {
		fmt.Println("✓ temperature_plot.png oluşturuldu")
	}

	if err := createBatteryPlot(data); err != nil {
		fmt.Printf("Batarya grafiği hatası: %v\n", err)
	} else {
		fmt.Println("✓ battery_plot.png oluşturuldu")
	}

	if err := createPowerPlot(data); err != nil {
		fmt.Printf("Güç grafiği hatası: %v\n", err)
	} else {
		fmt.Println("✓ power_plot.png oluşturuldu")
	}

	fmt.Println("\n✓ Tüm grafikler oluşturuldu!")
}

func readCSV(filename string) (*PlotData, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	data := &PlotData{}

	// Skip header
	for i := 1; i < len(records); i++ {
		time, _ := strconv.ParseFloat(records[i][0], 64)
		speed, _ := strconv.ParseFloat(records[i][1], 64)
		rpm, _ := strconv.ParseFloat(records[i][2], 64)
		motorTemp, _ := strconv.ParseFloat(records[i][3], 64)
		batteryTemp, _ := strconv.ParseFloat(records[i][4], 64)
		batterySoC, _ := strconv.ParseFloat(records[i][5], 64)
		motorPower, _ := strconv.ParseFloat(records[i][6], 64)
		batteryCurrent, _ := strconv.ParseFloat(records[i][7], 64)

		data.Time = append(data.Time, time)
		data.Speed = append(data.Speed, speed)
		data.RPM = append(data.RPM, rpm)
		data.MotorTemp = append(data.MotorTemp, motorTemp)
		data.BatteryTemp = append(data.BatteryTemp, batteryTemp)
		data.BatterySoC = append(data.BatterySoC, batterySoC)
		data.MotorPower = append(data.MotorPower, motorPower)
		data.BatteryCurrent = append(data.BatteryCurrent, batteryCurrent)
	}

	return data, nil
}

func createSpeedPlot(data *PlotData) error {
	p := plot.New()
	p.Title.Text = "Hız ve RPM - Zaman"
	p.X.Label.Text = "Zaman (s)"
	p.Y.Label.Text = "Hız (km/h)"

	// Speed line
	speedPoints := make(plotter.XYs, len(data.Time))
	for i := range data.Time {
		speedPoints[i].X = data.Time[i]
		speedPoints[i].Y = data.Speed[i]
	}

	speedLine, err := plotter.NewLine(speedPoints)
	if err != nil {
		return err
	}
	speedLine.Color = color.RGBA{R: 255, A: 255}
	speedLine.Width = vg.Points(2)
	p.Add(speedLine)
	p.Legend.Add("Hız", speedLine)

	// RPM line (secondary scale - divided by 100 to fit)
	rpmPoints := make(plotter.XYs, len(data.Time))
	for i := range data.Time {
		rpmPoints[i].X = data.Time[i]
		rpmPoints[i].Y = data.RPM[i] / 100.0 // Scale down for visibility
	}

	rpmLine, err := plotter.NewLine(rpmPoints)
	if err != nil {
		return err
	}
	rpmLine.Color = color.RGBA{B: 255, A: 255}
	rpmLine.Width = vg.Points(2)
	p.Add(rpmLine)
	p.Legend.Add("RPM/100", rpmLine)

	return p.Save(8*vg.Inch, 6*vg.Inch, "speed_plot.png")
}

func createTemperaturePlot(data *PlotData) error {
	p := plot.New()
	p.Title.Text = "Sıcaklık Değişimi"
	p.X.Label.Text = "Zaman (s)"
	p.Y.Label.Text = "Sıcaklık (°C)"

	// Motor temperature
	motorTempPoints := make(plotter.XYs, len(data.Time))
	for i := range data.Time {
		motorTempPoints[i].X = data.Time[i]
		motorTempPoints[i].Y = data.MotorTemp[i]
	}

	motorLine, err := plotter.NewLine(motorTempPoints)
	if err != nil {
		return err
	}
	motorLine.Color = color.RGBA{R: 255, G: 100, A: 255}
	motorLine.Width = vg.Points(2)
	p.Add(motorLine)
	p.Legend.Add("Motor", motorLine)

	// Battery temperature
	batteryTempPoints := make(plotter.XYs, len(data.Time))
	for i := range data.Time {
		batteryTempPoints[i].X = data.Time[i]
		batteryTempPoints[i].Y = data.BatteryTemp[i]
	}

	batteryLine, err := plotter.NewLine(batteryTempPoints)
	if err != nil {
		return err
	}
	batteryLine.Color = color.RGBA{G: 200, B: 255, A: 255}
	batteryLine.Width = vg.Points(2)
	p.Add(batteryLine)
	p.Legend.Add("Batarya", batteryLine)

	return p.Save(8*vg.Inch, 6*vg.Inch, "temperature_plot.png")
}

func createBatteryPlot(data *PlotData) error {
	p := plot.New()
	p.Title.Text = "Batarya Durumu"
	p.X.Label.Text = "Zaman (s)"
	p.Y.Label.Text = "SoC (%)"

	// SoC line
	socPoints := make(plotter.XYs, len(data.Time))
	for i := range data.Time {
		socPoints[i].X = data.Time[i]
		socPoints[i].Y = data.BatterySoC[i]
	}

	socLine, err := plotter.NewLine(socPoints)
	if err != nil {
		return err
	}
	socLine.Color = color.RGBA{G: 255, A: 255}
	socLine.Width = vg.Points(2)
	p.Add(socLine)
	p.Legend.Add("SoC", socLine)

	return p.Save(8*vg.Inch, 6*vg.Inch, "battery_plot.png")
}

func createPowerPlot(data *PlotData) error {
	p := plot.New()
	p.Title.Text = "Güç ve Akım"
	p.X.Label.Text = "Zaman (s)"
	p.Y.Label.Text = "Güç (W) / Akım (A×100)"

	// Power line
	powerPoints := make(plotter.XYs, len(data.Time))
	for i := range data.Time {
		powerPoints[i].X = data.Time[i]
		powerPoints[i].Y = data.MotorPower[i]
	}

	powerLine, err := plotter.NewLine(powerPoints)
	if err != nil {
		return err
	}
	powerLine.Color = color.RGBA{R: 255, G: 165, A: 255}
	powerLine.Width = vg.Points(2)
	p.Add(powerLine)
	p.Legend.Add("Motor Güç", powerLine)

	// Current line (scaled x100)
	currentPoints := make(plotter.XYs, len(data.Time))
	for i := range data.Time {
		currentPoints[i].X = data.Time[i]
		currentPoints[i].Y = data.BatteryCurrent[i] * 100.0
	}

	currentLine, err := plotter.NewLine(currentPoints)
	if err != nil {
		return err
	}
	currentLine.Color = color.RGBA{R: 100, G: 100, B: 255, A: 255}
	currentLine.Width = vg.Points(2)
	p.Add(currentLine)
	p.Legend.Add("Akım × 100", currentLine)

	return p.Save(8*vg.Inch, 6*vg.Inch, "power_plot.png")
}
