package main

import (
	"bufio"
	"fmt"
	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
	"os"
	"strings"
	"time"
)

type suggestion struct {
	Value       string
	Description string
}

func Init(version, path string) error {
	schema, err := schema(version)

	if err != nil {
		return err
	}

	values := make(map[string]string)
	values["dataContractSpecification"] = version

	promptRequiredFields(schema, values)

	valuesInSchema := inSchema(values, schema)

	return createDataContractSpecificationFile(valuesInSchema, path)
}

func schema(version string) (Schema, error) {
	var err error

	schemaFileName := fmt.Sprintf("schema-%v.json", version)
	file, err := os.ReadFile(schemaFileName)
	schema, err := GenerateSchema(file)

	if err != nil {
		return nil, err
	}

	schema.Sort()

	return *schema, err
}

func promptRequiredFields(schema Schema, values map[string]string) {
	for _, field := range schema.Flattened() {
		if field.Required && values[field.Identifier] == "" {
			values[field.Identifier], _ = prompt(fieldMessage(field), fieldSuggestion(field))
		}
	}
}

func fieldMessage(field SchemaField) string {
	if field.Description != nil {
		return fmt.Sprintf("Please type value for %v: %v\n", field.Identifier, *field.Description)
	} else {
		return fmt.Sprintf("Please enter %v\n", field.Identifier)
	}
}

func fieldSuggestion(field SchemaField) *suggestion {
	s := fieldSuggestionByIdentifier(field)
	if s != nil {
		return s
	}

	s = fieldSuggestionByDefault(field)
	if s != nil {
		return s
	}

	s = fieldSuggestionByFieldType(field)
	if s != nil {
		return s
	}

	return nil
}

func fieldSuggestionByIdentifier(field SchemaField) *suggestion {
	switch field.Identifier {
	case "info.id":
		return &suggestion{uuid.NewString(), "generated"}
	}
	return nil
}

func fieldSuggestionByFieldType(field SchemaField) *suggestion {
	switch field.Type {
	case SchemaFieldTypeDate:
		return &suggestion{time.Now().Format(time.DateOnly), "today"}
	}
	return nil
}

func fieldSuggestionByDefault(field SchemaField) *suggestion {
	if field.Default != "" {
		return &suggestion{field.Default, "default"}
	}
	return nil
}

func prompt(message string, suggestion *suggestion) (string, error) {
	printMessages(message, suggestion)
	input, err := readUserInput()

	if err != nil {
		return "", err
	} else if suggestion != nil && input == "" {
		return suggestion.Value, nil
	} else {
		return input, nil
	}
}

func printMessages(message string, suggestion *suggestion) {
	fmt.Print(message)
	if suggestion != nil {
		fmt.Printf("💡 press enter to use \"%v\" (%v)\n", suggestion.Value, suggestion.Description)
	}
}

func readUserInput() (string, error) {
	reader := bufio.NewReader(os.Stdin)

	input, error := reader.ReadString('\n')
	if error != nil {
		return "", error
	}
	return strings.TrimSuffix(input, "\n"), nil
}

func inSchema(values map[string]string, schema Schema) map[string]any {
	yamlMap := make(map[string]any)

	for _, schemaField := range schema {
		if value, ok := values[schemaField.Identifier]; ok {
			yamlMap[schemaField.FieldName] = value
			continue
		}

		switch schemaField.Type {
		case SchemaFieldTypeObject:
			yamlMap[schemaField.FieldName] = inSchema(values, *schemaField.ObjectSchema)
		case SchemaFieldTypeArray:
			yamlMap[schemaField.FieldName] = []any{}
		case SchemaFieldTypeString, SchemaFieldTypeDate, SchemaFieldTypeDuration:
			yamlMap[schemaField.FieldName] = ""
		default:
			yamlMap[schemaField.FieldName] = nil
		}
	}

	return yamlMap
}

func createDataContractSpecificationFile(values map[string]any, path string) error {
	if path == "" {
		path = "datacontract.yml"
	}

	file, err := createFile(path)
	defer file.Close()

	if err != nil {
		return err
	}

	yamlBytes, _ := yaml.Marshal(values)
	result := string(yamlBytes)

	fmt.Fprint(file, result)

	fmt.Println("---")
	fmt.Println(result)

	return nil
}

func createFile(path string) (*os.File, error) {
	file, error := os.Create(path)

	if error != nil {
		return nil, error
	}

	return file, nil
}