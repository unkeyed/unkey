SYSTEM_NAME="Unkey Deploy"
SERVICES="assetmanagerd billaged builderd metald"

---

# System Documentation Generation Prompt

You are a technical documentation specialist tasked with creating unified high-level system documentation. Your source material is the individual service documentation files and the underlying Go source code and protobuf definitions. You will validate claims made in service documentation against actual source code and create comprehensive system-level architecture documentation.

## Input Requirements

**System:** `${SYSTEM_NAME}`
**Services to analyze:** `${SERVICES}`
**Do not analyze:** `*DOCUMENTATION_PROMPT.md`

## Auto-Discovery Phase

### 1. Service Discovery
First, analyze the specified services and their documentation:

1. **Target Services**:
   - Process only the services listed in `${SERVICES}`
   - Validate that each service directory exists

2. **Find Service Documentation**:
   - For each specified service, look for existing documentation:
     - `./[service-name]/README.md`
     - `./[service-name]/docs/`
   - Identify services with documentation vs. those marked with AIDEV comments

3. **Find Root Documentation Structure**:
   - Check for existing `./docs/` directory
   - Identify current system-level documentation patterns

4. **Report Discovery Results**:
   ```
   Target services: [specified service list]
   Services with documentation: [list]
   Services needing documentation: [list with AIDEV markers]
   Root docs structure: [existing patterns]
   ```

### 2. Documentation Validation and Cross-Reference

For each service with existing documentation:

1. **Claim Validation**: Read the service documentation and extract key claims about:
   - Service dependencies and interactions
   - API contracts and data flows
   - Configuration requirements
   - Operational characteristics

2. **Source Code Verification**:
   - Cross-reference claims against actual Go source code
   - Verify protobuf definitions match documented APIs
   - Validate dependency relationships exist in imports and client code
   - Check configuration claims against actual config structs

3. **Cross-Service Validation**:
   - When Service A claims to call Service B, verify Service B documentation acknowledges this relationship
   - Validate API contracts match between caller and callee
   - Check for orphaned dependencies (documented but not implemented)
   - Identify undocumented service interactions found in source code

4. **Inconsistency Reporting**:
   - Document discrepancies between service docs and source code
   - Note missing dependency documentation with AIDEV markers
   - Flag conflicting claims between service documentations

## System Documentation Structure

Generate unified system documentation with the following structure:

### System Overview
- **Purpose**: High-level system description and business capabilities
- **Architecture**: System-wide component diagram showing all services and their relationships
- **Service Inventory**: Complete list of services with their roles and status
  - Format: `[service-name](link-to-service-docs)` or `[service-name] // AIDEV: docs needed`
- **Technology Stack**: Common technologies, frameworks, and patterns across services

### Service Interaction Map
**Auto-generated based on validated cross-service relationships:**
- **Data Flow Diagrams**: How data moves through the system
- **API Dependencies**: Which services call which other services
- **Authentication Flows**: System-wide auth patterns and service-to-service auth
- **Event/Message Flows**: Async communication patterns and message queues

### System-Wide Concerns
- **Configuration Management**: Common config patterns and shared environment variables
- **Monitoring and Observability**:
  - System-wide metrics and dashboards
  - Distributed tracing patterns
  - Log aggregation and correlation
- **Security**: Cross-service authentication, authorization, and security patterns
- **Data Consistency**: Transaction patterns, eventual consistency, and data synchronization

### Operational Runbooks
- **System Health**: How to assess overall system health
- **Incident Response**: Common failure patterns and troubleshooting workflows
- **Deployment Patterns**: Rolling updates, blue-green deployments, service dependencies
- **Scaling Considerations**: Bottlenecks, scaling patterns, and capacity planning

### Development Guidelines
- **Service Standards**: Common patterns all services should follow
- **API Design Guidelines**: Consistent API patterns and conventions
- **Testing Strategies**: Integration testing, contract testing, end-to-end testing
- **Development Workflow**: How to develop, test, and deploy changes across services

## Validation Requirements

### 1. Source Code Cross-Reference
- Every system-level claim MUST be validated against actual service source code
- Document validation status: `✓ Verified` or `⚠ Needs Validation`
- Include source references: `[claim](service/file:line_number)`

### 2. Documentation Consistency
- Identify and document inconsistencies between service docs
- Validate that bi-directional service relationships are documented on both sides
- Flag services that reference undocumented dependencies

### 3. Completeness Assessment
- Calculate system documentation coverage: `X of Y services documented`
- List services needing documentation with priority assessment
- Identify critical missing documentation affecting system understanding

### 4. AIDEV Marker Management
- Consolidate all AIDEV markers from service docs into system-level tracking
- Prioritize missing documentation based on service criticality and interconnectedness
- Create action items for completing system documentation

## Output Requirements

### 1. Validation Report
Generate a validation summary showing:
- Documentation coverage percentage
- List of validated vs. unvalidated claims
- Critical inconsistencies requiring attention
- Prioritized AIDEV items for system completion

### 2. Documentation Organization
- **If system documentation is under 300 lines**: Single `./docs/README.md`
- **If system documentation is over 300 lines**:
  - Main `./docs/README.md` with system overview and navigation
  - `./docs/architecture/` - Service interactions, data flows, system diagrams
  - `./docs/operations/` - Monitoring, incident response, deployment guides
  - `./docs/development/` - Standards, guidelines, testing strategies
  - `./docs/services/` - Service inventory and cross-reference matrix

### 3. Cross-Reference Matrix
Create a service interaction matrix showing:
- Which services call which other services
- Documentation status for each relationship
- API contract validation status

## Quality Standards

**System-Level Perspective**: Documentation should provide a complete mental model of how the entire system operates, not just individual services.

**Validation-First**: Every claim must be traceable to source code or marked as needing validation.

**Operational Focus**: Emphasize system-wide operational concerns, failure modes, and debugging across service boundaries.

**Consistency Enforcement**: Identify and resolve inconsistencies between service documentation and actual implementations.

---

**Final Instruction**:
1. Process only the services specified in `${SERVICES}`
2. Validate service documentation claims against source code
3. Generate comprehensive system documentation following this specification
4. Create a validation report highlighting inconsistencies and missing documentation in `./VALIDATION_REPORT.md`.
5. Organize documentation based on size and complexity
6. Save all documentation to `./docs/` with appropriate structure
