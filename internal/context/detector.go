package context

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Detector detects frameworks and languages in a repository
type Detector struct {
	rootPath string
}

// NewDetector creates a new framework/language detector
func NewDetector(rootPath string) *Detector {
	return &Detector{
		rootPath: rootPath,
	}
}

// DetectionResult contains detected frameworks and languages
type DetectionResult struct {
	Languages  map[Language]int       `json:"languages"`
	Frameworks []Framework            `json:"frameworks"`
	Patterns   []string               `json:"patterns"`
	Files      map[string]interface{} `json:"files"`
}

// Detect performs framework and language detection
func (d *Detector) Detect() (*DetectionResult, error) {
	result := &DetectionResult{
		Languages:  make(map[Language]int),
		Frameworks: make([]Framework, 0),
		Patterns:   make([]string, 0),
		Files:      make(map[string]interface{}),
	}

	// Scan repository for files
	files, err := d.scanRepository()
	if err != nil {
		return nil, fmt.Errorf("failed to scan repository: %w", err)
	}

	// Detect languages
	result.Languages = DetectLanguages(files)

	// Detect frameworks
	frameworks, err := d.detectFrameworks(files)
	if err != nil {
		return nil, fmt.Errorf("failed to detect frameworks: %w", err)
	}
	result.Frameworks = frameworks

	// Detect patterns
	patterns := d.detectPatterns(frameworks)
	result.Patterns = patterns

	// Store important files
	result.Files = d.findImportantFiles()

	return result, nil
}

// scanRepository scans the repository and returns all source files
func (d *Detector) scanRepository() ([]string, error) {
	files := make([]string, 0)

	err := filepath.Walk(d.rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip hidden directories and common ignore patterns
		if info.IsDir() {
			name := info.Name()
			if strings.HasPrefix(name, ".") || 
			   name == "node_modules" || 
			   name == "vendor" || 
			   name == "dist" || 
			   name == "build" ||
			   name == "target" ||
			   name == "__pycache__" {
				return filepath.SkipDir
			}
			return nil
		}

		// Only include source files
		if IsSourceFile(path) {
			relPath, _ := filepath.Rel(d.rootPath, path)
			files = append(files, relPath)
		}

		return nil
	})

	return files, err
}

// detectFrameworks detects frameworks based on files and content
func (d *Detector) detectFrameworks(files []string) ([]Framework, error) {
	frameworks := make([]Framework, 0)
	detected := make(map[string]bool)

	registry := GetFrameworkRegistry()

	// Check package files first (most reliable)
	packageFrameworks := d.detectFromPackageFiles()
	for _, fw := range packageFrameworks {
		if !detected[fw.Name] {
			frameworks = append(frameworks, fw)
			detected[fw.Name] = true
		}
	}

	// Check file patterns and content
	for _, fwInfo := range registry {
		if detected[fwInfo.Name] {
			continue
		}

		// Check for indicator patterns in files
		for _, file := range files {
			if detected[fwInfo.Name] {
				break
			}

			content, err := d.readFile(file)
			if err != nil {
				continue
			}

			// Check if any indicator is present
			for _, indicator := range fwInfo.Indicators {
				if strings.Contains(content, indicator) {
					frameworks = append(frameworks, Framework{
						Name:     fwInfo.Name,
						Type:     fwInfo.Type,
						Language: fwInfo.Language,
					})
					detected[fwInfo.Name] = true
					break
				}
			}
		}
	}

	return frameworks, nil
}

// detectFromPackageFiles detects frameworks from package.json, go.mod, requirements.txt, etc.
func (d *Detector) detectFromPackageFiles() []Framework {
	frameworks := make([]Framework, 0)

	// Check package.json (Node.js)
	packageJSON := d.readPackageJSON()
	if packageJSON != nil {
		frameworks = append(frameworks, d.detectFromNodePackages(packageJSON)...)
	}

	// Check go.mod (Go)
	goMod := d.readGoMod()
	if goMod != "" {
		frameworks = append(frameworks, d.detectFromGoMod(goMod)...)
	}

	// Check requirements.txt or pyproject.toml (Python)
	pythonDeps := d.readPythonDeps()
	if len(pythonDeps) > 0 {
		frameworks = append(frameworks, d.detectFromPythonDeps(pythonDeps)...)
	}

	// Check pom.xml or build.gradle (Java)
	javaDeps := d.readJavaDeps()
	if javaDeps != "" {
		frameworks = append(frameworks, d.detectFromJavaDeps(javaDeps)...)
	}

	return frameworks
}

// readPackageJSON reads and parses package.json
func (d *Detector) readPackageJSON() map[string]interface{} {
	path := filepath.Join(d.rootPath, "package.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	var pkg map[string]interface{}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return nil
	}

	return pkg
}

// detectFromNodePackages detects frameworks from package.json
func (d *Detector) detectFromNodePackages(pkg map[string]interface{}) []Framework {
	frameworks := make([]Framework, 0)
	registry := GetFrameworkRegistry()

	// Get dependencies
	deps := make(map[string]bool)
	if dependencies, ok := pkg["dependencies"].(map[string]interface{}); ok {
		for dep := range dependencies {
			deps[dep] = true
		}
	}
	if devDeps, ok := pkg["devDependencies"].(map[string]interface{}); ok {
		for dep := range devDeps {
			deps[dep] = true
		}
	}

	// Match against registry
	for _, fwInfo := range registry {
		for _, pkgKey := range fwInfo.PackageKeys {
			if deps[pkgKey] {
				frameworks = append(frameworks, Framework{
					Name:     fwInfo.Name,
					Type:     fwInfo.Type,
					Language: fwInfo.Language,
				})
				break
			}
		}
	}

	return frameworks
}

