package context

import (
	"path/filepath"
	"strings"
)

// Language represents a programming language
type Language string

const (
	LanguageGo         Language = "Go"
	LanguageJava       Language = "Java"
	LanguagePython     Language = "Python"
	LanguageJavaScript Language = "JavaScript"
	LanguageTypeScript Language = "TypeScript"
	LanguageRust       Language = "Rust"
	LanguageKotlin     Language = "Kotlin"
	LanguageSwift      Language = "Swift"
	LanguageRuby       Language = "Ruby"
	LanguagePHP        Language = "PHP"
	LanguageC          Language = "C"
	LanguageCPP        Language = "C++"
	LanguageCSharp     Language = "C#"
	LanguageUnknown    Language = "Unknown"
)

// languageExtensions maps file extensions to languages
var languageExtensions = map[string]Language{
	".go":   LanguageGo,
	".java": LanguageJava,
	".py":   LanguagePython,
	".js":   LanguageJavaScript,
	".jsx":  LanguageJavaScript,
	".ts":   LanguageTypeScript,
	".tsx":  LanguageTypeScript,
	".rs":   LanguageRust,
	".kt":   LanguageKotlin,
	".kts":  LanguageKotlin,
	".swift": LanguageSwift,
	".rb":   LanguageRuby,
	".php":  LanguagePHP,
	".c":    LanguageC,
	".h":    LanguageC,
	".cpp":  LanguageCPP,
	".cc":   LanguageCPP,
	".cxx":  LanguageCPP,
	".hpp":  LanguageCPP,
	".cs":   LanguageCSharp,
}

// DetectLanguage detects the programming language from a file path
func DetectLanguage(filePath string) Language {
	ext := strings.ToLower(filepath.Ext(filePath))
	if lang, ok := languageExtensions[ext]; ok {
		return lang
	}
	return LanguageUnknown
}

// DetectLanguages detects all languages in a list of file paths
func DetectLanguages(filePaths []string) map[Language]int {
	languages := make(map[Language]int)
	
	for _, path := range filePaths {
		lang := DetectLanguage(path)
		if lang != LanguageUnknown {
			languages[lang]++
		}
	}
	
	return languages
}

// GetPrimaryLanguage returns the most common language from a map
func GetPrimaryLanguage(languages map[Language]int) Language {
	var primary Language
	maxCount := 0
	
	for lang, count := range languages {
		if count > maxCount {
			maxCount = count
			primary = lang
		}
	}
	
	return primary
}

// IsSourceFile checks if a file is a source code file
func IsSourceFile(filePath string) bool {
	return DetectLanguage(filePath) != LanguageUnknown
}
