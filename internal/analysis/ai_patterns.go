package analysis

import (
	"regexp"
	"strings"
)

// Pattern detection helpers for AI-generated code

// hasExcessiveComments checks if the function has too many inline comments
func hasExcessiveComments(body string, loc int) bool {
	if body == "" || loc == 0 {
		return false
	}

	lines := strings.Split(body, "\n")
	commentCount := 0

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "/*") || strings.HasPrefix(trimmed, "*") {
			commentCount++
		}
	}

	// If more than 40% of lines are comments, it's excessive
	commentRatio := float64(commentCount) / float64(loc)
	return commentRatio > 0.4
}

// hasGenericComments checks for generic placeholder comments
func hasGenericComments(comments string) bool {
	if comments == "" {
		return false
	}

	genericPhrases := []string{
		"TODO: implement",
		"Add error handling here",
		"Handle edge cases",
		"This function",
		"This method",
		"Implement logic",
		"Add validation",
		"Process the",
		"Handle the",
	}

	commentsLower := strings.ToLower(comments)
	for _, phrase := range genericPhrases {
		if strings.Contains(commentsLower, strings.ToLower(phrase)) {
			return true
		}
	}

	return false
}

// hasVerboseDocstring checks if docstring just restates the function name
func hasVerboseDocstring(funcName, comments string) bool {
	if comments == "" || funcName == "" {
		return false
	}

	// Remove common prefixes to get the action word
	cleanName := strings.TrimPrefix(funcName, "get")
	cleanName = strings.TrimPrefix(cleanName, "set")
	cleanName = strings.TrimPrefix(cleanName, "handle")
	cleanName = strings.TrimPrefix(cleanName, "process")

	commentsLower := strings.ToLower(comments)
	funcNameLower := strings.ToLower(funcName)

	// Check if comment just repeats the function name
	if strings.Contains(commentsLower, funcNameLower) {
		// Count words - if very few words, likely just restating
		words := strings.Fields(comments)
		if len(words) < 15 && strings.Contains(commentsLower, funcNameLower) {
			return true
		}
	}

	return false
}

// hasGenericPrefix checks for generic function name prefixes
func hasGenericPrefix(funcName string) bool {
	genericPrefixes := []string{
		"handle", "process", "manage", "perform",
		"execute", "do", "run", "invoke",
	}

	nameLower := strings.ToLower(funcName)
	for _, prefix := range genericPrefixes {
		if strings.HasPrefix(nameLower, prefix) {
			return true
		}
	}

	return false
}

// isOverlyDescriptive checks for overly long/descriptive function names
func isOverlyDescriptive(funcName string) bool {
	// Names longer than 40 chars are usually too descriptive
	if len(funcName) > 40 {
		return true
	}

	// Check for patterns like "getUserByIdAndReturnUserObject"
	if strings.Count(funcName, "And") > 1 || strings.Count(funcName, "Or") > 1 {
		return true
	}

	return false
}

// hasExcessiveNullChecks detects defensive null checking patterns
func hasExcessiveNullChecks(body string) bool {
	if body == "" {
		return false
	}

	nullCheckPatterns := []string{
		"== null", "!= null", "is null", "is not null",
		"== nil", "!= nil", "== None", "!= None",
		"IsNullOrEmpty", "isNull", "isNotNull",
	}

	count := 0
	bodyLower := strings.ToLower(body)

	for _, pattern := range nullCheckPatterns {
		count += strings.Count(bodyLower, strings.ToLower(pattern))
	}

	// More than 3 null checks in one function is excessive
	return count > 3
}

// hasExcessiveTryCatch detects try-catch around every statement
func hasExcessiveTryCatch(body string) bool {
	if body == "" {
		return false
	}

	// Count try-catch blocks
	tryCount := strings.Count(strings.ToLower(body), "try {")
	tryCount += strings.Count(strings.ToLower(body), "try{")

	lines := strings.Split(body, "\n")
	nonEmptyLines := 0
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			nonEmptyLines++
		}
	}

	if nonEmptyLines == 0 {
		return false
	}

	// More than 1 try-catch per 10 lines is excessive
	ratio := float64(tryCount) / float64(nonEmptyLines) * 10
	return ratio > 1.0
}

// hasGenericExceptions detects generic exception messages
func hasGenericExceptions(body string) bool {
	if body == "" {
		return false
	}

	genericMessages := []string{
		"\"Error\"",
		"\"Exception\"",
		"\"Something went wrong\"",
		"\"An error occurred\"",
		"\"Failed\"",
		"\"Invalid input\"",
		"'Error'",
		"'Exception'",
		"'Something went wrong'",
	}

	bodyLower := strings.ToLower(body)
	for _, msg := range genericMessages {
		if strings.Contains(bodyLower, strings.ToLower(msg)) {
			return true
		}
	}

	return false
}

