package structs

type Clients struct {
	Longitude float64 `yaml:"longitude" json:"longitude"`
	Latitude  float64 `yaml:"latitude"  json:"latitude"`
	UUID      string  `yaml:"uuid"      json:"uuid"`
	Name      string  `yaml:"name"      json:"name"`
}
