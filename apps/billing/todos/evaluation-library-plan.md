# Evaluation Library Plan

## Overview

The evaluation library will provide a flexible framework for assessing content quality using various metrics. It will support running multiple metrics in parallel, aggregating results, and generating comprehensive reports. The library will be designed for reuse across different types of content and evaluation criteria.

## Directory Structure

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

## Core Components

### 1. Types (types.ts)

```typescript
/**
 * Base interface for all evaluation metrics
 */
export interface Metric<TInput = any, TOutput = any> {
  id: string;
  name: string;
  description: string;
  evaluate: (input: TInput) => Promise<MetricResult<TOutput>>;
}

/**
 * Result of a metric evaluation
 */
export interface MetricResult<TOutput = any> {
  score: number;
  passed: boolean;
  reason: string;
  suggestions?: string[];
  output?: TOutput;
}

/**
 * Input for SEO metrics
 */
export interface SEOMetricInput {
  content: string;
  primaryKeyword: string;
  secondaryKeywords?: string[];
  context?: Record<string, any>;
}

/**
 * Combined results of multiple metric evaluations
 */
export interface EvaluationResult {
  score: number;
  passed: boolean;
  total: number;
  passedCount: number;
  failedCount: number;
  durationMs: number;
  results: Record<string, MetricResult>;
  suggestions: string[];
}
```

### 2. Base Metric (metrics/base.ts)

```typescript
import { Metric, MetricResult } from "../types";

/**
 * Abstract base class for all metrics
 */
export abstract class BaseMetric<TInput = any, TOutput = any> implements Metric<TInput, TOutput> {
  abstract id: string;
  abstract name: string;
  abstract description: string;
  protected threshold = 0.7;
  
  abstract evaluate(input: TInput): Promise<MetricResult<TOutput>>;
  
  protected isPassing(score: number): boolean {
    return score >= this.threshold;
  }
  
  protected createResult(
    score: number,
    reason: string,
    suggestions: string[] = [],
    output?: TOutput
  ): MetricResult<TOutput> {
    return {
      score,
      passed: this.isPassing(score),
      reason,
      suggestions,
      output,
    };
  }
}
```

### 3. Evaluate Function (evaluate.ts)

```typescript
import { Metric, MetricResult, EvaluationResult } from "./types";

/**
 * Run multiple metrics on an input and aggregate the results
 */
export async function evaluate<TInput, TOutput = any>(
  input: TInput,
  metrics: Metric<TInput, TOutput>[]
): Promise<EvaluationResult> {
  const startTime = Date.now();
  const results: Record<string, MetricResult> = {};
  
  // Run all metrics in parallel
  const evaluationPromises = metrics.map(async (metric) => {
    try {
      const result = await metric.evaluate(input);
      results[metric.id] = result;
      return { metric, result, success: true };
    } catch (error) {
      console.error(`Error evaluating metric ${metric.id}:`, error);
      return { metric, error, success: false };
    }
  });
  
  await Promise.all(evaluationPromises);
  
  // Calculate aggregate statistics
  const passedCount = Object.values(results).filter(r => r.passed).length;
  const total = Object.keys(results).length;
  const failedCount = total - passedCount;
  
  // Calculate overall score (average of all metrics)
  const totalScore = Object.values(results).reduce((sum, r) => sum + r.score, 0);
  const score = total > 0 ? totalScore / total : 0;
  
  // Collect all suggestions
  const suggestions = Object.values(results)
    .flatMap(r => r.suggestions || [])
    .filter((suggestion, index, self) => self.indexOf(suggestion) === index); // Remove duplicates
  
  return {
    score,
    passed: score >= 0.7, // Overall passing threshold
    total,
    passedCount,
    failedCount,
    durationMs: Date.now() - startTime,
    results,
    suggestions,
  };
}
```

### 4. Reporting Utility (reporting.ts)