// detectJavaPatterns detects Java-specific AI patterns
func detectJavaPatterns(fn FunctionInfo) (bool, []string) {
	indicators := []string{}

	if fn.Body == "" {
		return false, indicators
	}

	body := fn.Body

	// 1. Builder pattern usage (common in AI-generated code)
	if strings.Contains(body, ".builder()") && strings.Contains(body, ".build()") {
		// Count chained method calls in builder
		builderChainCount := strings.Count(body, ".builder()") + 
			strings.Count(body, ".build()")
		// If there are many builder calls, likely AI-generated
		if builderChainCount >= 1 {
			indicators = append(indicators, "Builder pattern usage")
		}
	}

	// 2. Repository method patterns (findById, countBy*, etc.)
	repositoryPatterns := []string{
		"Repository.findById(",
		"Repository.countBy",
		"Repository.findBy",
		"Repository.deleteBy",
		"Repository.existsBy",
	}
	repoPatternCount := 0
	for _, pattern := range repositoryPatterns {
		if strings.Contains(body, pattern) {
			repoPatternCount++
		}
	}
	if repoPatternCount >= 2 {
		indicators = append(indicators, "Multiple repository pattern methods")
	}

	// 3. Optional with orElseThrow pattern (very common in AI code)
	if strings.Contains(body, ".orElseThrow(") {
		indicators = append(indicators, "Optional.orElseThrow pattern")
	}

	// 4. DTO usage with consistent naming
	if strings.Contains(body, "DTO") {
		indicators = append(indicators, "DTO pattern usage")
	}

	// 5. Method chaining (more than 3 dots in sequence)
	chainPattern := regexp.MustCompile(`\.\w+\([^)]*\)\s*\.\w+\([^)]*\)\s*\.\w+\([^)]*\)`)
	if chainPattern.MatchString(body) {
		indicators = append(indicators, "Method chaining pattern")
	}

	// 6. Lambda expressions with specific patterns
	if strings.Contains(body, "() ->") || strings.Contains(body, "-> new") {
		indicators = append(indicators, "Lambda expression pattern")
	}

	// 7. Consistent variable naming pattern (all use same suffix)
	// e.g., campusCount, totalUsers, totalStudents, activeStudents
	varPattern := regexp.MustCompile(`\b(total|active|count|number|is|has)\w+\s*=`)
	matches := varPattern.FindAllString(body, -1)
	if len(matches) >= 3 {
		indicators = append(indicators, "Consistent variable naming pattern")
	}

	// OLD PATTERNS

	// Unnecessary this. prefixes
	thisCount := strings.Count(body, "this.")
	if thisCount > 5 {
		indicators = append(indicators, "Excessive 'this.' prefixes")
	}

	// Verbose getter/setter instead of Lombok
	if (strings.HasPrefix(fn.Name, "get") || strings.HasPrefix(fn.Name, "set")) &&
		fn.LOC < 5 && len(fn.Parameters) <= 1 {
		// Simple getter/setter - could use Lombok
		if strings.Count(body, "return") == 1 || strings.Count(body, "this.") > 0 {
			indicators = append(indicators, "Verbose getter/setter (consider Lombok)")
		}
	}

	// Explicit type declarations where inference would work
	diamondPattern := regexp.MustCompile(`new\s+\w+<[\w\s,]+>\s*\(`)
	if diamondPattern.MatchString(body) {
		indicators = append(indicators, "Explicit type parameters (diamond operator could be used)")
	}

	return len(indicators) > 0, indicators
}

