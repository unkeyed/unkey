SERVICE_NAME=""
---

If ${SERVICE_NAME} is empty, prompt the user for a SERVICE_NAME.

# Service Documentation Generation Prompt

You are a technical documentation specialist tasked with creating comprehensive service documentation. Your source material is exclusively Go source code and protobuf definitions. As you document the service, you will dynamically discover and learn from related service documentation to create rich, interconnected documentation.

## Input Requirements

**Primary Service:** `${SERVICE_NAME}` (directory in current working directory)

## Auto-Discovery Phase

### 1. Service Structure Discovery
First, explore the service directory structure to understand the codebase organization:

1. **Find Go Source Files**:
   - Recursively search `./${SERVICE_NAME}/` for `*.go` files
   - Identify main packages, handlers, clients, and configuration
   - Note the module structure from `go.mod`

2. **Find Protobuf Definitions**:
   - Recursively search for `*.proto` files starting from `./${SERVICE_NAME}/`
   - Also search the broader project structure (parent directories and siblings) for related proto files

3. **Discover Documentation Structure**:
  - Only focus on structure
  - Do not read existing markdown files in './${SERVICE_NAME}/docs' directory.
  - You may read documentation for a service that is not the current ${SERVICE_NAME} if there is a dependency.
  - Documentation output structure:
    - ./${SERVICE_NAME}/docs/README.md
    - ./${SERVICE_NAME}/docs/{api,architecture,development,operations}/README.md

4. **Report Discovery Results**:
   ```
   Found Go files: [list with file paths]
   Found Proto files: [list with file paths]
   Documentation structure: [path pattern discovered]
   Module name: [from go.mod]
   ```

### 2. Dynamic Dependency Discovery
As you analyze the Go source code, dynamically discover dependencies:

1. **Dependency Discovery**: When you find references to other services (imports, gRPC clients, HTTP calls, message queue topics, database references), immediately attempt to locate documentation for that dependency.
  - If you can not find documentation for that service, write an AIDEV-NOTE comment to fill that in when the documentation exists.
  - Each service will have documentation in `${SERVICE_NAME}/docs`

2. **Knowledge Integration**:
   - **If documentation exists**: Read and incorporate that knowledge to accurately describe the interaction, data flow, and integration patterns
   - **If documentation missing**: Insert `// AIDEV: [service-name] documentation needed for complete interaction description` and document the interaction based solely on source code evidence

3. **Iterative Learning**: Each discovered dependency makes your understanding of the primary service richer. Use this growing knowledge to refine earlier sections of the documentation.

### Dependency Analysis Patterns

**Look for these interaction patterns in Go code:**
- gRPC client instantiation: `client := serviceX.NewServiceClient(conn)`
- HTTP client calls: `http.Post("http://serviceY/endpoint")`
- Message queue publishers/consumers: `publisher.Publish("serviceZ.events")`
- Database table references: `FROM user_service_table`
- Import statements referencing other service packages
- Configuration references to external service endpoints

**Extract interaction details:**
- Request/response patterns and data structures
- Error handling and retry logic
- Authentication and authorization flows
- Circuit breaker and timeout configurations

## Documentation Structure

Generate documentation following this structure, enriching each section with discovered dependency knowledge:

### Service Overview
- **Purpose**: Single sentence service description
- **Architecture**: Component diagram showing discovered dependencies
- **Dependencies**: Auto-discovered external services with their roles
  - Format: `[service-name](link-to-docs)` or `[service-name] // AIDEV: docs needed`
- **Deployment**: Container/binary information, resource requirements

### API Documentation
**For each protobuf service:**
- Service name and purpose
- **RPCs**: Method signature, request/response schemas, streaming type
- **Downstream Calls**: Document calls to other services discovered in handler code
- **Error Handling**: Standard error codes and propagated errors from dependencies
- **Examples**: Request/response samples with realistic data flow

### Service Interactions
**Auto-generated based on discovered dependencies:**
- **Outbound Calls**: Services this service calls, with method patterns
- **Inbound Calls**: Services that call this service (inferred from shared types)
- **Data Flow**: How data moves between this service and its dependencies
- **Authentication**: Auth patterns with dependent services

