package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"pehlivan-grand-prix/engine"
	"pehlivan-grand-prix/models"
	"time"
)

func startPhysicsEngine(tickrate int, tickChan chan bool) {
	ticker := time.NewTicker(time.Duration(tickrate) * time.Millisecond)
	for range ticker.C {
		//channel'e sinyal gönder
		tickChan <- true
	}
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

	// 2. 4 tane takımı listeye ekle
	takimlar = append(takimlar, &engine.TeamData{
		Data: &models.TeamData{
			TeamName: "Test1",
		},
	})

	http.HandleFunc("/simdata", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(takimlar)
	})
	http.Handle("/", http.FileServer(http.Dir(".")))
	go http.ListenAndServe(":8081", nil)

	physicsTick := make(chan bool)
	go startPhysicsEngine(50, physicsTick)
	tickCount := 0

	go func() {
		for range physicsTick {
			// Her tıkta tüm listeyi gez!
			for _, t := range takimlar {
				if !t.Data.IsInitialzied {
					t.InitializeSimulation()
				}

				t.ProcessActions("set_throttle", 100)

				tickCount++
				t.SimulateCar()
			}
			fmt.Printf("\n ------- %d \n", tickCount)
		}
	}()

	select {}
}
