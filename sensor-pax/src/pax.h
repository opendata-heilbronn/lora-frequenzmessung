#define LIBPAX_ARDUINO 1
#define LIBPAX_WIFI 0
#define LIBPAX_BLE 1

#include "libpax_api.h"

struct count_payload_t count_from_libpax;
bool new_data_available = false;
int current_count = 0;


void log()
{
  new_data_available = true;
  current_count = count_from_libpax.pax;
}

// libpax initialization
void InitPAX()
{
  struct libpax_config_t configuration;
  libpax_default_config(&configuration);
  configuration.blecounter = 1;
  configuration.blescantime = 0;
  configuration.wificounter = 0;
  configuration.wifi_channel_switch_interval = 50;
  configuration.wifi_rssi_threshold = -80;
  configuration.ble_rssi_threshold = -80;
  libpax_update_config(&configuration);

  // internal processing initialization
  libpax_counter_init(log, &count_from_libpax, 10, 1);
  libpax_counter_start();
}

void LoopPAX() {
    
}