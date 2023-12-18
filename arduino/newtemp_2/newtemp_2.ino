#include <DHT.h>
#include <WiFi.h>
#include <PubSubClient.h>

const char* ssid = "extend_60";
const char* password = "1231231235";
const char* mqtt_server = "172.16.60.211";
const char* mqtt_topic_temperature = "board1/temperature";
const char* mqtt_topic_humidity = "board1/humidity";

WiFiClient espClient;
PubSubClient client(espClient);

#define DHTPIN 26          // กำหนดขาที่เชื่อมต่อ DHT26
#define DHTTYPE DHT11     // กำหนดประเภทของ DHT (DHT11 หรือ DHT22)

DHT dht(DHTPIN, DHTTYPE);

void setup() {
  Serial.begin(115200);

  // เชื่อมต่อ WiFi
  WiFi.begin(ssid, password);
  
  while (WiFi.status() != WL_CONNECTED) {
    Serial.println("Connecting to WiFi...");
  }
  Serial.println("Connected to WiFi");

  // เชื่อมต่อ MQTT Broker
  client.setServer(mqtt_server, 1883);

  // รอให้เซ็นเซอร์เตรียมตัว
  delay(2000);
}

void loop() {
  // ตรวจสอบการเชื่อมต่อ MQTT
  if (!client.connected()) {
    reconnect();
  }

  // อ่านค่าอุณหภูมิและความชื้นจากเซ็นเซอร์
  float temperature = dht.readTemperature();
  float humidity = dht.readHumidity();

  // ตรวจสอบว่าการอ่านค่าจากเซ็นเซอร์เป็นค่าที่ถูกต้องหรือไม่
  if (isnan(temperature) || isnan(humidity)) {
    Serial.println("Failed to read from DHT sensor!");
    delay(1000);
    return;
  }

  // ส่งข้อมูลไปยัง MQTT Broker
  char tempMsg[10];
  snprintf(tempMsg, 10, "%.2f", temperature);
  client.publish(mqtt_topic_temperature, tempMsg);

  char humidityMsg[10];
  snprintf(humidityMsg, 10, "%.2f", humidity);
  client.publish(mqtt_topic_humidity, humidityMsg);

  delay(1500);  // รอ 1 วินาที
}

void reconnect() {
  // รอ 1 วินาที
  delay(1000);
  Serial.println("Reconnecting to MQTT Broker...");
  
  // ลองเชื่อมต่อใหม่
  if (client.connect("ESP32Client")) {
    Serial.println("Connected to MQTT Broker");
  } else {
    Serial.println("Failed to connect to MQTT Broker. Retrying...");
  }
}