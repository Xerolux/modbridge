# ModBridge Script Optimization Analysis

## Summary
The modbridge.sh installation script (1,342 lines) is a comprehensive installer with interactive menus, 
service management, and automatic updates. This analysis identifies 10 key optimization opportunities 
focusing on performance, code quality, and maintainability.

---

## TOP 10 OPTIMIZATION OPPORTUNITIES

### 1. **Redundant Port Checking Loop with Inefficient Logic**
**Location:** Lines 392-436 (`check_and_wait_for_ports` function)
**Severity:** HIGH (Performance)

**Problem:**
The port checking logic has a critical bug and inefficiency:
- Line 419 re-checks OLD blocked ports instead of the original PORTS variable
- Causes the loop to gradually check fewer ports and fail to detect when new ports become free
- Performs redundant lsof checks on the same ports repeatedly

**Current Code Pattern:**
```bash
BLOCKED_PORTS=()
for port in $PORTS; do
    if lsof -i ":$port" -sTCP:LISTEN -t >/dev/null 2>&1; then
        BLOCKED_PORTS+=("$port")
    fi
done
# ... later in loop ...
for port in "${BLOCKED_PORTS[@]}"; do  # WRONG: checks old array, not original
    if lsof -i ":$port" -sTCP:LISTEN -t >/dev/null 2>&1; then
        BLOCKED_PORTS+=("$port")  # Grows the array, never shrinks
    fi
done
```

**Impact:** 
- Ports may never be detected as free
- Installation can hang or fail unnecessarily
- Each iteration: ~12 lsof calls × 15 max iterations = 180+ system calls

**Recommendation:**
- Check each original port fresh each iteration, clear the array, rebuild it
- Use a single consolidated port check function
- Consider using `ss` (faster than lsof) or caching results

---

### 2. **Redundant pgrep Calls in Killing Processes (4 calls total)**
**Location:** Lines 343-389 (`kill_all_modbridge_processes` function)
**Severity:** MEDIUM (Performance/Redundancy)

**Problem:**
The function calls `pgrep` four separate times:
- Line 347: First check
- Line 363: Check after 0.5s sleep (in loop)
- Line 371: Check after full wait period
- Line 378: Final verification

Each call spawns a process and executes the search. With 15 iterations of the loop, 
this becomes 60+ process spawns just to manage termination.

**Current Pattern:**
```bash
PIDS=$(pgrep -x "modbridge" 2>/dev/null | grep -v "^$$\$" || true)
# ... kill ...
while [ $count -lt 10 ]; do
    sleep 0.5
    PIDS=$(pgrep -x "modbridge" 2>/dev/null || true)  # Redundant call
    if [ -z "$PIDS" ]; then
        break
    fi
    count=$((count + 1))
done
PIDS=$(pgrep -x "modbridge" 2>/dev/null || true)      # Redundant call
# ... check PIDS ...
PIDS=$(pgrep -x "modbridge" 2>/dev/null || true)      # Redundant call
```

**Impact:**
- Excessive system calls (60+ in worst case)
- Slower service stop/start/restart operations
- Unnecessary CPU usage during cleanup

**Recommendation:**
- Create a helper function: `get_modbridge_pids()` to centralize the logic
- Reuse results or cache between checks
- Refactor the loop to use a single source of truth for PID checking

---

### 3. **Duplicate Port Checking Code in Multiple Functions**
**Location:** Lines 401, 420 (check_and_wait_for_ports), 981, 1104 (status_service, health_check)
**Severity:** MEDIUM (Maintainability)

**Problem:**
The port checking logic is repeated in at least 3 different functions:
- `check_and_wait_for_ports`: Lines 401, 420
- `status_service`: Line 981
- `health_check`: Line 1104

Each has slightly different logic and error handling. Hardcoded port lists appear in multiple places:
- Line 393: `8080 5020 5021 5022 5023 5024 5025 5026 5027 5028 5029 5030`
- Line 980: `8080 5020 5021 5022 5023`
- Line 1102: `8080 5020 5021 5022 5023`

