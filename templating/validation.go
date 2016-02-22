package templating

import (
	"bytes"

	"github.com/coreos/coreos-cloudinit/config/validate"
)

func ValidateCloudConfig(config string) string {
	var errors bytes.Buffer
	errors.WriteString("\n")
	validationReport, _ := validate.Validate([]byte(config))
	for _, errorEntry := range validationReport.Entries() {
		errors.WriteString("#")
		errors.WriteString(errorEntry.String())
		errors.WriteString("\n")
	}
	return errors.String()
}