```typescript
import { EvaluationResult } from "./types";

/**
 * Generate a formatted report from evaluation results
 */
export function generateReport(result: EvaluationResult): string {
  const report = [
    "📊 Evaluation Report",
    "===================",
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

### 5. Main Exports (index.ts)

```typescript
// Export types
export * from "./types";

// Export base metric
export { BaseMetric } from "./metrics/base";

// Export metrics
export { KeywordUsageMetric } from "./metrics/keyword";
export { ReadabilityMetric } from "./metrics/readability";
export { ContentStructureMetric } from "./metrics/structure";

// Export evaluate function
export { evaluate } from "./evaluate";

// Export reporting utilities
export { generateReport } from "./reporting";
```

## SEO Metrics Implementation

### 1. Keyword Usage Metric (metrics/keyword.ts)

```typescript
import { BaseMetric } from "./base";
import { MetricResult, SEOMetricInput } from "../types";

export class KeywordUsageMetric extends BaseMetric<SEOMetricInput> {
  id = "keywordUsage";
  name = "Keyword Usage";
  description = "Evaluates how well keywords are incorporated in the content";
  
  async evaluate(input: SEOMetricInput): Promise<MetricResult> {
    const { content, primaryKeyword, secondaryKeywords = [] } = input;
    
    // Check primary keyword presence
    const primaryKeywordLower = primaryKeyword.toLowerCase();
    const contentLower = content.toLowerCase();
    
    // Split content into sections
    const lines = content.split("\n");
    const headings = lines.filter(line => line.startsWith("#"));
    const firstParagraph = this.getFirstParagraph(content);
    const lastParagraph = this.getLastParagraph(content);
    
    // Calculate scores
    const primaryKeywordScore = this.evaluatePrimaryKeyword(
      contentLower, 
      primaryKeywordLower, 
      headings, 
      firstParagraph, 
      lastParagraph
    );
    
    const keywordDensityScore = this.evaluateKeywordDensity(
      contentLower, 
      primaryKeywordLower
    );
    
    const secondaryKeywordsScore = this.evaluateSecondaryKeywords(
      contentLower, 
      secondaryKeywords
    );
    
    // Calculate overall score
    const score = (
      primaryKeywordScore * 0.5 + 
      keywordDensityScore * 0.3 + 
      secondaryKeywordsScore * 0.2
    );
    
    // Generate suggestions
    const suggestions = this.generateSuggestions(
      primaryKeywordScore, 
      keywordDensityScore, 
      secondaryKeywordsScore,
      primaryKeyword,
      secondaryKeywords,
      content
    );
    
    return this.createResult(
      score,
      this.generateReason(score, primaryKeywordScore, keywordDensityScore, secondaryKeywordsScore),
      suggestions,
      {
        primaryKeywordScore,
        keywordDensityScore,
        secondaryKeywordsScore,
        keywordDensity: this.calculateKeywordDensity(contentLower, primaryKeywordLower)
      }
    );
  }
  
  // Helper methods for evaluation
  private getFirstParagraph(content: string): string {
    // Implementation details...
    return "";
  }
  
  private getLastParagraph(content: string): string {
    // Implementation details...
    return "";
  }
  
  private evaluatePrimaryKeyword(
    content: string, 
    primaryKeyword: string, 
    headings: string[], 
    firstParagraph: string, 
    lastParagraph: string
  ): number {
    // Implementation details...
    return 0;
  }
  
  private evaluateKeywordDensity(content: string, primaryKeyword: string): number {
    // Implementation details...
    return 0;
  }
  
  private calculateKeywordDensity(content: string, keyword: string): number {
    // Implementation details...
    return 0;
  }
  
  private evaluateSecondaryKeywords(content: string, secondaryKeywords: string[]): number {
    // Implementation details...
    return 0;
  }
  
  private generateReason(
    score: number, 
    primaryKeywordScore: number, 
    keywordDensityScore: number, 
    secondaryKeywordsScore: number
  ): string {
    // Implementation details...
    return "";
  }
  
