package bank

import "digital.vasic.challenges/pkg/challenge"

// BankFile represents the JSON structure of a challenge bank file.
type BankFile struct {
	Version    string                 `json:"version"`
	Name       string                 `json:"name"`
	Challenges []challenge.Definition `json:"challenges"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}
