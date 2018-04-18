package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"sort"
	"strings"
)

var indent = "> "

type schemaObject struct {
	Definitions          schemaObjectMap `json:"definitions"`
	AdditionalProperties bool            `json:"additionalProperties"`
	Description          string          `json:"description"`
	Default              *interface{}    `json:"default"`
	Properties           schemaObjectMap `json:"properties"`
	Items                *schemaObject   `json:"items"`
	Required             []string        `json:"required"`
	Type                 string          `json:"type"`
	Title                string          `json:"title"`
	Examples             []schemaObject  `json:"examples"`
	OneOf                []schemaObject  `json:"oneOf"`
	Enum                 []interface{}   `json:"enum"`
	Ref                  string          `json:"$ref"`
}

type schemaObjectContext struct {
	Depth    int
	Path     string
	Name     string
	Required bool
}

type schemaObjectMap map[string]schemaObject

func (sop schemaObjectMap) SortedKeys() []string {
	var keys []string

	for key := range sop {
		keys = append(keys, key)
	}

	sort.Strings(keys)

	return keys
}

func main() {
	stdin, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		log.Panicf("reading stdin: %v", err)
	}

	var schema schemaObject

	err = json.Unmarshal(stdin, &schema)
	if err != nil {
		log.Panicf("unmarshaling stdin: %v", err)
	}

	var schemaContext schemaObjectContext

	if len(os.Args) > 1 {
		err = json.Unmarshal([]byte(os.Args[1]), &schemaContext)
		if err != nil {
			log.Panicf("unmarshaling stdin: %v", err)
		}
	}

	if schema.Title != "" {
		fmt.Printf("## %s {: #%s }\n", schema.Title, schemaContext.Path)
		fmt.Printf("\n")
	}

	if schema.Description != "" {
		fmt.Printf("%s\n", schema.Description)
		fmt.Printf("\n")
	}

	docsifyObjectProperties(schema, schemaContext)

	for definitionName, definition := range schema.Definitions {
		innerSchemaContext := schemaContext
		innerSchemaContext.Path = strings.TrimPrefix(fmt.Sprintf("%s.def-%s", innerSchemaContext.Path, definitionName), ".")
		innerSchemaContext.Depth += 1

		fmt.Printf("## %s {: #%s }\n", definition.Title, innerSchemaContext.Path)
		fmt.Printf("\n")

		if schema.Description != "" {
			fmt.Printf("%s\n", definition.Description)
			fmt.Printf("\n")
		}

		docsifyObjectProperties(definition, innerSchemaContext)
	}
}

func docsifyObject(schema schemaObject, schemaContext schemaObjectContext) {
	prefix := strings.Repeat(indent, schemaContext.Depth)

	if schema.Description != "" {
		fmt.Printf("%s%s\n", prefix, schema.Description)
		fmt.Printf("%s\n", prefix)
	}

	{
		fmt.Printf("%s * *Use*: ", prefix)

		if schemaContext.Required {
			fmt.Printf("Required")
		} else {
			fmt.Printf("Optional")
		}

		fmt.Printf("\n")
	}

	if schema.Type != "" {
		fmt.Printf("%s * *Type*: %s\n", prefix, schema.Type)
	}

	if schema.Default != nil {
		marshalBytes, _ := json.Marshal(schema.Default)
		fmt.Printf("%s * *Default*: `%s`\n", prefix, marshalBytes)
	}

	if schema.Ref != "" {
		fmt.Printf("%s * *Details*: [See Schema](#def-%s)\n", prefix, path.Base(schema.Ref))
	}

	if len(schema.Enum) > 0 {
		listPrefix := ""

		fmt.Printf("%s * *Supported Values*: ", prefix)

		for _, enum := range schema.Enum {
			marshalBytes, _ := json.Marshal(enum)
			fmt.Printf("%s`%s`", listPrefix, marshalBytes)

			listPrefix = ", "
		}

		fmt.Printf("\n")
	}

	if len(schema.Examples) > 0 {
		for _, example := range schema.Examples {
			marshalBytes, _ := json.MarshalIndent(example.Default, "", "  ")
			fmt.Printf("%s * *Example*: `%s`\n", prefix, marshalBytes)
		}
	}

	if schema.Type == "array" && schema.Items != nil {
		if len(schema.Items.OneOf) > 0 {
			for oneOfIdx, oneOf := range schema.Items.OneOf {
				innerSchemaContext := schemaContext
				innerSchemaContext.Path = fmt.Sprintf("%s[%d]", innerSchemaContext.Path, oneOfIdx)
				innerSchemaContext.Depth += 1

				fmt.Printf("%s\n", prefix)
				fmt.Printf("%s###%s %s {: #%s }\n", prefix, strings.Repeat("#", innerSchemaContext.Depth), oneOf.Title, innerSchemaContext.Path)
				fmt.Printf("%s\n", prefix)
				docsifyObject(oneOf, innerSchemaContext)
				fmt.Printf("%s\n", prefix)

				docsifyObjectProperties(*schema.Items, innerSchemaContext)
			}
		} else if len(schema.Items.Properties) > 0 {
			innerSchemaContext := schemaContext
			innerSchemaContext.Depth += 1

			fmt.Printf("%s\n", prefix)

			docsifyObjectProperties(*schema.Items, innerSchemaContext)
		}
	} else if schema.Type == "object" {
		innerSchemaContext := schemaContext
		innerSchemaContext.Depth += 1

		fmt.Printf("%s\n", prefix)

		docsifyObjectProperties(schema, innerSchemaContext)
	}
}

func docsifyObjectProperties(schema schemaObject, schemaContext schemaObjectContext) {
	prefix := strings.Repeat(indent, schemaContext.Depth)

	for _, propertyName := range schema.Properties.SortedKeys() {
		property := schema.Properties[propertyName]

		innerSchemaContext := schemaObjectContext{
			Depth: schemaContext.Depth,
			Path:  strings.TrimPrefix(fmt.Sprintf("%s.%s", schemaContext.Path, propertyName), "."),
			Name:  propertyName,
		}

		for _, requiredPropertyName := range schema.Required {
			if requiredPropertyName == propertyName {
				innerSchemaContext.Required = true

				break
			}
		}

		if property.Type == "array" && property.Items != nil && property.Items.Type == "object" {
			propertyName = fmt.Sprintf("%s[]", propertyName)
		}

		fmt.Printf("%s###%s `%s` {: #%s }\n", prefix, strings.Repeat("#", innerSchemaContext.Depth), propertyName, innerSchemaContext.Path)
		fmt.Printf("%s\n", prefix)
		docsifyObject(property, innerSchemaContext)
		fmt.Printf("%s\n", prefix)
	}
}
