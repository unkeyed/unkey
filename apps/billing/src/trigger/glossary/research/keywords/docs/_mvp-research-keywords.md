# Parent Task: Research Keywords MVP Specification

## Overview

Parent task to coordinate and combine results from multiple keyword research sources. This task orchestrates the parallel execution of different keyword research methods and unifies their results.

## Core Functionality

1. Execute three keyword research tasks in parallel:
   - Related Keywords (massiveonlinemarketing.nl)
   - Serper Search Results
   - Serper Autosuggest
2. Filter out keywords below 0.8 confidence score from Serper results
3. Enrich the two Serper results with keyword data from api.keywordseverywhere.com
4. Transform both RelatedKeywordSchema and EnrichedKeywordSchema into one unified schema
5. Deduplicate all results & return

## Input Schema

```typescript
interface ParentTaskPayload {
  inputTerm: string;
}
```

## Output Schema

```typescript
// We'll need to define this schema to combine both data structures
interface UnifiedKeywordSchema {
  keyword: string;
  volume: number;
  cpc: number;
  competition: number;
  confidence?: number; // From Serper results
  trends?: Array<{
    month: string;
    year: number;
    value: number;
  }>;
}

interface ParentTaskOutput {
  keywords: Array<UnifiedKeywordSchema>;
  metadata: {
    totalKeywords: number;
    sources: {
      relatedKeywords: number;
      serperSearch: number;
      serperAutosuggest: number;
    }
  }
}
```

## Implementation Steps

### 1. Task Setup

- Create parent task with ID "research_keywords"
- Prefix the file name with `_` to pin it to the top of the file list in the UI
- Import and validate all required subtasks

### 2. Parallel Execution & Sequential Enrichment

- Use batch.triggerByTaskAndWait for parallel execution of initial tasks
- After Serper tasks complete, run enrichment sequentially on their results
- Collect and validate results from all sources
- Handle potential failures gracefully

### 3. Data Unification

- Transform RelatedKeywordSchema results to UnifiedKeywordSchema
- Transform EnrichedKeywordSchema results to UnifiedKeywordSchema
- Combine all results into single array
- Deduplicate based on keyword text

### 4. Return Results

- Return unified keyword list
- Include basic metadata about sources and processing

## Error Handling

- Fail fast if any critical task fails
- Return partial results if non-critical tasks fail
- Log errors for debugging

## Guidelines

- Progress tracking via trigger's metadata (see trigger.mdc inside .cursor/rules)
- Use the `@trigger.dev/sdk/v3` for all trigger.dev related functionality
- Start with the test file and a basic implementation so that we can commit that
- Then we'll iterate on making the tests pass and improving the implementation

## Dependencies

- @trigger.dev/sdk/v3
- ./related-keywords
- ./serper-search
- ./serper-autosuggest
- ./enrich-keywords

## Next Steps

1. Create test file with basic happy path test
2. Implement minimal working version
== wait for approval & commit by user ==
3. Refine implementation
