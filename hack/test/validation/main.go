package main

import (
	"fmt"
	"github.com/pb33f/libopenapi"
	"k8s.io/apimachinery/pkg/util/yaml"
	"os"
)

type comWFYAML struct {
	ApiVersion string `yaml:"apiVersion"`
	Spec       struct {
		Versions []struct {
			Schema struct {
				OpenAPIV3Schema interface{} `yaml:"openAPIV3Schema"`
			} `yaml:"schema"`
		} `yaml:"versions"`
	} `yaml:"spec"`
}

func main() {
	manifestFilePath := os.Args[1]
	//crPath := os.Args[2]

	// TODO: use yq to pull out the actual schema and pass it in as a temp file so I don't have to figure out how bytes conversion works
	manifestYAMLFile, err := os.ReadFile(manifestFilePath)
	if err != nil {
		panic(err)
	}

	var comWF comWFYAML
	err = yaml.Unmarshal(manifestYAMLFile, &comWF)
	if err != nil {
		panic(err)
	}

	schemaDocument, err := libopenapi.NewDocument(comWF.Spec.Versions[0].Schema.OpenAPIV3Schema.([]byte))
	if err != nil {
		panic(err)
	}

	v3Model, errors := schemaDocument.BuildV3Model()

	// if anything went wrong when building the v3 model, a slice of errors will be returned
	if len(errors) > 0 {
		for i := range errors {
			fmt.Printf("error: %e\n", errors[i])
		}
		panic(fmt.Sprintf("cannot create v3 model from document: %d errors reported",
			len(errors)))
	}

	fmt.Printf("Here's the document?\n%+v\n\n", schemaDocument)
	fmt.Printf("Here's the document model?\n%+v", v3Model)
}
