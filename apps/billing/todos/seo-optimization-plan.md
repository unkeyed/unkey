# SEO Optimization Plan for Glossary Content

## Overview

This plan outlines the steps to enhance our glossary content generation with SEO optimization capabilities. We'll move the keyword research functionality to the research folder, implement filesystem-based storage instead of database operations, and create an evaluation library for assessing content quality from an SEO perspective.

## Tasks

### 1. Move Keyword Research to Research Folder

- [ ] Create a copy of the existing `keyword-research.ts` in the research folder
- [ ] Modify the copy to use filesystem storage instead of database operations
- [ ] Add a note about potentially moving back to database storage in the future
- [ ] Ensure the task returns the same data structure as the original

### 2. Create Evaluation Library

- [x] Create a new directory structure at `src/lib/eval`
- [x] Implement a base `Metric` interface
- [x] Create the following SEO-specific metrics:
  - [x] `KeywordUsageMetric`: Evaluates how well keywords are incorporated in the content
  - [x] `ReadabilityMetric`: Assesses content readability (sentence length, complexity)
  - [x] `ContentStructureMetric`: Evaluates heading structure, paragraph length, etc.
- [x] Implement an `evaluate` function that can run multiple metrics
- [x] Create a reporting utility for displaying evaluation results

### 3. Implement SEO Evaluation Task

- [x] Create a new task in the research folder for SEO evaluation
- [x] Integrate with the evaluation library
- [x] Ensure the task can access keyword research results
- [x] Implement logic to evaluate content against keywords
- [x] Return evaluation results and improvement suggestions

### 4. Create Main Orchestration Task

- [x] Create a task that orchestrates the entire workflow:
  1. Technical research
  2. Content generation
  3. SEO evaluation
- [x] Ensure proper error handling and metadata tracking
- [x] Return comprehensive results including content and SEO metrics

## Implementation Details

### Filesystem Storage Approach

We'll use JSON files stored in a local directory structure:

```
.cache/keyword-research/
  └── {term}.json
```

This will allow us to persist data between runs while avoiding database operations during development.

### Evaluation Library Structure

```
src/lib/eval/
  ├── index.ts           # Main exports
  ├── types.ts           # Type definitions
  ├── metrics/           # Individual metrics
  │   ├── base.ts        # Base metric interface
  │   ├── keyword.ts     # Keyword usage metric
  │   ├── readability.ts # Readability metric
  │   └── structure.ts   # Content structure metric
  ├── evaluate.ts        # Main evaluation function
  └── reporting.ts       # Reporting utilities
```

### SEO Evaluation Metrics

1. **Keyword Usage Metric**
   - Primary keyword in title, introduction, and conclusion
   - Keyword density (not too high, not too low)
   - Secondary keywords distribution
   - Semantic variations of keywords

2. **Readability Metric**
   - Sentence length
   - Paragraph length
   - Use of transition words
   - Active vs. passive voice
   - Reading level assessment

3. **Content Structure Metric**
   - Proper heading hierarchy
   - Use of lists and bullet points
   - Content length adequacy
   - Image alt text (if applicable)
   - Internal linking opportunities

### Main Orchestration Task

The main task will:

1. Run technical research to gather domain knowledge
2. Perform keyword research to identify SEO opportunities
3. Generate content based on the research
4. Evaluate the content from an SEO perspective
5. Return the content with SEO metrics and improvement suggestions

This approach allows us to:

- Keep the existing content generation workflow intact
- Add SEO capabilities as a parallel process
- Provide comprehensive SEO feedback without modifying the core content generation

## Testing Instructions

To test the workflow:

1. **Setup**

   ```bash
   # Create necessary directories
   mkdir -p .cache/keyword-research
   ```

2. **Run the Trigger.dev CLI**

   ```bash
   npx trigger.dev@latest dev
   ```

3. **Test the Workflow**

   ```bash
   # Use the Trigger.dev dashboard to trigger the main task with a test term
   # Example: { "inputTerm": "api authentication" }
   ```

4. **Verify Results**
   - Check the Trigger.dev dashboard for task execution details
   - Examine the metadata for SEO metrics and evaluation results
   - Review the generated content with SEO improvements

5. **Debug**
   - Check the .cache directory for stored keyword research data
   - Review logs for any errors or warnings
   - Use the Trigger.dev dashboard to inspect individual task runs

## Next Steps

1. Implement the keyword research task with filesystem storage
2. Implement the evaluation library
3. Implement the SEO evaluation task
4. Implement the main orchestration task
5. Test the complete workflow

## Notes

- The original database-backed keyword research implementation will be preserved
- We may return to using the database in the future
- The evaluation library should be designed for reuse across different types of content
