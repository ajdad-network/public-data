// Amalgamate all YAML family tree files into a single multi-tree YAML.
//
// Run:  go run ./cmd/amalgamate
//
// Produces: all-trees.yaml (uploadable to ajdad.net /admin/import)
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

type Family struct {
	Name         string `yaml:"name,omitempty"`
	Type         string `yaml:"type,omitempty"`
	SuffixMale   string `yaml:"suffixMale,omitempty"`
	SuffixFemale string `yaml:"suffixFemale,omitempty"`
	Hometown     string `yaml:"hometown,omitempty"`
	Description  string `yaml:"description,omitempty"`
	Pinned       bool   `yaml:"pinned,omitempty"`
	Visibility   string `yaml:"visibility,omitempty"`
	Category     string `yaml:"category,omitempty"`
}

type Person struct {
	ID        string `yaml:"id,omitempty"`
	Name      string `yaml:"name,omitempty"`
	Sex       string `yaml:"sex,omitempty"`
	Nickname  string `yaml:"nickname,omitempty"`
	Kunya     string `yaml:"kunya,omitempty"`
	Birthdate string `yaml:"birthdate,omitempty"`
	FatherID  string `yaml:"fatherId,omitempty"`
	MotherID  string `yaml:"motherId,omitempty"`
}

type DataFile struct {
	Family  Family   `yaml:"family"`
	Persons []Person `yaml:"persons"`
}

type Output struct {
	Trees []DataFile `yaml:"trees"`
}

func deriveCategory(rel string) string {
	if strings.HasPrefix(rel, "ruling-families/") {
		return "ruling"
	}
	if strings.HasPrefix(rel, "tribes/") {
		return "tribes"
	}
	if strings.HasPrefix(rel, "historical/") {
		return "historical"
	}
	return "other"
}

func findYAMLFiles(root string) ([]string, error) {
	var files []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		base := filepath.Base(path)
		if info.IsDir() && (base == "node_modules" || base == ".git" || base == ".github" || base == "cmd") {
			return filepath.SkipDir
		}
		if filepath.Ext(path) == ".yaml" || filepath.Ext(path) == ".yml" {
			files = append(files, path)
		}
		return nil
	})
	sort.Strings(files)
	return files, err
}

func main() {
	dataDir := "."
	if len(os.Args) > 1 {
		dataDir = os.Args[1]
	}

	absDir, _ := filepath.Abs(dataDir)
	fmt.Printf("Source: %s\n\n", absDir)

	files, err := findYAMLFiles(dataDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		os.Exit(1)
	}

	var trees []DataFile
	totalPersons := 0

	for _, filePath := range files {
		rel, _ := filepath.Rel(dataDir, filePath)

		raw, err := os.ReadFile(filePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  ERROR %s: %v\n", rel, err)
			os.Exit(1)
		}

		var data DataFile
		if err := yaml.Unmarshal(raw, &data); err != nil {
			fmt.Fprintf(os.Stderr, "  ERROR %s: %v\n", rel, err)
			os.Exit(1)
		}

		if data.Family.Name == "" || len(data.Persons) == 0 {
			fmt.Printf("  SKIP %s\n", rel)
			continue
		}

		// Strip family ID (may not match Firestore), derive category
		data.Family.Category = deriveCategory(rel)

		// Clean persons: only keep relevant fields
		cleaned := make([]Person, len(data.Persons))
		for i, p := range data.Persons {
			cleaned[i] = Person{
				ID:        p.ID,
				Name:      p.Name,
				Sex:       p.Sex,
				Nickname:  p.Nickname,
				Kunya:     p.Kunya,
				Birthdate: p.Birthdate,
				FatherID:  p.FatherID,
				MotherID:  p.MotherID,
			}
		}
		data.Persons = cleaned

		trees = append(trees, data)
		totalPersons += len(cleaned)
		fmt.Printf("  + %s — %s (%d persons)\n", rel, data.Family.Name, len(cleaned))
	}

	out, err := yaml.Marshal(Output{Trees: trees})
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR marshalling: %v\n", err)
		os.Exit(1)
	}

	outFile := "all-trees.yaml"
	if err := os.WriteFile(outFile, out, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR writing: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\nWrote %s\n", outFile)
	fmt.Printf("%d trees, %d persons, %dKB\n", len(trees), totalPersons, len(out)/1024)
}
