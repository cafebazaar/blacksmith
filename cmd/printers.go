package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"

	yaml "gopkg.in/yaml.v2"
)

func formatOutput(payload interface{}) string {
	format := RootCmd.PersistentFlags().Lookup("output").Value.String()
	var buf bytes.Buffer

	switch format {
	case "json":
		json.NewEncoder(&buf).Encode(payload)
	case "yaml":
		out, err := yaml.Marshal(payload)
		if err != nil {
			buf.WriteString(fmt.Sprintf("Marshal failed: %s\n", err))
		}
		buf.Write(out)
	default:
		buf.WriteString("payload type not supported\n")
	}

	return buf.String()
}