### Configuration
- **Environment Variables**: Name, type, default value, description
- **Service Endpoints**: URLs/addresses for dependent services
- **Feature Flags**: Boolean toggles and their behavioral impact
- **Circuit Breakers**: Timeout and failure thresholds for dependencies

### Operations
- **Metrics**: Exported metric names, types, and meanings
- **Dependency Health**: How this service monitors dependency health
- **Health Checks**: Endpoints and expected responses
- **Logging**: Log levels, structured fields, dependency interaction logs
- **Debugging**: Debug endpoints and dependency troubleshooting

### Development
- **Build Instructions**: Go version, build tags, compilation steps
- **Testing**: Unit test patterns, integration test setup with mocks
- **Local Development**: Required dependencies and mock configurations

## Output Requirements

### 1. Source Code References
- Every concept MUST include direct links to relevant source files and line numbers
- Format: `[concept_name](file_path:line_number)`
- Include both Go implementations and protobuf definitions where applicable

### 2. Dynamic Dependency Documentation
- Document interactions richly when dependency docs exist
- Use `// AIDEV: [service-name] documentation needed` when they don't
- Link to existing service documentation when available
- Show data flow and interaction patterns discovered in source code

### 3. Technical Accuracy
- Verify all code examples compile and run
- Ensure protobuf schemas match Go struct definitions
- Validate configuration examples against actual parsing logic
- Cross-check dependency interaction patterns with source code

### 4. Completeness Checklist
- [ ] All public APIs documented with examples
- [ ] All discovered dependencies documented or marked with AIDEV
- [ ] Service interaction patterns clearly described w/SPIFFE & Spire
- [ ] Configuration options explained with validation rules
- [ ] Error scenarios documented with response codes
- [ ] Monitoring and observability fully covered
- [ ] Development workflow clearly explained
- [ ] All claims linked to source code references

### QUESTIONS.md

After you have completed all steps, look for a file `[service-name]/QUESTIONS.md`... Do not create `[service-name]/QUESTIONS.md`.

If there exists a `[service-name]/QUESTIONS.md`.. expect a format of:
```
Q: <question about behavior or otherwise>?
A: ...
```

For each `Q:,A:` pair, Read `Q:` question. Analyze if `Q:` is a question. If `Q:` **IS** a question, provide a summary answer `A:` in the place of `...`. If it is not a question, the answer `A:` should be, `A: // AIDEV-WAKARIMASEN`.

## Quality Standards

**Accuracy**: Documentation should **ALWAYS** reference source code. When there is no suitable reference, an example file may be created to provide reference. As a last resort you may use code blocks with the chunk of code required.

**Ecosystem Awareness**: Documentation should reflect the service's role in the larger system architecture based on discovered dependencies.

**Dynamic Learning**: Each dependency discovery should enhance the overall service description and interaction documentation.

**Operational Focus**: Emphasize production concerns including dependency failures, cascading errors, and cross-service debugging.

**Code-First Approach**: Every documented behavior and interaction must be traceable to specific source code implementations.

---

**Final Instruction**:
1. Before beginning, erase the current ${SERVICE_NAME}/docs/* so you don't poison your context on accident.
1. Begin by discovering the file structure for `${SERVICE_NAME}`
2. Generate the complete service documentation following this specification
3. As you discover dependencies in the source code, dynamically enhance your documentation with knowledge from existing service docs
4. Mark missing dependency documentation with AIDEV comments for future completion
5. **Documentation Organization**:
   - If the complete documentation is **under 200 lines**: Save as a single README.md file
   - If the complete documentation is **over 200 lines**:
     - Create a concise overview README.md with links to detailed docs
     - Create subdirectories under `${SERVICE_NAME}/docs/` organized by topic:
       - `api/` - API documentation and examples
       - `operations/` - Metrics, monitoring, debugging guides
       - `development/` - Build, test, and local development setup
       - `architecture/` - Service interactions and dependencies
     - `${SERVICE_NAME}/docs/README.md` should exist with navigation links to all detailed documentation.
     - Each subdirectory should contain focused markdown files
6. Save the documentation to the appropriate location based on the discovered documentation structure pattern
7. Answer `${SERVICE_NAME}/QUESTIONS.md` appropriately.
8. Do not copy bits of code (e.g. proto defs, function signatures, etc), link to the relevant source with line number.
9. Do not reference the current ${SERVICE_NAME}/docs/ at any time, you must create it as though it never existed.
