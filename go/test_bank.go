package main

import (
	"database/sql"
	// "fmt"
	"log"
	"strconv"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	_ "github.com/lib/pq"
)

var (
	temperatureValues []float64
	humidityValues    []float64
	mu                sync.Mutex
)

func main() {
	broker := "172.16.60.211:1883"
	temperatureTopic := "board1/sensor_data/temperature"
	humidityTopic := "board1/sensor_data/humidity"

	// Create a new MQTT client
	opts := mqtt.NewClientOptions().AddBroker(broker)
	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Fatal(token.Error())
	}

	// Subscribe to the MQTT topics
	if token := client.Subscribe(temperatureTopic, 0, temperatureHandler); token.Wait() && token.Error() != nil {
		log.Fatal(token.Error())
	}
	if token := client.Subscribe(humidityTopic, 0, humidityHandler); token.Wait() && token.Error() != nil {
		log.Fatal(token.Error())
	}

	// Connect to PostgreSQL
	db, err := sql.Open("postgres", "host=172.16.60.211 user=adminbee9644 password=adminbee9644 dbname=smart_city_go sslmode=disable")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	for {
		// Wait for 10 data points
		for {
			mu.Lock()
			if len(temperatureValues) >= 30 && len(humidityValues) >= 30 {
				mu.Unlock()
				break
			}
			mu.Unlock()
			time.Sleep(1 * time.Second)
		}

		// Calculate and print the average values
		averageTemperature := calculateAverage(temperatureValues)
		averageHumidity := calculateAverage(humidityValues)
		// fmt.Printf("Average Temperature: %.2f\n", averageTemperature)
		// fmt.Printf("Average Humidity: %.2f\n", averageHumidity)

		// Insert data into PostgreSQL
		err := insertData(db, averageTemperature, averageHumidity)
		if err != nil {
			log.Println("Error inserting data into PostgreSQL:", err)
		}

		// Clear the data arrays
		mu.Lock()
		temperatureValues = nil
		humidityValues = nil
		mu.Unlock()
	}
}

func temperatureHandler(client mqtt.Client, msg mqtt.Message) {
	handleSensorData(msg, &temperatureValues)
}

func humidityHandler(client mqtt.Client, msg mqtt.Message) {
	handleSensorData(msg, &humidityValues)
}

func handleSensorData(msg mqtt.Message, values *[]float64) {
	// Convert []byte to string
	payload := string(msg.Payload())

	// Convert string to float64
	data, err := strconv.ParseFloat(payload, 64)
	if err != nil {
		log.Println("Error converting data to float64:", err)
		return
	}

	// Update the array with the latest data value
	mu.Lock()
	*values = append(*values, data)

	// Keep only the last 10 values in the array
	if len(*values) > 30 {
		*values = (*values)[len(*values)-30:]
	}
	mu.Unlock()
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

func insertData(db *sql.DB, temperature, humidity float64) error {
	mu.Lock()
	defer mu.Unlock()

	// Prepare the SQL statement
	stmt, err := db.Prepare("INSERT INTO sensor_data (id, temp, humid, time_in) VALUES (1, ROUND($1::numeric, 2), ROUND($2::numeric, 2), CURRENT_TIMESTAMP)")

	if err != nil {
		return err
	}
	defer stmt.Close()

	// Execute the SQL statement
	_, err = stmt.Exec(temperature, humidity)
	if err != nil {
		return err
	}

	return nil
}