  private generateSuggestions(
    primaryKeywordScore: number, 
    keywordDensityScore: number, 
    secondaryKeywordsScore: number,
    primaryKeyword: string,
    secondaryKeywords: string[],
    content: string
  ): string[] {
    // Implementation details...
    return [];
  }
}
```

### 2. Readability Metric (metrics/readability.ts)

```typescript
import { BaseMetric } from "./base";
import { MetricResult, SEOMetricInput } from "../types";

export class ReadabilityMetric extends BaseMetric<SEOMetricInput> {
  id = "readability";
  name = "Readability";
  description = "Assesses content readability (sentence length, complexity)";
  
  async evaluate(input: SEOMetricInput): Promise<MetricResult> {
    const { content } = input;
    
    // Calculate readability scores
    const sentenceLengthScore = this.evaluateSentenceLength(content);
    const paragraphLengthScore = this.evaluateParagraphLength(content);
    const transitionWordsScore = this.evaluateTransitionWords(content);
    const voiceScore = this.evaluateVoice(content);
    const readingLevelScore = this.evaluateReadingLevel(content);
    
    // Calculate overall score
    const score = (
      sentenceLengthScore * 0.25 + 
      paragraphLengthScore * 0.25 + 
      transitionWordsScore * 0.2 + 
      voiceScore * 0.15 + 
      readingLevelScore * 0.15
    );
    
    // Generate suggestions
    const suggestions = this.generateSuggestions(
      sentenceLengthScore,
      paragraphLengthScore,
      transitionWordsScore,
      voiceScore,
      readingLevelScore,
      content
    );
    
    return this.createResult(
      score,
      this.generateReason(score),
      suggestions,
      {
        sentenceLengthScore,
        paragraphLengthScore,
        transitionWordsScore,
        voiceScore,
        readingLevelScore,
        averageSentenceLength: this.calculateAverageSentenceLength(content),
        averageParagraphLength: this.calculateAverageParagraphLength(content),
        readingLevel: this.calculateReadingLevel(content)
      }
    );
  }
  
  // Helper methods for evaluation
  private evaluateSentenceLength(content: string): number {
    // Implementation details...
    return 0;
  }
  
  private evaluateParagraphLength(content: string): number {
    // Implementation details...
    return 0;
  }
  
  private evaluateTransitionWords(content: string): number {
    // Implementation details...
    return 0;
  }
  
  private evaluateVoice(content: string): number {
    // Implementation details...
    return 0;
  }
  
  private evaluateReadingLevel(content: string): number {
    // Implementation details...
    return 0;
  }
  
  private calculateAverageSentenceLength(content: string): number {
    // Implementation details...
    return 0;
  }
  
  private calculateAverageParagraphLength(content: string): number {
    // Implementation details...
    return 0;
  }
  
  private calculateReadingLevel(content: string): number {
    // Implementation details...
    return 0;
  }
  
  private generateReason(score: number): string {
    // Implementation details...
    return "";
  }
  
  private generateSuggestions(
    sentenceLengthScore: number,
    paragraphLengthScore: number,
    transitionWordsScore: number,
    voiceScore: number,
    readingLevelScore: number,
    content: string
  ): string[] {
    // Implementation details...
    return [];
  }
}
```

### 3. Content Structure Metric (metrics/structure.ts)

```typescript
import { BaseMetric } from "./base";
import { MetricResult, SEOMetricInput } from "../types";

export class ContentStructureMetric extends BaseMetric<SEOMetricInput> {
  id = "contentStructure";
  name = "Content Structure";
  description = "Evaluates heading structure, paragraph length, etc.";
  
