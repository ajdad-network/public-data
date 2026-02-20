package cmd_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"unicode"

	"gopkg.in/yaml.v3"
)

type Family struct {
	ID           string `yaml:"id"`
	Name         string `yaml:"name"`
	Type         string `yaml:"type"`
	Category     string `yaml:"category"`
	Visibility   string `yaml:"visibility"`
	SuffixMale   string `yaml:"suffixMale"`
	SuffixFemale string `yaml:"suffixFemale"`
	Hometown     string `yaml:"hometown"`
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
	uuidV4              = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

	validFamilyTypes      = map[string]bool{"aal": true, "aila": true, "qabila": true, "ashira": true, "usra": true}
	validCategories       = map[string]bool{"ruling": true, "historical": true, "tribes": true, "other": true}
	validVisibilities     = map[string]bool{"public": true, "private": true}
	validSexes            = map[string]bool{"male": true, "female": true}
	categoryFromDirectory = map[string]string{"ruling-families": "ruling", "tribes": "tribes", "historical": "historical"}
)

func containsArabic(s string) bool {
	for _, r := range s {
		if unicode.Is(unicode.Arabic, r) {
			return true
		}
	}
	return false
}

// repoRoot returns the repository root (one level up from cmd/).
func repoRoot() string {
	return ".."
}

func findYAMLFiles(t *testing.T) []string {
	t.Helper()
	root := repoRoot()
	var files []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		base := filepath.Base(path)
		if info.IsDir() && (base == "node_modules" || base == ".git" || base == ".github" || base == "cmd") {
			return filepath.SkipDir
		}
		if filepath.Ext(path) == ".yaml" && filepath.Base(path) != "all-trees.yaml" {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("finding YAML files: %v", err)
	}
	return files
}

// relFromRoot returns a file path relative to the repo root.
func relFromRoot(path string) string {
	rel, err := filepath.Rel(repoRoot(), path)
	if err != nil {
		return path
	}
	return rel
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
					"hometown":     data.Family.Hometown,
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

func TestFamilyRequiredFields(t *testing.T) {
	for _, file := range findYAMLFiles(t) {
		t.Run(file, func(t *testing.T) {
			data := loadDataFile(t, file)
			f := data.Family

			if f.ID == "" {
				t.Error("family.id is required")
			} else if !uuidV4.MatchString(f.ID) {
				t.Errorf("family.id: %q is not a valid UUIDv4", f.ID)
			}

			if f.Name == "" {
				t.Error("family.name is required")
			}

			if f.Type == "" {
				t.Error("family.type is required")
			} else if !validFamilyTypes[f.Type] {
				t.Errorf("family.type: %q must be one of: aal, aila, qabila, ashira, usra", f.Type)
			}

			if f.Category == "" {
				t.Error("family.category is required")
			} else if !validCategories[f.Category] {
				t.Errorf("family.category: %q must be one of: ruling, historical, tribes, other", f.Category)
			}

			if f.Visibility != "" && !validVisibilities[f.Visibility] {
				t.Errorf("family.visibility: %q must be public or private", f.Visibility)
			}
		})
	}
}

func TestCategoryMatchesDirectory(t *testing.T) {
	for _, file := range findYAMLFiles(t) {
		t.Run(file, func(t *testing.T) {
			data := loadDataFile(t, file)
			rel := relFromRoot(file)
			topDir := strings.SplitN(filepath.ToSlash(rel), "/", 2)[0]
			expected, ok := categoryFromDirectory[topDir]
			if !ok {
				return // file not in a known category directory
			}
			if data.Family.Category != expected {
				t.Errorf("family.category: %q does not match directory %q (expected %q)", data.Family.Category, topDir, expected)
			}
		})
	}
}

func TestPersonRequiredFields(t *testing.T) {
	for _, file := range findYAMLFiles(t) {
		t.Run(file, func(t *testing.T) {
			data := loadDataFile(t, file)
			for _, p := range data.Persons {
				if p.ID == "" {
					t.Errorf("person %q: id is required", p.Name)
				} else if !uuidV4.MatchString(p.ID) {
					t.Errorf("person %q: id %q is not a valid UUIDv4", p.Name, p.ID)
				}

				if p.Name == "" {
					t.Errorf("person %s: name is required", p.ID)
				}

				if p.Sex == "" {
					t.Errorf("%s: sex is required", p.Name)
				} else if !validSexes[p.Sex] {
					t.Errorf("%s: sex %q must be male or female", p.Name, p.Sex)
				}
			}
		})
	}
}

func TestPersonUniqueIDs(t *testing.T) {
	for _, file := range findYAMLFiles(t) {
		t.Run(file, func(t *testing.T) {
			data := loadDataFile(t, file)
			seen := make(map[string]string, len(data.Persons))
			for _, p := range data.Persons {
				if prev, exists := seen[p.ID]; exists {
					t.Errorf("duplicate id %s: used by %q and %q", p.ID, prev, p.Name)
				}
				seen[p.ID] = p.Name
			}
		})
	}
}

func TestReferentialIntegrity(t *testing.T) {
	for _, file := range findYAMLFiles(t) {
		t.Run(file, func(t *testing.T) {
			data := loadDataFile(t, file)
			ids := make(map[string]bool, len(data.Persons))
			for _, p := range data.Persons {
				ids[p.ID] = true
			}
			for _, p := range data.Persons {
				if p.FatherID != "" && !ids[p.FatherID] {
					t.Errorf("%s: fatherId %q does not reference a valid person", p.Name, p.FatherID)
				}
				if p.MotherID != "" && !ids[p.MotherID] {
					t.Errorf("%s: motherId %q does not reference a valid person", p.Name, p.MotherID)
				}
			}
		})
	}
}

func TestTopologicalOrder(t *testing.T) {
	for _, file := range findYAMLFiles(t) {
		t.Run(file, func(t *testing.T) {
			data := loadDataFile(t, file)
			seen := make(map[string]bool, len(data.Persons))
			for _, p := range data.Persons {
				if p.FatherID != "" && !seen[p.FatherID] {
					t.Errorf("%s: fatherId %q appears after child in persons list", p.Name, p.FatherID)
				}
				if p.MotherID != "" && !seen[p.MotherID] {
					t.Errorf("%s: motherId %q appears after child in persons list", p.Name, p.MotherID)
				}
				seen[p.ID] = true
			}
		})
	}
}
