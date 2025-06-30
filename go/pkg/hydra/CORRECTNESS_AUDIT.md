# Hydra Workflow Engine Correctness Audit

## 🚨 Critical Issues Found

### **Exactly-Once Execution BROKEN**
1. **Step ID mismatch**: Creation uses `"-"` but completion uses `"_"` → steps never marked complete
2. **Non-atomic step execution**: Step runs, then state update → crash = duplicate execution  
3. **Marshalling failure after step success**: Step side effects happen, but marked as failed → re-runs

### **Worker Recovery BROKEN** 
1. **No heartbeats**: Workers don't send heartbeats → dead worker detection impossible
2. **Incomplete lease cleanup**: Only resets workflow status, not step state
3. **No step-level recovery**: Orphaned steps never get picked up

### **Infinite Loops POSSIBLE**
1. **No retry limit enforcement**: Steps retry forever (MaxAttempts ignored)
2. **Backoff overflow**: `1 << attempts` can overflow → immediate retries

### **Race Conditions EVERYWHERE**
1. **Multiple workers on same step**: No step-level locking
2. **Status update races**: Workflow status updated outside lease transaction
3. **Missing DB constraints**: No unique constraints on step IDs

## 🎯 What We Need to Fix

### **Priority 1: Exactly-Once Execution**
- [ ] Fix step ID format consistency 
- [ ] Make step execution + state update atomic
- [ ] Handle marshalling failures correctly

### **Priority 2: Worker Recovery**
- [ ] Implement actual heartbeat sending
- [ ] Add step-level lease/recovery logic
- [ ] Fix orphaned step detection

### **Priority 3: Retry Logic**
- [ ] Enforce step retry limits
- [ ] Fix backoff calculation overflow
- [ ] Add maximum retry bounds

### **Priority 4: Concurrency Control**  
- [ ] Add step-level locking
- [ ] Fix workflow state transition races
- [ ] Add database unique constraints

## 🧪 Tests We Need

### **Correctness Property Tests**
```
✅ Step executes exactly once (even with worker crashes)
✅ Workflow terminates eventually (success or max retries)  
✅ Worker crash recovery works correctly
✅ No race conditions between workers
✅ Database state is always consistent
```

### **Failure Scenario Tests**
```
🔍 Worker crashes mid-step execution
🔍 Database connection lost during state update
🔍 Multiple workers process same workflow
🔍 Step succeeds but marshalling fails
🔍 Heartbeat system under network partitions
```

## 🚧 Current State: NOT PRODUCTION READY

The workflow engine has fundamental correctness issues that **will** cause:
- Duplicate side effects (emails sent twice, payments processed twice)
- Lost work (workflows stuck forever)
- Inconsistent state (steps show as failed but actually succeeded)

**Recommendation**: Fix Priority 1 issues before any production use.