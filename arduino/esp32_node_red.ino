#include <WiFi.h>
#include <PubSubClient.h>
#include <DHT.h>

#define DHTPIN 26      // กำหนดขาที่เชื่อมต่อกับ DHT11
#define DHTTYPE DHT11 // กำหนดประเภทของเซ็นเซอร์ (DHT11, DHT22, DHT21)

const char* ssid = "extend_60";        // ชื่อของ WiFi
const char* password = "1231231235";    // รหัสของ WiFi
const char* mqtt_server = "172.16.60.206"; // IP address ของ Raspberry Pi

WiFiClient espClient;
PubSubClient client(espClient);
DHT dht(DHTPIN, DHTTYPE);

void setup_wifi() {
  delay(10);
  Serial.println();
  Serial.print("Connecting to ");
  Serial.println(ssid);

  WiFi.begin(ssid, password);

  while (WiFi.status() != WL_CONNECTED) {
    delay(500);
    Serial.print(".");
  }

  Serial.println("");
  Serial.println("WiFi connected");
  Serial.println("IP address: ");
  Serial.println(WiFi.localIP());
}

void reconnect() {
  while (!client.connected()) {
    Serial.print("Attempting MQTT connection...");
    if (client.connect("ESP32Client")) {
      Serial.println("connected");
    } else {
      Serial.print("failed, rc=");
      Serial.print(client.state());
      Serial.println(" try again in 5 seconds");
      delay(5000);
    }
  }
}

void setup() {
  Serial.begin(115200);
  dht.begin();
  setup_wifi();
  client.setServer(mqtt_server, 1883);
}

void loop() {
  if (!client.connected()) {
    reconnect();
  }
  client.loop();

  delay(2000);

  float humidity = dht.readHumidity();
  float temperature = dht.readTemperature();

  if (isnan(humidity) || isnan(temperature)) {
    Serial.println("Failed to read from DHT sensor!");
    return;
  }

  char tempString[8];
  char humidString[8];
  dtostrf(temperature, 6, 2, tempString);
  dtostrf(humidity, 6, 2, humidString);

  client.publish("temp", tempString); // ส่งค่าอุณหภูมิไปยัง topic "temp"
  client.publish("humid", humidString); // ส่งค่าความชื้นไปยัง topic "humid"
}
