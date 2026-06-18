package dkan

// DKANEntityResponse represents the standard response object returned by DKAN
// when a new entity (like a Dataset or Resource) is successfully created.
type DKANEntityResponse struct {
	Identifier string `json:"identifier"`
}
