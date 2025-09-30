#include "customs.h"
#include "logging.h"
#include "LoRaWan_APP.h"
#include "display.h"

/*LoraWan region, select in arduino IDE tools*/
LoRaMacRegion_t loraWanRegion = ACTIVE_REGION;

/*LoraWan Class, Class A and Class C are supported*/
DeviceClass_t loraWanClass = CLASS_A;

/*the application data transmission duty cycle.  value in [ms].*/
uint32_t appTxDutyCycle = 15000;

/*OTAA or ABP*/
bool overTheAirActivation = true;

/*ADR enable*/
bool loraWanAdr = true;

/* Indicates if the node is sending confirmed or unconfirmed messages */
bool isTxConfirmed = false; // Changed to false for power efficiency (unconfirmed messages use less airtime)
int guess = 0;

/* Application port */
uint8_t appPort = 2;
/*!
 * Number of trials to transmit the frame, if the LoRaMAC layer did not
 * receive an acknowledgment. The MAC performs a datarate adaptation,
 * according to the LoRaWAN Specification V1.0.2, chapter 18.4, according
 * to the following table:
 *
 * Transmission nb | Data Rate
 * ----------------|-----------
 * 1 (first)       | DR
 * 2               | DR
 * 3               | max(DR-1,0)
 * 4               | max(DR-1,0)
 * 5               | max(DR-2,0)
 * 6               | max(DR-2,0)
 * 7               | max(DR-3,0)
 * 8               | max(DR-3,0)
 *
 * Note, that if NbTrials is set to 1 or 2, the MAC will not decrease
 * the datarate, in case the LoRaMAC layer did not receive an acknowledgment
 */
uint8_t confirmedNbTrials = 2; // Reduced from 4 to 2 for power efficiency

/* Prepares the payload of the frame */
static void prepareTxFrame( float ValueDensity, float ValueBattery)
{
    logMessage("prepareTxFrame");
    String payload = String(sensor_id) + ",0," + String(ValueDensity, 4) + ",1," +String(ValueBattery, 4);
    logMessage("payload ----");
    logMessage(payload);
    logMessage("end ----");

    appDataSize = payload.length() + 1;
    payload.getBytes(appData, appDataSize);
}

RTC_DATA_ATTR bool firstrun = true;

void InitLORA()
{
    Mcu.begin(HELTEC_BOARD, SLOW_CLK_TPYE);

    if (firstrun)
    {
#if ENABLE_DISPLAY
        LoRaWAN.displayMcuInit();
#endif
        firstrun = false;
    }
}

void LoopLORA(int current_count,double battery_percentage)
{
    switch (deviceState)
    {
    case DEVICE_STATE_INIT:
    {
#if (LORAWAN_DEVEUI_AUTO)
        LoRaWAN.generateDeveuiByChipID();
#endif
        LoRaWAN.init(loraWanClass, loraWanRegion);
        LoRaWAN.setDefaultDR(3);
        break;
    }
    case DEVICE_STATE_JOIN:
    {
#if ENABLE_DISPLAY
        LoRaWAN.displayJoining();
#endif
        LoRaWAN.join();
        break;
    }
    case DEVICE_STATE_SEND:
    {
        logMessage("startSending");
#if ENABLE_DISPLAY
        LoRaWAN.displaySending();
#endif
        if (current_count < 6 )
        {
                logMessage("Under 6 people, we dont send it for security reeasones");
                Serial.flush();
                Serial.end();
                VextOFF();
                esp_deep_sleep_start();
                break;
        }
        
        guess = static_cast<int>(current_count* factor);
        prepareTxFrame( float(guess),float(battery_percentage));
        LoRaWAN.send();
        logMessage("Send guess: " + String(guess,4));
        logMessage("Send battery: " + String(battery_percentage,4));
        deviceState = DEVICE_STATE_CYCLE;


        logMessage("Going to sleep now");
        delay(100);
        Serial.flush();
        Serial.end();
        VextOFF();
        esp_deep_sleep_start();
        logMessage("This will never be printed");

        break;
    }
    case DEVICE_STATE_CYCLE:
    {
        // Schedule next packet transmission
        txDutyCycleTime = appTxDutyCycle + randr(-APP_TX_DUTYCYCLE_RND, APP_TX_DUTYCYCLE_RND);
        LoRaWAN.cycle(txDutyCycleTime);
        deviceState = DEVICE_STATE_SLEEP;
        break;
    }
    case DEVICE_STATE_SLEEP:
    {
#if ENABLE_DISPLAY
        LoRaWAN.displayAck();
#endif
        LoRaWAN.sleep(loraWanClass);
        break;
    }
    default:
    {
        deviceState = DEVICE_STATE_INIT;
        break;
    }
    }
}