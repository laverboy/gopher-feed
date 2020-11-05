package main

import (
	"encoding/json"
	"log"
	"math/rand"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

const (
	HowOftenGophersLifeDrops = 2 * time.Second
	HowOftenStatusIsUpdated = 250 * time.Millisecond
)

type FeedRequest struct {
	Feed string
}

type Gopher struct {
	Name string
	Life int
}

type State struct {
	Gophers []Gopher
}

func (s *State) asJSON() []byte {
	currentLife := map[string]int{}
	for _, gopher := range s.Gophers {
		currentLife[gopher.Name] = gopher.Life
	}
	bytes, err := json.Marshal(currentLife)
	if err != nil {
		panic("unable to marshal state as map string to int") // should never happen
	}
	return bytes
}

func (s *State) randomLifeReduction() {
	randGopher := rand.Intn(len(s.Gophers))
	if s.Gophers[randGopher].Life != 0 {
		lifeReductions := []int{5, 10, 15}
		randNumber := rand.Intn(len(lifeReductions))
		newLife := s.Gophers[randGopher].Life - lifeReductions[randNumber]
		if newLife < 0 {
			newLife = 0
		}
		s.Gophers[randGopher].Life = newLife
	}
}

var upgrader = websocket.Upgrader{} // use default options

func socket(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("unable to upgrade connection to websocket:", err)
		return
	}
	defer c.Close()

	var s = State{
		Gophers: []Gopher{
			{"blue", 50},
			{"green", 50},
			{"purple", 50},
		},
	}

	ticker := time.NewTicker(HowOftenGophersLifeDrops)
	quickTicker := time.NewTicker(HowOftenStatusIsUpdated)
	defer ticker.Stop()
	defer quickTicker.Stop()

	go func() {
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				log.Println("unable to read message:", err)
				return
			}

			var fr FeedRequest
			if err := json.Unmarshal(message, &fr); err != nil {
				log.Println("unable to parse message:", err)
				return
			}

			if fr.Feed != "" {
				for i, gopher := range s.Gophers {
					if fr.Feed == gopher.Name {
						gopher.Life += 10
						s.Gophers[i] = gopher
					}
				}
			}
		}
	}()

	for {
		select {
		case <-ticker.C: // reduce a random gopher's life - wahahaha
			s.randomLifeReduction()
		case <-quickTicker.C: // broadcast state
			log.Println("broadcast state", s.Gophers)
			if err := c.WriteMessage(websocket.TextMessage, s.asJSON()); err != nil {
				log.Println("unable to write message:", err)
				return
			}
		}
	}
}

func main() {
	http.HandleFunc("/ws", socket)
	http.Handle("/", http.FileServer(http.Dir("./site")))
	log.Fatal(http.ListenAndServe(":3000", nil))
}