**Impact:**
- If port configuration changes, must update 3+ locations
- Inconsistent behavior between functions
- Code duplication violates DRY principle
- Each function uses different port lists!

**Recommendation:**
- Define port list as: `MODBRIDGE_PORTS="8080 5020 5021 5022 5023 5024 5025 5026 5027 5028 5029 5030"`
- Create a reusable function: `is_port_listening()` or `check_ports()`
- Use the central port list everywhere

---

### 4. **Repeated Binary Size Detection with Inconsistent Logic**
**Location:** Lines 277, 793-802, 1027-1036
**Severity:** MEDIUM (Maintainability)

**Problem:**
The logic to detect binary variant (full vs headless) appears in 3 different places with different approaches:

1. **show_installation_status (line 277):**
   ```bash
   SIZE=$(stat -c%s "$INSTALL_DIR/modbridge" 2>/dev/null || stat -f%z ... )
   if [ "$SIZE" -lt 8000000 ]; then
       echo "Variante: Headless"
   ```

2. **update_modbridge (lines 793-802):**
   ```bash
   if file "$INSTALL_DIR/modbridge" 2>/dev/null | grep -q "not stripped"; then
       CURRENT_VARIANT="full (mit WebUI)"
   else
       SIZE=$(stat -f%z "$INSTALL_DIR/modbridge" 2>/dev/null || stat -c%s ...)
       if [ "$SIZE" -lt 8000000 ]; then
           CURRENT_VARIANT="headless (ohne WebUI)"
   ```

3. **version_service (lines 1027-1036):** Same as #1

**Issues:**
- Different methods: file command vs size comparison (inconsistent)
- Hardcoded size threshold (8MB) appears in 3 places
- Fallback order differs between `stat -c` and `stat -f` across locations
- The `file | grep "not stripped"` check is unreliable

**Impact:**
- Variant detection may differ between functions
- Changing the threshold requires 3 edits
- Fragile reliance on binary size heuristic

**Recommendation:**
- Define constant: `FULL_BINARY_SIZE_THRESHOLD=8000000`
- Create function: `detect_binary_variant()` that returns "full" or "headless"
- Use consistent detection logic everywhere

---

### 5. **Logging Pattern Creates Pipeline with tee (Performance Impact)**
**Location:** Lines 43, 117, 121, 125 (all logging functions)
**Severity:** LOW (Performance/Style)

**Problem:**
Every log call uses a pipe to tee:
```bash
echo -e "${GREEN}[$(date +'%H:%M:%S')]${NC} $1" | tee -a "$LOG_FILE"
```

This:
- Creates a subshell for pipe
- Spawns tee process for every log call
- Script logs extensively (~100+ log calls in typical run)
- Results in 100+ additional process spawns

**Better Alternative:**
Use bash builtins:
```bash
{
    echo -e "${GREEN}[$(date +'%H:%M:%S')]${NC} $1"
} | tee -a "$LOG_FILE"
```

Or even better, batch writes or use redirects for high-frequency logging.

**Impact:**
- ~100+ unnecessary process spawns per script run
- Measurable slowdown on systems with slow process creation
- Every log adds ~1-2ms overhead

**Recommendation:**
- Use output redirection to append to file directly:
  ```bash
  {
      echo -e "${GREEN}[$(date +'%H:%M:%S')]${NC} $1"
  } >> "$LOG_FILE"
  echo -e "${GREEN}[$(date +'%H:%M:%S')]${NC} $1"
  ```
- Or redirect stdout/stderr at function level

---

### 6. **Multiple Version Fetches Without Caching**
**Location:** Lines 213-216, 224-225, 251-252, 637, 780-781, 853
**Severity:** MEDIUM (Performance)

**Problem:**
The `get_latest_version()` function calls `fetch_available_versions()` which makes an API call to GitHub:
```bash
get_latest_version() {
    local VERSIONS=$(fetch_available_versions)  # Fetches from GitHub API
    echo "$VERSIONS" | head -n 1
}
```

