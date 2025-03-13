# SEO Evaluation Task Plan

## Overview

The SEO evaluation task will assess content quality from an SEO perspective using multiple metrics. It will analyze how well keywords are incorporated, evaluate readability, and check content structure, providing a comprehensive evaluation with actionable improvement suggestions.

## Task Structure

```typescript
export const seoEvaluationTask = task({
  id: "seo_evaluation",
  retry: {
    maxAttempts: 2,
  },
  run: async ({ 
    content, 
    primaryKeyword, 
    secondaryKeywords 
  }: { 
    content: string; 
    primaryKeyword: string; 
    secondaryKeywords: string[] 
  }) => {
    // 1. Initialize metrics
    // 2. Run evaluations
    // 3. Generate report
    // 4. Return results
  }
});
```

## Implementation Steps

### 1. Initialize Metrics

```typescript
// Import metrics
import { KeywordUsageMetric, ReadabilityMetric, ContentStructureMetric } from "@/lib/eval/metrics";
import { evaluate } from "@/lib/eval";

// Initialize metrics
const metrics = [
  new KeywordUsageMetric(),
  new ReadabilityMetric(),
  new ContentStructureMetric()
];
```

### 2. Run Evaluations

```typescript
// Prepare input for evaluation
const input = {
  content,
  primaryKeyword,
  secondaryKeywords,
  context: {
    contentType: "glossary",
    targetAudience: "api developers"
  }
};

// Run evaluations
const evaluationResult = await evaluate(input, metrics);
```

### 3. Generate Report

```typescript
// Generate a formatted report
const report = generateReport(evaluationResult);

// Log the report to the console
console.info(report);
```

### 4. Return Results

```typescript
// Return the evaluation results
return {
  overallScore: evaluationResult.score,
  passed: evaluationResult.passed,
  metrics: {
    keywordUsage: evaluationResult.results.keywordUsage.score,
    readability: evaluationResult.results.readability.score,
    contentStructure: evaluationResult.results.contentStructure.score
  },
  suggestions: evaluationResult.suggestions,
  report
};
```

## Metrics Implementation

### 1. Keyword Usage Metric

This metric will evaluate how well keywords are incorporated in the content:

- **Primary Keyword**: Check for presence in title, introduction, and conclusion
- **Keyword Density**: Calculate and evaluate density (optimal range: 1-3%)
- **Secondary Keywords**: Check distribution throughout the content
- **Semantic Variations**: Identify and evaluate use of semantic variations

```typescript
export class KeywordUsageMetric extends BaseMetric<SEOMetricInput> {
  id = "keywordUsage";
  name = "Keyword Usage";
  description = "Evaluates how well keywords are incorporated in the content";
  
  async evaluate(input: SEOMetricInput): Promise<MetricResult> {
    // Implementation details...
  }
}
```

### 2. Readability Metric

This metric will assess content readability:

- **Sentence Length**: Evaluate average sentence length
- **Paragraph Length**: Check for appropriate paragraph length
- **Transition Words**: Identify use of transition words
- **Active vs. Passive Voice**: Detect and evaluate voice usage
- **Reading Level**: Calculate and evaluate reading level (e.g., Flesch-Kincaid)

```typescript
export class ReadabilityMetric extends BaseMetric<SEOMetricInput> {
  id = "readability";
  name = "Readability";
  description = "Assesses content readability (sentence length, complexity)";
  
  async evaluate(input: SEOMetricInput): Promise<MetricResult> {
    // Implementation details...
  }
}
```

### 3. Content Structure Metric

This metric will evaluate content structure:

- **Heading Hierarchy**: Check for proper heading structure (H2, H3, etc.)
- **Lists and Bullet Points**: Identify and evaluate use of lists
- **Content Length**: Assess if content length is adequate
- **Image Alt Text**: Check for image alt text (if applicable)
- **Internal Linking**: Identify internal linking opportunities

```typescript
export class ContentStructureMetric extends BaseMetric<SEOMetricInput> {
  id = "contentStructure";
  name = "Content Structure";
  description = "Evaluates heading structure, paragraph length, etc.";
  
  async evaluate(input: SEOMetricInput): Promise<MetricResult> {
    // Implementation details...
  }
}
```

## Reporting Utility

The reporting utility will format evaluation results for easy interpretation:

```typescript
function generateReport(result: EvaluationResult): string {
  const report = [
    "📊 SEO Evaluation Report",
    "========================",
    `🕒 Duration: ${result.durationMs / 1000}s`,
    `📝 Total Evaluations: ${result.total}`,
    `✅ Passed: ${result.passedCount}`,
    `❌ Failed: ${result.failedCount}`,
    `📈 Overall Score: ${(result.score * 100).toFixed(1)}%`,
    "",
    "📊 Metrics Breakdown",
    "------------------"
  ];
  
  // Add metric-specific results
  for (const [id, metricResult] of Object.entries(result.results)) {
    report.push(`${metricResult.name}:`);
    report.push(`  Score: ${(metricResult.score * 100).toFixed(1)}%`);
    report.push(`  ${metricResult.passed ? "✅ Passed" : "❌ Failed"}`);
    report.push(`  Reason: ${metricResult.reason}`);
    
    if (metricResult.suggestions && metricResult.suggestions.length > 0) {
      report.push("  Suggestions:");
      metricResult.suggestions.forEach(suggestion => {
        report.push(`    - ${suggestion}`);
      });
    }
    
    report.push("");
  }
  
  // Add overall suggestions
  if (result.suggestions.length > 0) {
    report.push("🔍 Improvement Suggestions");
    report.push("-------------------------");
    result.suggestions.forEach((suggestion, index) => {
      report.push(`${index + 1}. ${suggestion}`);
    });
  }
  
  return report.join("\n");
}
```

## Metadata Tracking

The task will use metadata to track progress and provide real-time updates:

```typescript
// Initialize metadata
metadata.replace({
  status: "running",
  startedAt: new Date().toISOString(),
  content: {
    length: content.length,
    primaryKeyword,
    secondaryKeywordsCount: secondaryKeywords.length
  },
  progress: 0
});

// Update metadata for each metric
metadata.set("currentMetric", "keywordUsage");
metadata.set("progress", 0.33);

// After completion
metadata.set("results", {
  overallScore: evaluationResult.score,
  metrics: {
    keywordUsage: evaluationResult.results.keywordUsage.score,
    readability: evaluationResult.results.readability.score,
    contentStructure: evaluationResult.results.contentStructure.score
  },
  suggestionsCount: evaluationResult.suggestions.length
});
metadata.set("status", "completed");
metadata.set("completedAt", new Date().toISOString());
metadata.set("progress", 1);
```

## Error Handling

The task will implement robust error handling:

1. **Metric Failures**: If a single metric fails, continue with other metrics
2. **Input Validation**: Validate input before processing
3. **Fallbacks**: Provide default values if certain analyses fail

## Testing Strategy

1. **Unit Testing**: Test each metric in isolation
2. **Integration Testing**: Test the evaluation with mock content
3. **Edge Cases**: Test with various content types and keyword distributions

## Future Enhancements

1. **AI-Powered Suggestions**: Use AI to generate more specific improvement suggestions
2. **Competitive Analysis**: Compare against top-ranking content
3. **Custom Metrics**: Allow adding custom evaluation metrics
4. **Multilingual Support**: Add support for evaluating content in different languages