// detectJSPatterns detects JavaScript/TypeScript-specific AI patterns
func detectJSPatterns(fn FunctionInfo) (bool, []string) {
	indicators := []string{}

	if fn.Body == "" {
		return false, indicators
	}

	body := fn.Body

	// 1. Arrow functions everywhere
	arrowCount := strings.Count(body, "=>")
	if arrowCount >= 2 {
		indicators = append(indicators, "Multiple arrow functions")
	}

	// 2. Array method chaining (.map().filter().reduce())
	chainPatterns := []string{".map(", ".filter(", ".reduce(", ".forEach("}
	chainCount := 0
	for _, pattern := range chainPatterns {
		chainCount += strings.Count(body, pattern)
	}
	if chainCount >= 2 {
		indicators = append(indicators, "Array method chaining")
	}

	// 3. Async/await patterns
	if strings.Contains(body, "async ") && strings.Contains(body, "await ") {
		indicators = append(indicators, "Async/await pattern")
	}

	// 4. Destructuring usage
	destructuringPattern := regexp.MustCompile(`const\s*\{[^}]+\}\s*=`)
	if destructuringPattern.MatchString(body) {
		indicators = append(indicators, "Destructuring pattern")
	}

	// 5. Template literals
	if strings.Count(body, "`") >= 2 {
		indicators = append(indicators, "Template literals usage")
	}

	// 6. Spread operator
	if strings.Contains(body, "...") {
		indicators = append(indicators, "Spread operator usage")
	}

	// 7. Optional chaining (?.)
	if strings.Contains(body, "?.") {
		indicators = append(indicators, "Optional chaining")
	}

	// 8. Nullish coalescing (??)
	if strings.Contains(body, "??") {
		indicators = append(indicators, "Nullish coalescing operator")
	}

	// 9. Consistent const/let usage (no var)
	hasConst := strings.Contains(body, "const ")
	hasLet := strings.Contains(body, "let ")
	hasVar := strings.Contains(body, "var ")
	if (hasConst || hasLet) && !hasVar {
		indicators = append(indicators, "Modern variable declarations (const/let)")
	}

	// 10. Try-catch with specific patterns
	if strings.Contains(body, "try {") && strings.Contains(body, "} catch") {
		indicators = append(indicators, "Try-catch error handling")
	}

	// OLD PATTERNS (keep for backwards compatibility)

	// Excessive async/await
	asyncCount := strings.Count(body, "await ")
	lines := strings.Split(body, "\n")
	if asyncCount > len(lines)/3 {
		indicators = append(indicators, "Excessive async/await usage")
	}

	// Verbose promise chains
	thenCount := strings.Count(body, ".then(")
	if thenCount > 3 {
		indicators = append(indicators, "Long promise chain")
	}

	return len(indicators) > 0, indicators
}

// detectGoPatterns detects Go-specific AI patterns
func detectGoPatterns(fn FunctionInfo) (bool, []string) {
	indicators := []string{}

	if fn.Body == "" {
		return false, indicators
	}

	body := fn.Body

	// 1. Consistent error handling after every call
	ifErrCount := strings.Count(body, "if err != nil")
	if ifErrCount >= 2 {
		indicators = append(indicators, "Multiple error checks")
	}

	// 2. Defer usage
	if strings.Contains(body, "defer ") {
		indicators = append(indicators, "Defer statement usage")
	}

	// 3. Struct initialization patterns
	structPattern := regexp.MustCompile(`\w+\{`)
	matches := structPattern.FindAllString(body, -1)
	if len(matches) >= 2 {
		indicators = append(indicators, "Struct initialization pattern")
	}

	// 4. Method receiver patterns
	if strings.Contains(body, "func (") && strings.Contains(body, ") ") {
		indicators = append(indicators, "Method receiver pattern")
	}

	// 5. Context usage
	if strings.Contains(body, "context.") || strings.Contains(body, "ctx ") {
		indicators = append(indicators, "Context usage")
	}

	// 6. Error wrapping patterns
	if strings.Contains(body, "fmt.Errorf(") || strings.Contains(body, "errors.Wrap") {
		indicators = append(indicators, "Error wrapping pattern")
	}

	// 7. Multiple return values
	returnPattern := regexp.MustCompile(`return\s+\w+,\s*\w+`)
	if returnPattern.MatchString(body) {
		indicators = append(indicators, "Multiple return values")
	}

	// 8. Interface usage
	if strings.Contains(body, "interface{") || strings.Contains(body, "any") {
		indicators = append(indicators, "Interface usage")
	}

	// 9. Goroutine usage
	if strings.Contains(body, "go func(") || strings.Contains(body, "go ") {
		indicators = append(indicators, "Goroutine pattern")
	}

	// 10. Consistent nil checks
	nilCheckCount := strings.Count(body, "!= nil") + strings.Count(body, "== nil")
	if nilCheckCount >= 2 {
		indicators = append(indicators, "Multiple nil checks")
	}

	// OLD PATTERNS

	// Inconsistent error handling
	returnErrCount := strings.Count(body, "return err")
	if ifErrCount > 0 && returnErrCount > 0 {
		if ifErrCount != returnErrCount {
			indicators = append(indicators, "Inconsistent error handling")
		}
	}

	return len(indicators) > 0, indicators
}

