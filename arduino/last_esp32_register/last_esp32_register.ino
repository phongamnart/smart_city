#include <WiFi.h>         
#include <WiFiManager.h>  
#include "FS.h"
#include "SPIFFS.h"  
#include <HTTPClient.h>
#include <ArduinoJson.h>

WiFiManager wm;

String TOKEN;

int button = 25;

uint32_t chipId = 0;

void setup() {
  Serial.begin(115200);
  delay(10);

  pinMode(button, INPUT_PULLUP);

  wm.setTimeout(120);

  if (!wm.autoConnect("Akara")) {
    Serial.println("Failed to connect and hit timeout");
    delay(3000);
    ESP.restart();
    delay(5000);
  }

  Serial.println("Connected to WiFi");

  if (!SPIFFS.begin(true)) {
    Serial.println("An Error has occurred while mounting SPIFFS");
    return;
  }

  const char *filePath = "/file.config";
  if (!tokenExists(filePath)) {
    String token = generateRandomToken(20);
    writeToFile(filePath, token);
  }

  TOKEN = readFromFile(filePath);

  

  for (int i = 0; i < 17; i = i + 8) {
    chipId |= ((ESP.getEfuseMac() >> (40 - i)) & 0xff) << i;
  }

  //Serial.printf("ESP32 Chip model = %s Rev %d\n", ESP.getChipModel(), ESP.getChipRevision());
  //Serial.printf("This chip has %d cores\n", ESP.getChipCores());

  Serial.print("Chip ID: ");
  Serial.println(chipId);

  Serial.print("Token: ");
  Serial.println(TOKEN);

  sendDataToServer(chipId, TOKEN);

}

void loop() {
  if (digitalRead(button) == LOW) {
    wm.resetSettings();
    // resetToken();
    delay(1000);
    ESP.restart();
    delay(5000);
  }
}

void writeToFile(const String &path, const String &data) {
  File file = SPIFFS.open(path, FILE_WRITE);
  if (!file) {
    Serial.println("Failed to open file for writing");
    return;
  }
  if (file.print(data)) {
    Serial.println("File written successfully");
  } else {
    Serial.println("Write failed");
  }
  file.close();
}

String readFromFile(const String &path) {
  File file = SPIFFS.open(path, FILE_READ);
  if (!file || file.isDirectory()) {
    Serial.println("Failed to open file for reading");
    return String();
  }
  String fileContent;
  while (file.available()) {
    fileContent += char(file.read());
  }
  file.close();
  return fileContent;
}

String generateRandomToken(int length) {
  const char *charSet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789";
  String token;
  for (int i = 0; i < length; ++i) {
    int randomIndex = random(strlen(charSet));
    token += charSet[randomIndex];
  }
  return token;
}

bool tokenExists(const String &path) {
  File file = SPIFFS.open(path, FILE_READ);
  if (!file || file.isDirectory()) {
    Serial.println("Failed to open file for reading");
    return false;
  }
  String fileContent = readFromFile(path);
  file.close();
  return !fileContent.isEmpty();
}

void resetToken() {
  const char *filePath = "/file.config";

  if (SPIFFS.exists(filePath)) {
    SPIFFS.remove(filePath);
    Serial.println("Old Token file removed");
  }

}

void sendDataToServer(uint32_t chipId, String token) {
  HTTPClient http;
  
  http.begin("http://172.16.60.206:8080/esp32data");

  StaticJsonDocument<200> doc;
  doc["chipId"] = chipId;
  doc["token"] = token;

  String jsonString;
  serializeJson(doc, jsonString);

  int httpResponseCode = http.POST(jsonString);

  if (httpResponseCode > 0) {
    Serial.print("HTTP Response code: ");
    Serial.println(httpResponseCode);
    String response = http.getString();
    Serial.println(response);
  } else {
    Serial.print("Error code: ");
    Serial.println(httpResponseCode);
  }

  http.end();
}