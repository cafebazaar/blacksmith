package templating

import (
	"bytes"

	"github.com/coreos/coreos-cloudinit/config/validate"
)

func ValidateCloudConfig(config string) string {
	var buf bytes.Buffer
	buf.WriteString("\n")
	validationReport, _ := validate.Validate([]byte(config))
	for _, errorEntry := range validationReport.Entries() {
		buf.WriteString("# " + errorEntry.String() + "\n")
	}
	return buf.String()
}
