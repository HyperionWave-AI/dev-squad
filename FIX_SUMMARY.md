# Fix Summary: .env.hyper Variable Loading in dev-hot.sh

**Date:** 2025-10-12
**Agent:** SRE
**Task ID:** 7c436e23-1223-4d38-8fd9-2c1fb6c60472

## Problem

The `dev-hot.sh` script was failing to load environment variables from `.env.hyper` due to improper handling of quoted values:

```bash
# OLD (BROKEN) - Line 85 & 88:
export $(grep -v '^#' .env.hyper | xargs)
```

**Error:**
- `xargs: unterminated quote` when processing `MONGODB_URI` with quoted values
- Variables not properly exported to Air/Go subprocess
- Backend connected to wrong database: `coordinator_db_max` (system env) instead of `hyper_coordinator_db_dev_squad` (.env.hyper)

## Solution Implemented

Replaced the problematic `xargs` approach with proper shell sourcing:

```bash
# NEW (FIXED) - Lines 85 & 88:
set -a; source .env.hyper; set +a
```

**Why this works:**
- `set -a` enables automatic export mode (all variables are automatically exported)
- `source .env.hyper` reads and executes the file in current shell
- Properly handles quoted values, special characters, and complex strings
- Variables are exported to all child processes (Air → Go backend)
- `set +a` disables auto-export mode after sourcing

## Changes Made

**File:** `/Users/maxmednikov/MaxSpace/dev-squad/scripts/dev-hot.sh`

**Lines changed:**
- Line 85: `.env.hyper` loading
- Line 88: `.env` fallback loading

## Testing & Verification

### Test Command:
```bash
./scripts/dev-hot.sh
```

### Expected Output (✅ VERIFIED):
```
[Backend] INFO  coordinator/main.go:138  Connecting to MongoDB Atlas  {"database": "hyper_coordinator_db_dev_squad"}
[Backend] INFO  coordinator/main.go:179  Qdrant client initialized  {"knowledgeCollection": "hyper_dev_squad_knowledge"}
```

### Results:
✅ **Database name correct:** `hyper_coordinator_db_dev_squad` (from .env.hyper)
✅ **Knowledge collection correct:** `hyper_dev_squad_knowledge` (from .env.hyper)
✅ **No xargs errors:** Quoted MongoDB URI loaded successfully
✅ **Variables exported to subprocess:** Air/Go backend receives correct environment

## TODO Status

### TODO 1: ✅ COMPLETED
**ID:** 06d323ef-8238-48bf-a3a3-48a85345a117
**Task:** Replace xargs with proper sourcing
**Status:** Completed
**Implementation:** Lines 85 and 88 now use `set -a; source .env.hyper; set +a`

### TODO 2: ✅ COMPLETED
**ID:** d5fcc518-b8f1-498c-b0d9-87dbe49340e2
**Task:** Test backend loads correct DB name
**Status:** Completed
**Verification:** Backend logs confirm database: `hyper_coordinator_db_dev_squad`

## Impact

**Before Fix:**
- ❌ `xargs: unterminated quote` error
- ❌ Backend used wrong database (`coordinator_db_max`)
- ❌ Environment variables not reaching subprocess

**After Fix:**
- ✅ Clean startup, no errors
- ✅ Backend uses correct database (`hyper_coordinator_db_dev_squad`)
- ✅ All environment variables properly loaded and exported

## Technical Details

### The Problem with `xargs`:
The original pattern `export $(grep -v '^#' .env.hyper | xargs)` fails because:
1. `xargs` concatenates all input into space-separated arguments
2. Shell word-splitting treats quotes inconsistently
3. Complex strings like `MONGODB_URI="mongodb+srv://...?retryWrites=true&w=majority"` break xargs parsing
4. Variables may not be properly exported to subprocesses

### Why `set -a; source; set +a` is Better:
1. **Proper sourcing:** Executes .env file in current shell context
2. **Auto-export:** `set -a` ensures all variables are automatically exported
3. **Quote handling:** Shell properly interprets quoted values
4. **Subprocess inheritance:** Exported variables are available to all child processes
5. **Standard approach:** This is the recommended pattern for .env files in bash

## Future Recommendations

1. **Consistency:** Use this pattern everywhere .env files are loaded
2. **Validation:** Consider adding environment variable validation after sourcing
3. **Documentation:** Update README with correct .env sourcing pattern
4. **Error handling:** Add checks to ensure required variables are set after sourcing

## Related Files

- `/Users/maxmednikov/MaxSpace/dev-squad/scripts/dev-hot.sh` - Fixed script
- `/Users/maxmednikov/MaxSpace/dev-squad/.env.hyper` - Environment configuration
- `/Users/maxmednikov/MaxSpace/dev-squad/coordinator/main.go` - Backend that consumes variables

## References

- Bash manual: `help set` (for `set -a` documentation)
- POSIX standard: Shell automatic export behavior
- Best practices: [The Twelve-Factor App - Config](https://12factor.net/config)
