package mcap_schema_resolver

import (
	"fmt"
	"testing"

	"github.com/okieraised/monitoring-agent/internal/utilities"
)

func TestStripArraySuffix(t *testing.T) {
	fmt.Println(utilities.StripArraySuffix("int32[]"))
	fmt.Println(utilities.StripArraySuffix("FloatingPointRange[<=1]"))
	fmt.Println(utilities.StripArraySuffix("string[]"))
}
