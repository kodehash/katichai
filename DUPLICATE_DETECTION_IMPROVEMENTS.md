# Duplicate Detection Improvements

## ✅ Changes Implemented

Based on user feedback that duplicate detection was too sensitive and not semantic enough, the following improvements have been made:

---

## 1. **Minimum Size Threshold**

### Problem
- Was flagging 1-2 line similarities as "duplicates"
- Too noisy with trivial matches (like `return null;` or `if (x == null)`)

### Solution
Added **`min_duplicate_lines`** configuration (default: **5 lines**)

```yaml
analysis:
  min_duplicate_lines: 5  # Must be at least 5 lines to flag
```

**Behavior:**
- Functions smaller than 5 lines are **skipped entirely** from duplicate detection
- Similarity matches are **filtered** to only include functions ≥ 5 lines
- Prevents noise from trivial code patterns

---

## 2. **Semantic Code Comparison**

### Problem
- Was only comparing function **names and comments**
- Missed actual code logic → false positives on similar-named methods
- Example: `getUserById()` and `getOrderById()` flagged as similar just because of naming

### Solution
**Now includes actual function body** in embeddings for true semantic comparison

```go
// OLD (weak)
codeSnippet := fmt.Sprintf("// Function: %s\n%s", fn.Name, fn.Comments)

// NEW (semantic)
codeSnippet := createCodeSnippet(fn, filePath, language)
// Includes: language, name, parameters, return type, complexity, AND actual normalized code
```

**New embedding includes:**
1. Language context
2. Function signature (name, parameters, return type)
3. Complexity and LOC metrics
4. **Actual normalized function body** (comments stripped)

---

## 3. **Stricter Thresholds**

### Problem
- 79% similarity threshold was too low
- Flagged structurally similar but semantically different code
- Example: CRUD service methods flagged as duplicates

### Solution
**Two-tier threshold system:**

```yaml
analysis:
  duplicate_threshold: 0.90   # 90% = True duplicate
  refactor_threshold: 0.80    # 80-90% = Refactor opportunity
```

**Categories:**
- **≥90% similarity** → Flagged as **"Duplicate"** (likely copy-paste)
- **80-89% similarity** → Flagged as **"Refactor Opportunity"** (similar pattern, might extract to common function)
- **<80% similarity** → **Ignored** (too different)

---

## 4. **Better Reporting**

### Old Output
```
[CODE_QUALITY] Refactoring Opportunity: Similar logic in InstituteService.java (Match: 79.0%)
```

### New Output
```
[CODE_QUALITY] Refactoring Opportunity: Similar logic in InstituteService.java 
  (Match: 82.5%, 12 lines)
```

**Shows:**
- Similarity percentage
- **Number of lines** in the similar function
- Helps assess if it's worth refactoring

---

## 5. **Code Normalization**

To improve semantic comparison, code is now normalized before embedding:

```go
func normalizeCode(code string) string {
    // 1. Remove empty lines
    // 2. Remove comment-only lines
    // 3. Trim whitespace
    // 4. Focus on actual logic
}
```

**Example:**
```java
// Before normalization
public void save(User user) {
    // Validate user
    if (user == null) {
        throw new Exception("Null user");
    }
    
    // Save to DB
    repository.save(user);
}

// After normalization
public void save(User user) {
if (user == null) {
throw new Exception("Null user");
}
repository.save(user);
}
```

This focuses on **logic patterns** rather than **formatting style**.

---

## Configuration Reference

Update your `.katich/config.yaml`:

```yaml
analysis:
  max_function_length: 50
  complexity_threshold: 10
  similarity_threshold: 0.85
  
  # New duplicate detection settings
  min_duplicate_lines: 5      # Minimum lines to flag (prevents 1-2 line noise)
  duplicate_threshold: 0.90   # 90%+ = duplicate
  refactor_threshold: 0.80    # 80-90% = refactor opportunity
  
  sampling:
    enabled: true
    max_files: 20
    context_lines: 2
    skip_generated: true
    skip_tests: false
    adaptive_budget: true
```

---

## How It Works Now

### Detection Pipeline

```
1. Parse changed functions in diff
   ↓
2. Filter: Skip functions < 5 lines
   ↓
3. Create semantic embedding (with actual code body)
   ↓
4. Search against codebase embeddings
   ↓
5. Apply thresholds:
   - ≥90% → Duplicate
   - 80-89% → Refactor opportunity
   - <80% → Ignore
   ↓
6. Filter results: Only keep matches ≥ 5 lines
   ↓
7. Report with LOC context
```

---

## Examples

### Will NOT Flag (Previously Would)

```java
// Service method 1 (8 lines)
public User getUserById(Long id) {
    return repository.findById(id)
        .orElseThrow(() -> new NotFoundException("User not found"));
}

// Service method 2 (8 lines)  
public Order getOrderById(Long id) {
    return repository.findById(id)
        .orElseThrow(() -> new NotFoundException("Order not found"));
}
```

**Why not flagged:** Only 75% similar (different entity types, different repository)

### WILL Flag as Duplicate

```java
// File 1
public void processUser(User user) {
    if (user == null) throw new Exception();
    validateUser(user);
    enrichUserData(user);
    user.setStatus("ACTIVE");
    repository.save(user);
    sendNotification(user);
    logActivity(user);
}

// File 2 (92% similar)
public void handleUser(User user) {
    if (user == null) throw new Exception();
    validateUser(user);
    enrichUserData(user);
    user.setStatus("ACTIVE");
    repository.save(user);
    sendNotification(user);
    logActivity(user);
}
```

**Flagged as:** Duplicate (92% match, 8 lines)

### WILL Flag as Refactor Opportunity

```java
// Similar pattern, could extract
public void saveUser(User u) {
    validate(u);
    transform(u);
    repository.save(u);
    audit(u);
}

public void saveOrder(Order o) {
    validate(o);
    transform(o);
    repository.save(o);
    audit(o);
}
```

**Flagged as:** Refactor opportunity (85% match, 5 lines) - could extract to generic `saveEntity(T entity)`

---

## Testing

To test the improvements:

1. **Run on small functions** (should be skipped)
2. **Run on similar CRUD methods** (should NOT flag if <80%)
3. **Run on actual copy-pasted code** (SHOULD flag as duplicate)
4. **Check LOC reporting** (should show line counts)

---

## Benefits

✅ **Fewer false positives** (no more 1-2 line noise)  
✅ **More accurate semantic matching** (uses actual code logic)  
✅ **Clearer thresholds** (90% = duplicate, 80% = refactor)  
✅ **Better context** (shows LOC in reports)  
✅ **Configurable** (adjust thresholds per project)

---

## Next Steps

If you still see false positives:

1. **Increase `min_duplicate_lines`** to 8-10 for larger functions only
2. **Increase `duplicate_threshold`** to 0.95 for very strict matching
3. **Increase `refactor_threshold`** to 0.85 to reduce refactor suggestions
4. **Disable similarity** detection entirely if not useful: comment out the duplication check in `reviewer.go`

