package data_test

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"
	"unicode"

	"gopkg.in/yaml.v3"
)

type Family struct {
	Name         string `yaml:"name"`
	SuffixMale   string `yaml:"suffixMale"`
	SuffixFemale string `yaml:"suffixFemale"`
	Description  string `yaml:"description"`
}

type Person struct {
	ID        string `yaml:"id"`
	Name      string `yaml:"name"`
	Sex       string `yaml:"sex"`
	Nickname  string `yaml:"nickname"`
	Kunya     string `yaml:"kunya"`
	Birthdate string `yaml:"birthdate"`
	FatherID  string `yaml:"fatherId"`
	MotherID  string `yaml:"motherId"`
}

type DataFile struct {
	Family  Family   `yaml:"family"`
	Persons []Person `yaml:"persons"`
}

var (
	westernDigit        = regexp.MustCompile(`[0-9]`)
	easternArabicNumber = regexp.MustCompile(`^[٠-٩]+$`)
)

func containsArabic(s string) bool {
	for _, r := range s {
		if unicode.Is(unicode.Arabic, r) {
			return true
		}
	}
	return false
}

func findYAMLFiles(t *testing.T) []string {
	t.Helper()
	var files []string
	err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		base := filepath.Base(path)
		if info.IsDir() && (base == "node_modules" || base == ".git" || base == ".github") {
			return filepath.SkipDir
		}
		if filepath.Ext(path) == ".yaml" {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("finding YAML files: %v", err)
	}
	return files
}

func loadDataFile(t *testing.T, path string) DataFile {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	var data DataFile
	if err := yaml.Unmarshal(raw, &data); err != nil {
		t.Fatalf("parsing %s: %v", path, err)
	}
	return data
}

func TestArabicText(t *testing.T) {
	for _, file := range findYAMLFiles(t) {
		t.Run(file, func(t *testing.T) {
			data := loadDataFile(t, file)

			t.Run("family_fields", func(t *testing.T) {
				fields := map[string]string{
					"name":         data.Family.Name,
					"suffixMale":   data.Family.SuffixMale,
					"suffixFemale": data.Family.SuffixFemale,
					"description":  data.Family.Description,
				}
				for field, value := range fields {
					if value == "" {
						continue
					}
					if !containsArabic(value) {
						t.Errorf("family.%s: %q does not contain Arabic characters", field, value)
					}
					if westernDigit.MatchString(value) {
						t.Errorf("family.%s: %q contains Western digits (use Eastern Arabic numerals ٠-٩)", field, value)
					}
				}
			})

			t.Run("person_fields", func(t *testing.T) {
				for _, p := range data.Persons {
					fields := map[string]string{
						"name":     p.Name,
						"nickname": p.Nickname,
						"kunya":    p.Kunya,
					}
					for field, value := range fields {
						if value == "" {
							continue
						}
						if !containsArabic(value) {
							t.Errorf("%s.%s: %q does not contain Arabic characters", p.Name, field, value)
						}
						if westernDigit.MatchString(value) {
							t.Errorf("%s.%s: %q contains Western digits (use Eastern Arabic numerals ٠-٩)", p.Name, field, value)
						}
					}
				}
			})
		})
	}
}

func TestEasternArabicNumerals(t *testing.T) {
	for _, file := range findYAMLFiles(t) {
		t.Run(file, func(t *testing.T) {
			data := loadDataFile(t, file)
			for _, p := range data.Persons {
				if p.Birthdate == "" {
					continue
				}
				if !easternArabicNumber.MatchString(p.Birthdate) {
					t.Errorf("%s: birthdate %q must use Eastern Arabic numerals (٠-٩)", p.Name, p.Birthdate)
				}
			}
		})
	}
}

func TestConnectivity(t *testing.T) {
	for _, file := range findYAMLFiles(t) {
		t.Run(file, func(t *testing.T) {
			data := loadDataFile(t, file)
			persons := data.Persons

			ids := make(map[string]bool, len(persons))
			motherIDs := make(map[string]bool)
			for _, p := range persons {
				ids[p.ID] = true
				if p.MotherID != "" {
					motherIDs[p.MotherID] = true
				}
			}

			// Check for orphan females
			for _, p := range persons {
				if p.Sex == "female" && p.FatherID == "" && !motherIDs[p.ID] {
					t.Errorf("%s (%s): female without fatherId and not referenced as motherId", p.Name, p.ID)
				}
			}

			// Build undirected adjacency list
			adj := make(map[string][]string, len(persons))
			for _, p := range persons {
				if p.FatherID != "" && ids[p.FatherID] {
					adj[p.ID] = append(adj[p.ID], p.FatherID)
					adj[p.FatherID] = append(adj[p.FatherID], p.ID)
				}
				if p.MotherID != "" && ids[p.MotherID] {
					adj[p.ID] = append(adj[p.ID], p.MotherID)
					adj[p.MotherID] = append(adj[p.MotherID], p.ID)
				}
			}

			// BFS from first person
			visited := make(map[string]bool, len(persons))
			queue := []string{persons[0].ID}
			visited[persons[0].ID] = true
			for len(queue) > 0 {
				current := queue[0]
				queue = queue[1:]
				for _, neighbor := range adj[current] {
					if !visited[neighbor] {
						visited[neighbor] = true
						queue = append(queue, neighbor)
					}
				}
			}

			for _, p := range persons {
				if !visited[p.ID] {
					t.Errorf("%s (%s): disconnected from the tree", p.Name, p.ID)
				}
			}
		})
	}
}
