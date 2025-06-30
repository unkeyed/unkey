# Hydra Test Plan

## Test Status Legend
- ❌ Not Started
- 🟡 In Progress  
- ✅ Complete

---

## Level 1: Basic Building Blocks

### Test 1.1: Hydra Instance Creation ❌
**Given:** A valid store configuration  
**When:** Creating a new Hydra instance  
**Then:** Instance is created successfully without panicking

### Test 1.2: Simple Workflow Registration ❌
**Given:** A workflow struct with Name() and Run() methods  
**When:** Registering the workflow with Hydra  
**Then:** Workflow is registered without error

### Test 1.3: Workflow Execution ID Generation ❌
**Given:** A registered workflow and valid request  
**When:** Starting a workflow execution  
**Then:** A unique execution ID is returned (starts with "wf_")

### Test 1.4: WorkflowContext Interface ❌
**Given:** A workflow execution in progress  
**When:** Accessing WorkflowContext methods  
**Then:** ExecutionID() and WorkflowName() return correct values

---

## Level 2: Single Step Workflows

### Test 2.1: Step with String Return ❌
**Given:** A workflow with one step that returns "hello"  
**When:** Workflow executes completely  
**Then:** Step result is "hello" and workflow completes successfully

### Test 2.2: Step with Typed Input ❌
**Given:** A workflow that takes `{Name: "John"}` as input  
**When:** Step accesses the input data  
**Then:** Step receives correctly typed input and can use it

### Test 2.3: Step Checkpointing ❌
**Given:** A workflow with one step that has already completed  
**When:** Running the same workflow execution again  
**Then:** Step is skipped (cached) and returns previous result

### Test 2.4: Step Failure Handling ❌
**Given:** A workflow with a step that returns an error  
**When:** Workflow executes  
**Then:** Workflow fails with the step's error message

---

## Level 3: Multi-Step Workflows

### Test 3.1: Sequential Steps ❌
**Given:** A workflow with step1 → step2  
**When:** Workflow executes completely  
**Then:** Both steps execute in order

### Test 3.2: Data Flow Between Steps ❌
**Given:** Step1 returns "data", Step2 uses that data  
**When:** Workflow executes  
**Then:** Step2 receives Step1's output correctly

### Test 3.3: Partial Execution Recovery ❌
**Given:** A workflow where step1 completes but step2 fails  
**When:** Retrying the workflow execution  
**Then:** Step1 is skipped (cached), Step2 runs again

---

## Level 4: Advanced Features

### Test 4.1: Sleep Functionality ❌
**Given:** A workflow that calls Sleep(1 second)  
**When:** Workflow executes  
**Then:** Workflow suspends, resumes after 1 second, and completes

### Test 4.2: Worker Concurrency ❌
**Given:** 2 workflow executions started simultaneously  
**When:** Worker processes both with concurrency=2  
**Then:** Both workflows complete successfully

### Test 4.3: Workflow Retry Logic ❌
**Given:** A workflow that fails on first attempt  
**When:** Workflow is retried  
**Then:** Workflow eventually succeeds on retry

### Test 4.4: Workflow Timeout ❌
**Given:** A workflow with a 1-second timeout that sleeps for 2 seconds  
**When:** Workflow executes  
**Then:** Workflow times out and fails

---

## Level 5: Real-World Scenarios

### Test 5.1: Multiple Workflow Types ❌
**Given:** EmailWorkflow and OrderWorkflow registered on same worker  
**When:** Starting executions of both types  
**Then:** Worker processes both types correctly

### Test 5.2: External Service Integration ❌
**Given:** A workflow that calls an external HTTP service (mocked)  
**When:** Service returns success/failure  
**Then:** Workflow handles both cases appropriately

### Test 5.3: Long-Running Workflow ❌
**Given:** A workflow with 10+ steps  
**When:** Workflow executes with some steps failing and retrying  
**Then:** Workflow eventually completes with proper checkpointing

---

## Test Infrastructure

### Test Utilities Needed:
- [ ] Test helper for creating Hydra instances
- [ ] Mock workflows for testing
- [ ] Assertion helpers for workflow state
- [ ] Time manipulation utilities for sleep tests

### Test Data:
- [ ] Simple request/response structs
- [ ] Mock external service responses
- [ ] Test workflow definitions