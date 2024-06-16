package cmd

import (
	"encoding/json"
	"fmt"
	"log"

	"gopkg.in/yaml.v2"
)

func output(data interface{}) {
	switch outputFormat {
	case "yaml":
		out, err := yaml.Marshal(data)
		if err != nil {
			log.Fatalf("Error marshalling to YAML: %v", err)
		}
		fmt.Println(string(out))
	case "json":
		out, err := json.MarshalIndent(data, "", "  ")
		if err != nil {
			log.Fatalf("Error marshalling to JSON: %v", err)
		}
		fmt.Println(string(out))
	default:
		log.Fatalf("Invalid output format: %s", outputFormat)
	}
}
