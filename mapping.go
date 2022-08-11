package main

import (
	"gopkg.in/yaml.v3"
	"os"
)

type Mapping struct {
	IdNames map[string]string `yaml:"id_to_names"`
}

func readMapping(filename string) (*Mapping, error) {
	m := &Mapping{}

	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	d := yaml.NewDecoder(file)

	if err := d.Decode(&m); err != nil {
		return nil, err
	}
	return m, nil
}
