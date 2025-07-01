# Go Documentation Generation Prompt

Use this prompt when asking an LLM to generate documentation following our guidelines:

---

**TASK**: Generate comprehensive Go documentation for the provided code following our team's documentation standards.

**CRITICAL REQUIREMENTS**:
1. **Read the full GO_DOCUMENTATION_GUIDELINES.md file first** - This contains our complete standards
2. **Every exported item MUST be documented** - No exceptions
3. **Be comprehensive and verbose** - Prefer thorough explanations over terse summaries
4. **Add substantial value** - Documentation must teach beyond what's obvious from signatures
5. **Write in full sentences, not bullet points** - Documentation should read like prose
6. **Internal code should explain "why", not "how"** - Focus on reasoning and trade-offs
7. **Use context-specific approaches** - Different documentation styles for packages, functions, types, etc.

**SPECIFIC INSTRUCTIONS**:

**For Package Documentation**:
- **MUST create a dedicated `doc.go` file** - never put package docs in other files
- Content: Only package documentation and package declaration, no other code
- Start with "Package [name] [verb]..."
- Use `#` headers to organize sections (Key Types, Usage, Error Handling, etc.)
- Explain purpose, scope, and architectural reasoning
- Include complete, runnable examples
- Use extensive cross-references with `[TypeName]` and `[FunctionName]` format
- Reference related packages and external resources

**For Public Functions/Methods**:
Write natural, comprehensive documentation that explains what matters for each specific function. Start with what the function does, then include whatever information is actually relevant and useful for callers. This might include parameter explanations, return values, error conditions, performance notes, concurrency considerations, or usage examples - but only include what actually adds value for that particular function. Don't force every function into the same template. Write as flowing prose that reads naturally.

**For Internal Functions**:
- Explain WHY this implementation was chosen
- Document architectural decisions and trade-offs
- Reference alternatives considered and why they were rejected
- Focus on reasoning, not mechanics

**For Types**:
- Explain what the type represents in the system
- Document important invariants and constraints
- Explain lifecycle and usage patterns
- For interfaces, explain design decisions

**QUALITY STANDARDS**:
- Match Go standard library documentation quality
- Be clear, concise, and practical
- Start with the item name, use present tense
- Provide examples for complex usage
- Explain error conditions thoroughly

**EXAMPLES - INADEQUATE vs COMPREHENSIVE**:

❌ **INADEQUATE** (adds little value):
```go
// GetJob retrieves a job by its unique ID.
// Returns the job details or an error if the job is not found.
func (c *Client) GetJob(jobID string) (*Job, error)
```

✅ **COMPREHENSIVE** (substantial value):
```go
// GetJob retrieves a complete job record by its unique identifier from persistent storage.
//
// This method performs a direct database lookup and returns the full job record including
// all metadata, execution status, retry history, timing information, and error details.
// The returned job reflects the current state at query time but may become stale
// immediately in a concurrent environment where workers are actively processing jobs.
//
// Use this method for:
//   - Administrative monitoring and job inspection
//   - Debugging failed or stuck jobs
//   - Building dashboards and reporting tools
//   - Audit trails and compliance logging
//
// For job processing workflows, workers should use the internal job claiming mechanism
// rather than this method, as it doesn't provide the necessary coordination primitives.
//
// Parameters:
//   - jobID: Must be a valid UUID string as returned by Enqueue(). Invalid UUIDs
//     or malformed strings will result in sql.ErrNoRows.
//
// Returns:
//   - (*Job, nil): Successfully retrieved job with complete metadata
//   - (nil, sql.ErrNoRows): No job exists with the given ID
//   - (nil, error): Database connection issues, query timeouts, or storage failures
//
// Performance: Single indexed database lookup, typically <1ms. Does not scale with
// queue size or job count.
//
// Concurrency: Safe for concurrent use but provides no consistency guarantees.
// The job state may change between when this method returns and when the caller
// acts on the information.
func (c *Client) GetJob(jobID string) (*Job, error)
```

**VALIDATION CHECKLIST** (verify before submitting):
- [ ] **Package has a dedicated `doc.go` file** with comprehensive documentation
- [ ] Every exported item is documented
- [ ] Package documentation in `doc.go` includes purpose, examples, and cross-references
- [ ] Public functions explain contract and usage clearly
- [ ] Internal code explains "why" decisions were made
- [ ] Error conditions are thoroughly documented
- [ ] Documentation follows Go conventions (starts with name, present tense)
- [ ] Complex logic includes reasoning for chosen approach

**PROVIDE THE CODE TO DOCUMENT HERE**:
[Insert Go code that needs documentation]

---

**Expected Output**: The same Go code with comprehensive documentation added, following all guidelines above.