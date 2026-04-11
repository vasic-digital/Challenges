package bank

import "digital.vasic.challenges/pkg/challenge"

// BankFile represents the JSON structure of a challenge bank file.
//
// Two root keys are accepted for the list of definitions:
//   - "challenges" — canonical Challenges module key
//   - "test_cases" — key used by HelixQA test banks so the same
//     bank files can flow through both loaders unchanged
//
// parseBankFile merges TestCases into Challenges when Challenges is
// empty so downstream code only ever reads the Challenges slice.
type BankFile struct {
	Version     string                 `json:"version"`
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Challenges  []challenge.Definition `json:"challenges"`
	TestCases   []challenge.Definition `json:"test_cases,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}
