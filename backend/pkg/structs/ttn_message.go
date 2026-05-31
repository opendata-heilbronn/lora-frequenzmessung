package structs

import "time"

type TtnMessage struct {
	EndDeviceIds struct {
		DeviceId       string `json:"device_id"`
		ApplicationIds struct {
			ApplicationId string `json:"application_id"`
		} `json:"application_ids"`
		DevEui  string `json:"dev_eui"`
		JoinEui string `json:"join_eui"`
		DevAddr string `json:"dev_addr"`
	} `json:"end_device_ids"`
	CorrelationIds []string  `json:"correlation_ids"`
	ReceivedAt     time.Time `json:"received_at"`
	UplinkMessage  struct {
		SessionKeyId string `json:"session_key_id"`
		FPort        int    `json:"f_port"`
		FrmPayload   string `json:"frm_payload"`
		RxMetadata   []struct {
			GatewayIds struct {
				GatewayId string `json:"gateway_id"`
				Eui       string `json:"eui"`
			} `json:"gateway_ids"`
			Time        time.Time `json:"time"`
			Timestamp   int       `json:"timestamp"`
			Rssi        int       `json:"rssi"`
			ChannelRssi int       `json:"channel_rssi"`
			Snr         float64   `json:"snr"`
			Location    struct {
				Latitude  float64 `json:"latitude"`
				Longitude float64 `json:"longitude"`
				Altitude  int     `json:"altitude"`
				Source    string  `json:"source"`
			} `json:"location"`
			UplinkToken  string    `json:"uplink_token"`
			ReceivedAt   time.Time `json:"received_at"`
			ChannelIndex int       `json:"channel_index,omitempty"`
		} `json:"rx_metadata"`
		Settings struct {
			DataRate struct {
				Lora struct {
					Bandwidth       int    `json:"bandwidth"`
					SpreadingFactor int    `json:"spreading_factor"`
					CodingRate      string `json:"coding_rate"`
				} `json:"lora"`
			} `json:"data_rate"`
			Frequency string    `json:"frequency"`
			Timestamp int       `json:"timestamp"`
			Time      time.Time `json:"time"`
		} `json:"settings"`
		ReceivedAt      time.Time `json:"received_at"`
		Confirmed       bool      `json:"confirmed"`
		ConsumedAirtime string    `json:"consumed_airtime"`
		VersionIds      struct {
			BrandId         string `json:"brand_id"`
			ModelId         string `json:"model_id"`
			HardwareVersion string `json:"hardware_version"`
			FirmwareVersion string `json:"firmware_version"`
			BandId          string `json:"band_id"`
		} `json:"version_ids"`
		NetworkIds struct {
			NetId          string `json:"net_id"`
			NsId           string `json:"ns_id"`
			TenantId       string `json:"tenant_id"`
			ClusterId      string `json:"cluster_id"`
			ClusterAddress string `json:"cluster_address"`
		} `json:"network_ids"`
	} `json:"uplink_message"`
}