  async evaluate(input: SEOMetricInput): Promise<MetricResult> {
    const { content } = input;
    
    // Calculate structure scores
    const headingHierarchyScore = this.evaluateHeadingHierarchy(content);
    const listsScore = this.evaluateLists(content);
    const contentLengthScore = this.evaluateContentLength(content);
    const imageAltTextScore = this.evaluateImageAltText(content);
    const internalLinkingScore = this.evaluateInternalLinking(content);
    
    // Calculate overall score
    const score = (
      headingHierarchyScore * 0.3 + 
      listsScore * 0.2 + 
      contentLengthScore * 0.3 + 
      imageAltTextScore * 0.1 + 
      internalLinkingScore * 0.1
    );
    
    // Generate suggestions
    const suggestions = this.generateSuggestions(
      headingHierarchyScore,
      listsScore,
      contentLengthScore,
      imageAltTextScore,
      internalLinkingScore,
      content
    );
    
    return this.createResult(
      score,
      this.generateReason(score),
      suggestions,
      {
        headingHierarchyScore,
        listsScore,
        contentLengthScore,
        imageAltTextScore,
        internalLinkingScore,
        headingCount: this.countHeadings(content),
        listCount: this.countLists(content),
        contentLength: content.length,
        imageCount: this.countImages(content),
        linkCount: this.countLinks(content)
      }
    );
  }
  
  // Helper methods for evaluation
  private evaluateHeadingHierarchy(content: string): number {
    // Implementation details...
    return 0;
  }
  
  private evaluateLists(content: string): number {
    // Implementation details...
    return 0;
  }
  
  private evaluateContentLength(content: string): number {
    // Implementation details...
    return 0;
  }
  
  private evaluateImageAltText(content: string): number {
    // Implementation details...
    return 0;
  }
  
  private evaluateInternalLinking(content: string): number {
    // Implementation details...
    return 0;
  }
  
  private countHeadings(content: string): number {
    // Implementation details...
    return 0;
  }
  
  private countLists(content: string): number {
    // Implementation details...
    return 0;
  }
  
  private countImages(content: string): number {
    // Implementation details...
    return 0;
  }
  
  private countLinks(content: string): number {
    // Implementation details...
    return 0;
  }
  
  private generateReason(score: number): string {
    // Implementation details...
    return "";
  }
  
  private generateSuggestions(
    headingHierarchyScore: number,
    listsScore: number,
    contentLengthScore: number,
    imageAltTextScore: number,
    internalLinkingScore: number,
    content: string
  ): string[] {
    // Implementation details...
    return [];
  }
}
```

## Usage Examples

### Basic Usage

```typescript
import { evaluate, KeywordUsageMetric, ReadabilityMetric, ContentStructureMetric } from "@/lib/eval";

const metrics = [
  new KeywordUsageMetric(),
  new ReadabilityMetric(),
  new ContentStructureMetric()
];

const input = {
  content: "...",
  primaryKeyword: "api authentication",
  secondaryKeywords: ["oauth", "jwt", "api keys"]
};

const result = await evaluate(input, metrics);
console.log(`Overall score: ${result.score}`);
console.log(`Passed: ${result.passed}`);
```

### With Reporting

```typescript
import { evaluate, generateReport, KeywordUsageMetric } from "@/lib/eval";

const metrics = [new KeywordUsageMetric()];
const result = await evaluate(input, metrics);
const report = generateReport(result);
console.log(report);
```

### Custom Metrics

```typescript
import { BaseMetric, evaluate } from "@/lib/eval";

class CustomMetric extends BaseMetric<SEOMetricInput> {
  id = "custom";
  name = "Custom Metric";
  description = "A custom evaluation metric";
  
  async evaluate(input: SEOMetricInput): Promise<MetricResult> {
    // Custom evaluation logic
    return this.createResult(0.8, "Custom evaluation reason", ["Custom suggestion"]);
  }
}

const metrics = [new CustomMetric()];
const result = await evaluate(input, metrics);
```

## Testing Strategy

1. **Unit Tests**: Test each metric and utility function in isolation
2. **Integration Tests**: Test the evaluation function with multiple metrics
3. **Edge Cases**: Test with various content types and edge cases

## Future Enhancements

1. **Metric Registry**: Allow registering and discovering metrics dynamically
2. **Weighted Evaluation**: Support custom weights for different metrics
3. **Async Metrics**: Support metrics that require external API calls
4. **Caching**: Cache evaluation results for improved performance
5. **Visualization**: Add visualization options for evaluation results
