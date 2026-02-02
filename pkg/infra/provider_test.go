package infra

import (
	"testing"
)

func TestInfraProvider_Interface(t *testing.T) {
	var _ InfraProvider = &ContainersAdapter{}
}