// detectPythonPatterns detects Python-specific AI patterns
func detectPythonPatterns(fn FunctionInfo) (bool, []string) {
	indicators := []string{}

	if fn.Body == "" {
		return false, indicators
	}

	body := fn.Body

	// 1. List comprehensions
	if strings.Contains(body, "for ") && (strings.Contains(body, " in ") && strings.Contains(body, "]")) {
		indicators = append(indicators, "List comprehension usage")
	}

	// 2. F-string usage
	fstringPattern := regexp.MustCompile(`f["']`)
	if fstringPattern.MatchString(body) {
		indicators = append(indicators, "F-string formatting")
	}

	// 3. Type hints everywhere
	typeHintPattern := regexp.MustCompile(`->\s*\w+:`)
	if typeHintPattern.MatchString(body) || strings.Contains(body, ": int") || 
	   strings.Contains(body, ": str") || strings.Contains(body, ": List") {
		indicators = append(indicators, "Type hints usage")
	}

	// 4. Try-except-else pattern
	if strings.Contains(body, "try:") && strings.Contains(body, "except") {
		if strings.Contains(body, "else:") {
			indicators = append(indicators, "Try-except-else pattern")
		} else {
			indicators = append(indicators, "Try-except error handling")
		}
	}

	// 5. Pathlib usage
	if strings.Contains(body, "Path(") || strings.Contains(body, "pathlib") {
		indicators = append(indicators, "Pathlib usage")
	}

	// 6. Dataclass usage
	if strings.Contains(body, "@dataclass") || strings.Contains(body, "dataclass") {
		indicators = append(indicators, "Dataclass pattern")
	}

	// 7. Context manager (with statement)
	if strings.Contains(body, "with ") && strings.Contains(body, " as ") {
		indicators = append(indicators, "Context manager pattern")
	}

	// 8. Dict comprehension
	if strings.Contains(body, "for ") && strings.Contains(body, " in ") && 
	   (strings.Contains(body, "{") && strings.Contains(body, ":")) {
		indicators = append(indicators, "Dict comprehension")
	}

	// 9. Decorator usage
	decoratorPattern := regexp.MustCompile(`@\w+`)
	matches := decoratorPattern.FindAllString(body, -1)
	if len(matches) >= 2 {
		indicators = append(indicators, "Multiple decorators")
	}

	// 10. Consistent use of None checks
	if strings.Contains(body, "is None") || strings.Contains(body, "is not None") {
		indicators = append(indicators, "Proper None comparison")
	}

	// OLD PATTERNS

	// Unnecessary lambda usage
	lambdaCount := strings.Count(body, "lambda ")
	if lambdaCount > 2 {
		indicators = append(indicators, "Multiple lambda functions")
	}

	return len(indicators) > 0, indicators
}

// detectCSharpPatterns detects C#-specific AI patterns
func detectCSharpPatterns(fn FunctionInfo) (bool, []string) {
	indicators := []string{}

	if fn.Body == "" {
		return false, indicators
	}

	body := fn.Body

	// 1. LINQ usage
	linqPatterns := []string{".Where(", ".Select(", ".FirstOrDefault(", ".Any(", ".Count(", ".OrderBy("}
	linqCount := 0
	for _, pattern := range linqPatterns {
		if strings.Contains(body, pattern) {
			linqCount++
		}
	}
	if linqCount >= 2 {
		indicators = append(indicators, "LINQ query methods")
	}

	// 2. Async/await patterns
	if strings.Contains(body, "async ") && strings.Contains(body, "await ") {
		indicators = append(indicators, "Async/await pattern")
	}

	// 3. Null-coalescing operator (??)
	if strings.Contains(body, "??") {
		indicators = append(indicators, "Null-coalescing operator")
	}

	// 4. Null-conditional operator (?.)
	if strings.Contains(body, "?.") {
		indicators = append(indicators, "Null-conditional operator")
	}

	// 5. Property patterns and init-only setters
	if strings.Contains(body, "{ get; set; }") || strings.Contains(body, "{ get; init; }") {
		indicators = append(indicators, "Property pattern")
	}

	// 6. Expression-bodied members (=>)
	if strings.Contains(body, "=>") {
		indicators = append(indicators, "Expression-bodied members")
	}

	// 7. var usage everywhere
	varCount := strings.Count(body, "var ")
	if varCount >= 2 {
		indicators = append(indicators, "Consistent var usage")
	}

	// 8. String interpolation ($"")
	if strings.Contains(body, "$\"") {
		indicators = append(indicators, "String interpolation")
	}

	// 9. Pattern matching (is, switch expressions)
	if strings.Contains(body, " is ") || strings.Contains(body, "switch") {
		indicators = append(indicators, "Pattern matching")
	}

	// 10. Try-catch-finally pattern
	if strings.Contains(body, "try") && strings.Contains(body, "catch") {
		indicators = append(indicators, "Try-catch error handling")
	}

	// 11. Dependency injection patterns
	if strings.Contains(body, "I") && strings.Contains(body, "Service") {
		indicators = append(indicators, "Service interface pattern")
	}

	// 12. Task usage (async programming)
	if strings.Contains(body, "Task<") || strings.Contains(body, "Task.") {
		indicators = append(indicators, "Task-based async pattern")
	}

	return len(indicators) > 0, indicators
}