// readGoMod reads go.mod file
func (d *Detector) readGoMod() string {
	path := filepath.Join(d.rootPath, "go.mod")
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(data)
}

// detectFromGoMod detects frameworks from go.mod
func (d *Detector) detectFromGoMod(content string) []Framework {
	frameworks := make([]Framework, 0)
	registry := GetFrameworkRegistry()

	for _, fwInfo := range registry {
		if fwInfo.Language != LanguageGo {
			continue
		}

		for _, pkgKey := range fwInfo.PackageKeys {
			if strings.Contains(content, pkgKey) {
				frameworks = append(frameworks, Framework{
					Name:     fwInfo.Name,
					Type:     fwInfo.Type,
					Language: fwInfo.Language,
				})
				break
			}
		}
	}

	return frameworks
}

// readPythonDeps reads Python dependencies
func (d *Detector) readPythonDeps() []string {
	deps := make([]string, 0)

	// Try requirements.txt
	reqPath := filepath.Join(d.rootPath, "requirements.txt")
	if data, err := os.ReadFile(reqPath); err == nil {
		lines := strings.Split(string(data), "\n")
		deps = append(deps, lines...)
	}

	// Try pyproject.toml
	pyprojectPath := filepath.Join(d.rootPath, "pyproject.toml")
	if data, err := os.ReadFile(pyprojectPath); err == nil {
		deps = append(deps, string(data))
	}

	return deps
}

// detectFromPythonDeps detects frameworks from Python dependencies
func (d *Detector) detectFromPythonDeps(deps []string) []Framework {
	frameworks := make([]Framework, 0)
	registry := GetFrameworkRegistry()

	depsStr := strings.Join(deps, "\n")

	for _, fwInfo := range registry {
		if fwInfo.Language != LanguagePython {
			continue
		}

		for _, pkgKey := range fwInfo.PackageKeys {
			if strings.Contains(depsStr, pkgKey) {
				frameworks = append(frameworks, Framework{
					Name:     fwInfo.Name,
					Type:     fwInfo.Type,
					Language: fwInfo.Language,
				})
				break
			}
		}
	}

	return frameworks
}

// readJavaDeps reads Java dependencies
func (d *Detector) readJavaDeps() string {
	// Try pom.xml
	pomPath := filepath.Join(d.rootPath, "pom.xml")
	if data, err := os.ReadFile(pomPath); err == nil {
		return string(data)
	}

	// Try build.gradle
	gradlePath := filepath.Join(d.rootPath, "build.gradle")
	if data, err := os.ReadFile(gradlePath); err == nil {
		return string(data)
	}

	return ""
}

// detectFromJavaDeps detects frameworks from Java dependencies
func (d *Detector) detectFromJavaDeps(content string) []Framework {
	frameworks := make([]Framework, 0)
	registry := GetFrameworkRegistry()

	for _, fwInfo := range registry {
		if fwInfo.Language != LanguageJava {
			continue
		}

		for _, pkgKey := range fwInfo.PackageKeys {
			if strings.Contains(content, pkgKey) {
				frameworks = append(frameworks, Framework{
					Name:     fwInfo.Name,
					Type:     fwInfo.Type,
					Language: fwInfo.Language,
				})
				break
			}
		}
	}

	return frameworks
}

// detectPatterns detects architectural patterns based on frameworks
func (d *Detector) detectPatterns(frameworks []Framework) []string {
	patterns := make([]string, 0)
	detected := make(map[string]bool)

	for _, fw := range frameworks {
		var pattern string
		
		switch fw.Name {
		case FrameworkSpringBoot:
			pattern = "Controller → Service → Repository"
		case FrameworkExpress, FrameworkFastAPI, FrameworkGin:
			pattern = "Router → Handler → Service"
		case FrameworkReact, FrameworkVue, FrameworkAngular:
			pattern = "Component-based Architecture"
		case FrameworkNextJS, FrameworkNuxt:
			pattern = "File-based Routing + SSR/SSG"
		}

		if pattern != "" && !detected[pattern] {
			patterns = append(patterns, pattern)
			detected[pattern] = true
		}
	}

	return patterns
}

// findImportantFiles finds configuration and important files
func (d *Detector) findImportantFiles() map[string]interface{} {
	files := make(map[string]interface{})

	importantFiles := []string{
		"package.json",
		"go.mod",
		"requirements.txt",
		"pyproject.toml",
		"pom.xml",
		"build.gradle",
		"Cargo.toml",
		"tsconfig.json",
		"next.config.js",
		"vite.config.js",
		"tailwind.config.js",
		"angular.json",
		"nuxt.config.js",
	}

	for _, file := range importantFiles {
		path := filepath.Join(d.rootPath, file)
		if _, err := os.Stat(path); err == nil {
			files[file] = true
		}
	}

	return files
}

// readFile reads a file relative to root path
func (d *Detector) readFile(relPath string) (string, error) {
	path := filepath.Join(d.rootPath, relPath)
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