This is called multiple times in a single execution:
1. `install_modbridge`: Line 522 (get_current_version + get_latest_version)
2. `show_installation_status`: Lines 251-252 (both get_current_version and get_latest_version)
3. `update_modbridge`: Lines 780-781 (both get_current_version and get_latest_version)
4. `install_modbridge`: Line 637 (fetch_available_versions again)

Each API call to GitHub takes 200-500ms. In `show_installation_status`, it may be called even
when not needed.

**Impact:**
- Redundant API calls add 500ms-2s to script execution
- Unnecessary network traffic
- Script appears slow/hangs from user perspective
- Rate limiting risk on GitHub API

**Recommendation:**
- Cache version list in a temporary variable: `CACHED_VERSIONS=""`
- Check cache before fetching: `if [ -z "$CACHED_VERSIONS" ]; then fetch...; fi`
- Or add a 60-second TTL cache using file timestamps
- Lazy load: only fetch versions when actually needed

---

### 7. **Inefficient Version List Construction for Whiptail Dialogs**
**Location:** Lines 641-650 (install_modbridge), 857-866 (update_modbridge)
**Severity:** MEDIUM (Performance/Code Quality)

**Problem:**
Building the whiptail dialog options string dynamically:
```bash
local VERSION_LIST=""
local i=1
while IFS= read -r version; do
    if [ $i -eq 1 ]; then
        VERSION_LIST="$VERSION_LIST \"$version\" \"Version $i\" \"ON\""
    else
        VERSION_LIST="$VERSION_LIST \"$version\" \"Version $i\" \"OFF\""
    fi
    i=$((i+1))
done <<< "$VERSIONS"

eval "SELECTED_VERSION=\$(whiptail ... $VERSION_LIST ...)"
```

Issues:
1. **Duplicate code:** Exact same logic in install (641-650) AND update (857-866)
2. **String concatenation inefficiency:** Building string in loop with quotes
3. **eval danger:** Using `eval` with constructed strings is a security anti-pattern (though in this case relatively safe)
4. **Unbounded growth:** If many versions exist, string becomes very large

**Impact:**
- Duplicate code means fixes must be made in two places
- Slightly slower string concatenation
- eval could be source of injection if input validated incorrectly
- List becomes unwieldy with many versions

**Recommendation:**
- Extract to function: `build_version_selection_dialog()`
- Use array instead of string concatenation
- Use command substitution instead of eval
- Consider limiting shown versions (e.g., last 10) to keep dialog manageable

---

### 8. **Repeated stat Command with Multiple Fallbacks**
**Location:** Lines 277, 797, 1027 (stat -c vs stat -f)
**Severity:** LOW (Maintainability)

**Problem:**
Getting file size uses:
```bash
local SIZE=$(stat -c%s "$INSTALL_DIR/modbridge" 2>/dev/null || stat -f%z "$INSTALL_DIR/modbridge" 2>/dev/null)
```

This is repeated verbatim in 3 locations. The fallback order is also reversed:
- Lines 277, 1027: `stat -c` then `stat -f`
- Lines 797: `stat -f` then `stat -c` (DIFFERENT ORDER!)

**Issues:**
- Inconsistent fallback order could lead to different results
- Repeated code means changes must be made in 3 places
- Verbose and hard to maintain

**Impact:**
- Potential inconsistency if `-c` and `-f` produce different results
- Maintenance burden

**Recommendation:**
- Create function: `get_file_size() { local f="$1"; stat -c%s "$f" 2>/dev/null || stat -f%z "$f" 2>/dev/null; }`
- Use consistently throughout

---

### 9. **Unused MODBRIDGE_FORCE Pattern with Global State**
**Location:** Lines 1288-1298, 1302-1305, 520
**Severity:** LOW (Code Quality)

**Problem:**
Parsing command-line arguments to set `FORCE_INSTALL` variable at script level:
```bash
FORCE_INSTALL=0
SKIP_STATUS=0

for arg in "$@"; do
    if [ "$arg" = "--force" ]; then
        FORCE_INSTALL=1
    fi
    if [ "$arg" = "--skip-status" ]; then
        SKIP_STATUS=1
    fi
done
```

