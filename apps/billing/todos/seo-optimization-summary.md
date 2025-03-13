# SEO Optimization for Glossary Content - Summary

## Overview

We've developed a comprehensive plan to enhance our glossary content generation with SEO optimization capabilities. This plan includes moving the keyword research functionality to the research folder, implementing filesystem-based storage, creating an evaluation library, and building a main orchestration task that coordinates the entire workflow.

## Completed Planning

We've completed the planning phase for all major components:

1. **Keyword Research Task**: Designed a filesystem-based version of the existing keyword research task
2. **Evaluation Library**: Designed a flexible framework for assessing content quality using various metrics
3. **SEO Evaluation Task**: Designed a task that evaluates content from an SEO perspective
4. **Main Orchestration Task**: Designed a task that coordinates the entire workflow

## Implementation Plan

The implementation will proceed in the following order:

1. **Keyword Research Task**
   - Create a copy of the existing task in the research/seo folder
   - Modify it to use filesystem storage instead of database operations
   - Ensure it returns the same data structure as the original

2. **Evaluation Library**
   - Implement the core components (types, base metric, evaluate function, reporting)
   - Implement the SEO-specific metrics (keyword usage, readability, content structure)
   - Test the library with sample content

3. **SEO Evaluation Task**
   - Implement the task in the research/seo folder
   - Integrate it with the evaluation library
   - Test it with sample content and keywords

4. **Main Orchestration Task**
   - Implement the task that coordinates the entire workflow
   - Ensure proper error handling and metadata tracking
   - Test the complete workflow

5. **Add Tracing**
   - Implement custom traces for better observability
   - Add spans for key operations in each task
   - Enhance debugging capabilities

## Key Features

### Keyword Research

- Extracts keywords from search results, titles, and headers
- Stores results in the filesystem for development
- Provides a list of relevant keywords for content optimization

### Evaluation Library

- Provides a flexible framework for assessing content quality
- Supports multiple metrics for comprehensive evaluation
- Generates detailed reports with improvement suggestions

### SEO Evaluation

- Evaluates keyword usage, readability, and content structure
- Provides an overall SEO score and specific improvement suggestions
- Generates a formatted report for easy interpretation

### Main Orchestration

- Coordinates technical research, content generation, and SEO evaluation
- Provides comprehensive results including content and SEO metrics
- Tracks progress and handles errors appropriately

### Tracing

- Uses OpenTelemetry for distributed tracing
- Provides visibility into task execution flow
- Helps identify performance bottlenecks and errors
- Includes custom spans for key operations

## Testing Strategy

1. **Unit Testing**: Test each component in isolation
2. **Integration Testing**: Test the workflow with mock data
3. **End-to-End Testing**: Test the complete workflow with real terms

## Next Steps

1. Set up the necessary directory structure:
   ```bash
   mkdir -p apps/billing/src/trigger/glossary/research/seo
   mkdir -p apps/billing/src/lib/eval/metrics
   mkdir -p apps/billing/.cache/keyword-research
   ```

2. Implement the keyword research task:
   ```bash
   # Create the file
   touch apps/billing/src/trigger/glossary/research/seo/keyword-research.ts
   ```

3. Implement the evaluation library:
   ```bash
   # Create the files
   touch apps/billing/src/lib/eval/types.ts
   touch apps/billing/src/lib/eval/metrics/base.ts
   touch apps/billing/src/lib/eval/metrics/keyword.ts
   touch apps/billing/src/lib/eval/metrics/readability.ts
   touch apps/billing/src/lib/eval/metrics/structure.ts
   touch apps/billing/src/lib/eval/evaluate.ts
   touch apps/billing/src/lib/eval/reporting.ts
   touch apps/billing/src/lib/eval/index.ts
   ```

4. Implement the SEO evaluation task:
   ```bash
   # Create the file
   touch apps/billing/src/trigger/glossary/research/seo/evaluate-content.ts
   ```

5. Implement the main orchestration task:
   ```bash
   # Create the file
   touch apps/billing/src/trigger/glossary/generate-with-seo.ts
   ```

6. Test the workflow:
   ```bash
   # Run the Trigger.dev CLI
   npx trigger.dev@latest dev
   ```

## Documentation

Detailed plans for each component are available in the following files:

- [SEO Optimization Plan](/apps/billing/todos/seo-optimization-plan.md)
- [Evaluation Library Plan](/apps/billing/todos/evaluation-library-plan.md)
- [SEO Evaluation Task Plan](/apps/billing/todos/seo-evaluation-task-plan.md)
- [Main Orchestration Task Plan](/apps/billing/todos/orchestration-task-plan.md)

## Future Enhancements

1. **Content Optimization**: Automatically apply SEO suggestions to improve content
2. **Competitive Analysis**: Compare content against top-ranking pages
3. **Historical Tracking**: Track SEO improvements over time
4. **Custom Metrics**: Allow adding custom evaluation metrics
5. **Database Integration**: Move back to database storage for production

## Tracing Implementation

To implement tracing in our tasks, we'll use Trigger.dev's built-in tracing capabilities:

```typescript
import { logger, task } from "@trigger.dev/sdk/v3";

// Example of adding tracing to the keyword research task
export const keywordResearchTask = task({
  id: "seo_keyword_research",
  retry: {
    maxAttempts: 3,
  },
  run: async ({ term, onCacheHit = "stale" as CacheStrategy }) => {
    // Check cache with tracing
    const cachedData = await logger.trace("check-keyword-cache", async (span) => {
      span.setAttribute("term", term);
      span.setAttribute("cacheStrategy", onCacheHit);
      
      const data = readKeywordsFromCache(term);
      span.setAttribute("cacheHit", !!data);
      return data;
    });
    
    if (cachedData && onCacheHit === "stale") {
      return {
        message: `Found existing keywords for ${term}`,
        term,
        keywords: cachedData.keywords,
      };
    }
    
    // Trace search query generation
    const entryWithSearchQuery = await logger.trace("generate-search-query", async (span) => {
      span.setAttribute("term", term);
      const result = await getOrCreateSearchQuery({ term, onCacheHit });
      span.setAttribute("success", !!result?.searchQuery);
      return result;
    });
    
    // Continue with other operations, adding traces as needed...
  }
});
```

### Key Tracing Points

We'll add tracing to the following key operations:

1. **Keyword Research**:
   - Cache operations
   - Search query generation
   - API calls to external services

2. **Evaluation Library**:
   - Metric evaluations
   - Performance-intensive operations

3. **SEO Evaluation**:
   - Overall evaluation process
   - Individual metric calculations

4. **Main Orchestration**:
   - Task transitions
   - Error handling
   - Performance bottlenecks

### Benefits

- **Improved Debugging**: Easily identify where issues occur
- **Performance Insights**: Find bottlenecks in the workflow
- **Better Observability**: Understand the execution flow
- **Error Tracking**: Trace errors to their source 