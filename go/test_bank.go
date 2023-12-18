package main

import (
	"database/sql"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	_ "github.com/lib/pq"
)

var (
	mu                      sync.Mutex
	temperatureValuesBoard1 []float64
	humidityValuesBoard1    []float64
	temperatureValuesBoard2 []float64
	humidityValuesBoard2    []float64
)

// Config holds the configuration values
type Config struct {
	Broker                   string
	TemperatureTopic1        string
	HumidityTopic1           string
	TemperatureTopic2        string
	HumidityTopic2           string
	DatabaseConnectionString string
}

func main() {
	config := Config{
		Broker:                   "172.16.60.211:1883",
		TemperatureTopic1:        "board1/sensor_data/temperature",
		HumidityTopic1:           "board1/sensor_data/humidity",
		TemperatureTopic2:        "board2/sensor_data/temperature",
		HumidityTopic2:           "board2/sensor_data/humidity",
		DatabaseConnectionString: "host=172.16.60.211 user=adminbee9644 password=adminbee9644 dbname=smart_city_go sslmode=disable",
	}

	opts := mqtt.NewClientOptions().AddBroker(config.Broker)
	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Fatal(token.Error())
	}

	if token := client.Subscribe(config.TemperatureTopic1, 0, temperatureHandlerBoard1); token.Wait() && token.Error() != nil {
		log.Fatal(token.Error())
	}
	if token := client.Subscribe(config.HumidityTopic1, 0, humidityHandlerBoard1); token.Wait() && token.Error() != nil {
		log.Fatal(token.Error())
	}

	if token := client.Subscribe(config.TemperatureTopic2, 0, temperatureHandlerBoard2); token.Wait() && token.Error() != nil {
		log.Fatal(token.Error())
	}
	if token := client.Subscribe(config.HumidityTopic2, 0, humidityHandlerBoard2); token.Wait() && token.Error() != nil {
		log.Fatal(token.Error())
	}

	db, err := sql.Open("postgres", config.DatabaseConnectionString)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Use channels to signal completion of 50-second intervals
	intervalCh := time.Tick(50 * time.Second)

	// Main loop
	for {
		// Wait for data points or 50-second interval completion
		select {
		case <-waitForDataPoints(50*time.Second, temperatureValuesBoard1, humidityValuesBoard1, temperatureValuesBoard2, humidityValuesBoard2):
			averageTemperatureBoard1 := calculateAverage(temperatureValuesBoard1)
			averageHumidityBoard1 := calculateAverage(humidityValuesBoard1)

			averageTemperatureBoard2 := calculateAverage(temperatureValuesBoard2)
			averageHumidityBoard2 := calculateAverage(humidityValuesBoard2)

			log.Printf("Average Temperature (Board 1): %.2f\n", averageTemperatureBoard1)
			log.Printf("Average Humidity (Board 1): %.2f\n", averageHumidityBoard1)

			log.Printf("Average Temperature (Board 2): %.2f\n", averageTemperatureBoard2)
			log.Printf("Average Humidity (Board 2): %.2f\n", averageHumidityBoard2)

			err := insertData(db, averageTemperatureBoard1, averageHumidityBoard1, 1)
			if err != nil {
				log.Println("Error inserting data into PostgreSQL:", err)
			}

			err = insertData(db, averageTemperatureBoard2, averageHumidityBoard2, 2)
			if err != nil {
				log.Println("Error inserting data into PostgreSQL:", err)
			}

			clearDataArrays()

		case <-intervalCh:
			log.Println("50-second interval completed.")
		}
	}
}

func temperatureHandlerBoard1(client mqtt.Client, msg mqtt.Message) {
	handleSensorData(msg, &temperatureValuesBoard1)
}

func humidityHandlerBoard1(client mqtt.Client, msg mqtt.Message) {
	handleSensorData(msg, &humidityValuesBoard1)
}

func temperatureHandlerBoard2(client mqtt.Client, msg mqtt.Message) {
	handleSensorData(msg, &temperatureValuesBoard2)
}

func humidityHandlerBoard2(client mqtt.Client, msg mqtt.Message) {
	handleSensorData(msg, &humidityValuesBoard2)
}

func handleSensorData(msg mqtt.Message, values *[]float64) {
	// Convert []byte to string
	payload := string(msg.Payload())

	// Trim leading and trailing whitespaces
	payload = strings.TrimSpace(payload)

	// Convert string to float64
	data, err := strconv.ParseFloat(payload, 64)
	if err != nil {
		log.Println("Error converting data to float64:", err)
		return
	}

	// Update the array with the latest data value
	mu.Lock()
	*values = append(*values, data)
	mu.Unlock()
}

func waitForDataPoints(timeout time.Duration, boards ...[]float64) <-chan struct{} {
	dataReady := make(chan struct{})
	go func() {
		defer close(dataReady)
		for {
			mu.Lock()
			ready := true
			for _, board := range boards {
				if len(board) == 0 {
					ready = false
					break
				}
			}
			mu.Unlock()

			if ready {
				return
			}
			time.Sleep(1 * time.Second)
		}
	}()
	return dataReady
}

func calculateAverage(values []float64) float64 {
	sum := 0.0
	for _, value := range values {
		sum += value
	}
	if len(values) > 0 {
		return sum / float64(len(values))
	}
	return 0.0
}

func insertData(db *sql.DB, temperature, humidity float64, boardID int) error {
	stmt, err := db.Prepare("INSERT INTO sensor_data (id, temp, humid, time_in) VALUES ($3, ROUND($1::numeric, 2), ROUND($2::numeric, 2), CURRENT_TIMESTAMP)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(temperature, humidity, boardID)
	if err != nil {
		return err
	}

	return nil
}

func clearDataArrays() {
	mu.Lock()
	defer mu.Unlock()

	temperatureValuesBoard1 = nil
	humidityValuesBoard1 = nil
	temperatureValuesBoard2 = nil
	humidityValuesBoard2 = nil
}
