package structs

type Clients struct {
	Longitude float64 `yaml:"longitude"`
	Latitude  float64 `yaml:"latitude"`
	UUID      string  `yaml:"uuid"`
	Name      string  `yaml:"name"`
}