Then uses `export MODBRIDGE_FORCE=1` as a workaround:
```bash
if [ $FORCE_INSTALL -eq 1 ]; then
    export MODBRIDGE_FORCE=1
fi
```

The original flag is never actually passed to the function; instead a different variable is used.

**Issues:**
- Confusing dual-variable pattern
- Global state makes function behavior implicit
- Non-obvious that `install_modbridge()` checks `MODBRIDGE_FORCE` not `FORCE_INSTALL`

**Impact:**
- Code harder to understand and maintain
- Global state can cause subtle bugs

**Recommendation:**
- Pass flags as function arguments instead
- Or at least name variables consistently
- Remove export, use local scope

---

### 10. **Missing Error Handling and set -e Fragility**
**Location:** Line 16 (set -e), throughout script
**Severity:** MEDIUM (Reliability)

**Problem:**
Script starts with `set -e` (exit on any error), but then many commands explicitly suppress errors:
```bash
set -e  # Exit on error
...
PIDS=$(pgrep -x "modbridge" 2>/dev/null || true)
rm -f "$TEMP_SCRIPT"
systemctl stop "$SERVICE_NAME" 2>/dev/null || true
```

Using `|| true` and `2>/dev/null` to suppress errors works, but is inconsistent.

Issues:
1. **Contradiction:** `set -e` says "fail fast", but `|| true` says "ignore errors"
2. **Fragile:** If a developer forgets `|| true` on a command that's expected to fail, the script exits
3. **Unclear intent:** Is a failure here acceptable or a real error?
4. **Inconsistent:** Some error handling uses `if ...; then`, others use `|| true`

**Impact:**
- Script may exit unexpectedly in edge cases
- Error handling logic is scattered and inconsistent
- Difficult to understand which failures are acceptable

**Recommendation:**
- Either use `set -e` properly with explicit `|| true` for expected failures
- Or remove `set -e` and use explicit error checking
- Create helper: `exec_safe() { "$@" || true; }` for optional commands
- Better: use function patterns like:
  ```bash
  if ! cleanup_modbridge; then
      log_warn "Cleanup incomplete"
  fi
  ```

---

## SUMMARY TABLE

| # | Issue | Type | Severity | Frequency | Impact |
|---|-------|------|----------|-----------|--------|
| 1 | Redundant port check loop | Performance | HIGH | ~1/15s in loop | 180+ lsof calls |
| 2 | Multiple pgrep calls | Performance | MEDIUM | 4x per cleanup | 60+ process spawns |
| 3 | Duplicate port code | Maintainability | MEDIUM | 3 locations | Update lag |
| 4 | Binary variant detection | Maintainability | MEDIUM | 3 locations | Inconsistency |
| 5 | Logging pipe pattern | Performance | LOW | 100+ calls | 100+ subshells |
| 6 | No version caching | Performance | MEDIUM | ~3x per run | 500ms-2s added |
| 7 | Duplicate dialog code | Code Quality | MEDIUM | 2 locations | Maintainability |
| 8 | Repeated stat command | Maintainability | LOW | 3 locations | Inconsistency |
| 9 | Global state pattern | Code Quality | LOW | 1 location | Unclear behavior |
| 10 | set -e fragility | Reliability | MEDIUM | Throughout | Edge case failures |

---

## RECOMMENDED REFACTORING STRATEGY

### Phase 1 (High Impact)
1. Fix port checking loop (issue #1)
2. Add version caching (issue #6)
3. Consolidate pgrep calls (issue #2)

### Phase 2 (Maintainability)
1. Extract helper functions for duplicate code (#3, #4, #8)
2. Refactor dialog building (#7)
3. Improve error handling (#10)

### Phase 3 (Polish)
1. Remove global state (#9)
2. Optimize logging pipeline (#5)
3. Add comprehensive testing

---

## ESTIMATED PERFORMANCE GAINS

After implementing all optimizations:
- **Script execution time:** 5-15% faster (mainly from version caching & API calls)
- **System call overhead:** 40-50% reduction (fewer pgrep, lsof, process spawns)
- **Code maintainability:** Significantly improved through consolidation
- **Reliability:** Better error handling consistency

