import { type TestCase, createTestRunner, errorResultSchema, okResultSchema } from "@/lib/test";
import type { ZodIssue } from "zod";
import { researchKeywords } from "./_research-keywords";
import {
  TaskOutputSchema as EnrichKeywordsTaskOutputSchema,
  enrichKeywordsTask,
} from "./enrich-keywords";
import { RelatedKeywordsOutputSchema, relatedKeywordsTask } from "./related-keywords";
import {
  TaskOutputSchema as SerperAutosuggestTaskOutputSchema,
  serperAutosuggestTask,
} from "./serper-autosuggest";
import { KeywordSchema, TaskOutputSchema, serperSearchTask } from "./serper-search";

// Test cases for the parent research-keywords task
const researchKeywordsTestCases: TestCase<typeof researchKeywords>[] = [
  {
    name: "researchKeywordsBasicTest",
    input: {
      inputTerm: "MIME types",
    },
    validate(result) {
      const validation = okResultSchema.safeParse(result);
      if (!validation.success) {
        console.info(
          `Test '${this.name}' failed. Expected a valid result, but got: ${JSON.stringify(result)}`,
        );
        console.info(validation.error.errors.map((e) => e.message).join("\n"));
        return false;
      }

      const output = validation.data.output;

      // Check if we have keywords
      if (!output.keywords || output.keywords.length === 0) {
        console.warn(
          `Test '${this.name}' failed. Expected keywords to be non-empty, but got: ${output.keywords?.length ?? 0} keywords`,
        );
        return false;
      }

      // Check metadata structure
      if (
        !output.metadata ||
        typeof output.metadata.totalKeywords !== "number" ||
        !output.metadata.sources ||
        typeof output.metadata.sources.relatedKeywords !== "number" ||
        typeof output.metadata.sources.serperSearch !== "number" ||
        typeof output.metadata.sources.serperAutosuggest !== "number"
      ) {
        console.warn(
          `Test '${this.name}' failed. Invalid metadata structure: ${JSON.stringify(output.metadata)}`,
        );
        return false;
      }

      // Check if total keywords matches the array length
      if (output.metadata.totalKeywords !== output.keywords.length) {
        console.warn(
          `Test '${this.name}' failed. Metadata total (${output.metadata.totalKeywords}) doesn't match actual keywords length (${output.keywords.length})`,
        );
        return false;
      }

      // Check if source counts add up
      const totalFromSources =
        output.metadata.sources.relatedKeywords +
        output.metadata.sources.serperSearch +
        output.metadata.sources.serperAutosuggest;

      // Note: totalFromSources might be greater than totalKeywords due to deduplication
      if (totalFromSources < output.metadata.totalKeywords) {
        console.warn(
          `Test '${this.name}' failed. Source totals (${totalFromSources}) less than total keywords (${output.metadata.totalKeywords})`,
        );
        return false;
      }

      // Check deduplication metadata
      if (
        !output.metadata.deduplication ||
        typeof output.metadata.deduplication.total !== "number" ||
        typeof output.metadata.deduplication.skippedEnrichment !== "number" ||
        typeof output.metadata.deduplication.duplicatesRemoved !== "number"
      ) {
        console.warn(
          `Test '${this.name}' failed. Invalid deduplication metadata structure: ${JSON.stringify(output.metadata.deduplication)}`,
        );
        return false;
      }

      // Verify case-insensitive deduplication
      const normalizedKeywords = new Set(
        output.keywords.map((k: { keyword: string }) => k.keyword.toLowerCase().trim()),
      );
      if (normalizedKeywords.size !== output.keywords.length) {
        console.warn(`Test '${this.name}' failed. Found case-insensitive duplicates in results`);
        return false;
      }

      // Check if deduplication counts make sense
      if (
        output.metadata.deduplication.total < output.keywords.length ||
        output.metadata.deduplication.duplicatesRemoved !==
          output.metadata.deduplication.total - output.keywords.length
      ) {
        console.warn(`Test '${this.name}' failed. Deduplication counts don't add up`);
        return false;
      }

      // Verify that we're not unnecessarily enriching keywords
      if (output.metadata.deduplication.skippedEnrichment < 0) {
        console.warn(
          `Test '${this.name}' failed. Invalid skipped enrichment count: ${output.metadata.deduplication.skippedEnrichment}`,
        );
        return false;
      }

      // Check keyword structure
      const invalidKeywords = output.keywords.filter(
        (k: { keyword: string; volume: number; cpc: number; competition: number }) =>
          typeof k.keyword !== "string" ||
          typeof k.volume !== "number" ||
          typeof k.cpc !== "number" ||
          typeof k.competition !== "number",
      );

      if (invalidKeywords.length > 0) {
        console.warn(
          `Test '${this.name}' failed. Found ${invalidKeywords.length} invalid keyword structures`,
        );
        return false;
      }

      // Check for topic relevance
      const hasMimeKeywords = output.keywords.some(
        (k: { keyword: string }) =>
          k.keyword.toLowerCase().includes("mime") || k.keyword.toLowerCase().includes("type"),
      );

      if (!hasMimeKeywords) {
        console.warn(`Test '${this.name}' failed. Expected keywords to be related to "MIME types"`);
        return false;
      }

      console.info(`Test '${this.name}' passed. ✔︎`);
      return true;
    },
  },
  {
    name: "researchKeywordsEmptyInputTest",
    input: {
      inputTerm: "", // Empty term should cause an error
    },
    validate(result) {
      const validation = errorResultSchema.safeParse(result);
      if (!validation.success) {
        console.info(
          `Test '${this.name}' failed. Expected an error result, but got: ${JSON.stringify(result)}`,
        );
        return false;
      }

      console.info(`Test '${this.name}' passed. ✔︎`);
      return true;
    },
  },
];

// Test cases for related-keywords task
const relatedKeywordsTestCases: TestCase<typeof relatedKeywordsTask>[] = [
  {
    name: "relatedKeywordsBasicTest",
    input: {
      inputTerm: "MIME types",
    },
    validate(result) {
      const validation = okResultSchema.safeParse(result);
      if (!validation.success) {
        console.info(
          `Test '${this.name}' failed. Expected a valid result, but got: ${JSON.stringify(result)}`,
        );
        console.info(validation.error.errors.map((e) => e.message).join("\n"));
        return false;
      }

      const outputValidation = RelatedKeywordsOutputSchema.safeParse(validation.data.output);
      if (!outputValidation.success) {
        console.warn(
          `Test '${this.name}' failed. Expected a valid output format, but got: ${JSON.stringify(validation.data.output)}`,
        );
        console.warn(outputValidation.error.errors.map((e) => e.message).join("\n"));
        return false;
      }

      const output = outputValidation.data;

      // Check if the inputTerm matches
      if (output.inputTerm !== "MIME types") {
        console.warn(
          `Test '${this.name}' failed. Expected inputTerm to be "MIME types", but got: ${output.inputTerm}`,
        );
        return false;
      }

      // Check if there are keywords in the result
      if (output.keywordIdeas.length === 0) {
        console.warn(
          `Test '${this.name}' failed. Expected keywords to be non-empty, but got: ${output.keywordIdeas.length} keywords`,
        );
        return false;
      }

      // Check if the keywords are related to the topic
      const hasMimeKeywords = output.keywordIdeas.some(
        (k: { keyword: string }) =>
          k.keyword.toLowerCase().includes("mime") || k.keyword.toLowerCase().includes("type"),
      );

      if (!hasMimeKeywords) {
        console.warn(`Test '${this.name}' failed. Expected keywords to be related to "MIME types"`);
        return false;
      }

      console.info(`Test '${this.name}' passed. ✔︎`);
      return true;
    },
  },
  {
    name: "relatedKeywordsErrorTest",
    input: {
      inputTerm: "", // Empty term should cause an error
    },
    validate(result) {
      const validation = errorResultSchema.safeParse(result);
      if (!validation.success) {
        console.info(
          `Test '${this.name}' failed. Expected an error result, but got: ${JSON.stringify(result)}`,
        );
        return false;
      }

      console.info(`Test '${this.name}' passed. ✔︎`);
      return true;
    },
  },
];

// Test cases for serper-search task
const serperSearchTestCases: TestCase<typeof serperSearchTask>[] = [
  {
    name: "serperSearchBasicTest",
    input: {
      inputTerm: "MIME types",
    },
    validate(result) {
      const validation = okResultSchema.safeParse(result);
      if (!validation.success) {
        console.info(
          `Test '${this.name}' failed. Expected a valid result, but got: ${JSON.stringify(result)}`,
        );
        console.info(validation.error.errors.map((e) => e.message).join("\n"));
        return false;
      }

      const outputValidation = TaskOutputSchema.safeParse(validation.data.output);
      if (!outputValidation.success) {
        console.warn(
          `Test '${this.name}' failed. Expected a valid output format, but got: ${JSON.stringify(validation.data.output)}`,
        );
        console.warn(outputValidation.error.errors.map((e) => e.message).join("\n"));
        return false;
      }

      const output = outputValidation.data;

      // Check if the inputTerm matches
      if (output.inputTerm !== "MIME types") {
        console.warn(
          `Test '${this.name}' failed. Expected inputTerm to be "MIME types", but got: ${output.inputTerm}`,
        );
        return false;
      }

      // Check if there are organic results
      if (!output.searchResult.organic || output.searchResult.organic.length === 0) {
        console.warn(
          `Test '${this.name}' failed. Expected organic results to be non-empty, but got: ${output.searchResult.organic?.length ?? 0} results`,
        );
        return false;
      }

      // Check if there are keywords in the result
      if (output.keywords.length === 0) {
        console.warn(
          `Test '${this.name}' failed. Expected keywords to be non-empty, but got: ${output.keywords.length} keywords`,
        );
        return false;
      }

      // Check if we have keywords from both sources
      const hasRelatedSearchKeywords = output.keywords.some((k) => k.source === "related_search");
      const hasLLMKeywords = output.keywords.some((k) => k.source === "llm_extracted");

      if (!hasRelatedSearchKeywords || !hasLLMKeywords) {
        console.warn(
          `Test '${this.name}' failed. Expected keywords from both sources (related_search and llm_extracted)`,
        );
        return false;
      }

      // Check if the keywords are related to the topic
      const hasMimeKeywords = output.keywords.some(
        (k: { keyword: string }) =>
          k.keyword.toLowerCase().includes("mime") || k.keyword.toLowerCase().includes("type"),
      );

      if (!hasMimeKeywords) {
        console.warn(`Test '${this.name}' failed. Expected keywords to be related to "MIME types"`);
        return false;
      }

      // Check keyword schema compliance
      const keywordValidations = output.keywords.map((k) => KeywordSchema.safeParse(k));
      const invalidKeywords = keywordValidations.filter((v) => !v.success);
      if (invalidKeywords.length > 0) {
        console.warn(
          `Test '${this.name}' failed. Found ${invalidKeywords.length} invalid keywords`,
        );
        return false;
      }

      console.info(`Test '${this.name}' passed. ✔︎`);
      return true;
    },
  },
  {
    name: "serperSearchEmptyInputTest",
    input: {
      inputTerm: "", // Empty term should cause an error
    },
    validate(result) {
      const validation = errorResultSchema.safeParse(result);
      if (!validation.success) {
        console.info(
          `Test '${this.name}' failed. Expected an error result, but got: ${JSON.stringify(result)}`,
        );
        return false;
      }

      // Since error is typed as unknown in errorResultSchema, we need to do runtime checks
      const error = validation.data.error;
      if (typeof error !== "object" || !error || !("message" in error)) {
        console.warn(
          `Test '${this.name}' failed. Expected error to have a message property, but got: ${JSON.stringify(error)}`,
        );
        return false;
      }

      // Now TypeScript knows error.message exists and is a property
      const message = (error as { message: unknown }).message;
      if (typeof message !== "string" || !message.includes("Input term is required")) {
        console.warn(
          `Test '${this.name}' failed. Expected error message to include "Input term is required", but got: ${message}`,
        );
        return false;
      }

      console.info(`Test '${this.name}' passed. ✔︎`);
      return true;
    },
  },
];

// Test cases for serper-autosuggest task
const serperAutosuggestTestCases: TestCase<typeof serperAutosuggestTask>[] = [
  {
    name: "serperAutosuggestBasicTest",
    input: {
      inputTerm: "MIME types",
    },
    validate(result) {
      const validation = okResultSchema.safeParse(result);
      if (!validation.success) {
        console.info(
          `Test '${this.name}' failed. Expected a valid result, but got: ${JSON.stringify(result)}`,
        );
        console.info(validation.error.errors.map((e) => e.message).join("\n"));
        return false;
      }

      const outputValidation = SerperAutosuggestTaskOutputSchema.safeParse(validation.data.output);
      if (!outputValidation.success) {
        console.warn(
          `Test '${this.name}' failed. Expected a valid output format, but got: ${JSON.stringify(validation.data.output)}`,
        );
        console.warn(outputValidation.error.errors.map((e) => e.message).join("\n"));
        return false;
      }

      const output = outputValidation.data;

      // Check if the inputTerm matches
      if (output.inputTerm !== "MIME types") {
        console.warn(
          `Test '${this.name}' failed. Expected inputTerm to be "MIME types", but got: ${output.inputTerm}`,
        );
        return false;
      }

      // Check if there are keywords in the result
      if (output.keywords.length === 0) {
        console.warn(
          `Test '${this.name}' failed. Expected keywords to be non-empty, but got: ${output.keywords.length} keywords`,
        );
        return false;
      }

      // Check if all keywords have the correct source and confidence
      const invalidKeywords = output.keywords.filter(
        (k) => k.source !== "autosuggest" || k.confidence !== 1.0,
      );
      if (invalidKeywords.length > 0) {
        console.warn(
          `Test '${this.name}' failed. Found keywords with incorrect source or confidence:`,
          invalidKeywords,
        );
        return false;
      }

      // Check if the keywords are related to the topic
      const hasMimeKeywords = output.keywords.some(
        (k: { keyword: string }) =>
          k.keyword.toLowerCase().includes("mime") || k.keyword.toLowerCase().includes("type"),
      );
      if (!hasMimeKeywords) {
        console.warn(
          `Test '${this.name}' failed. Expected keywords to be related to "MIME types", but found none`,
        );
        return false;
      }

      console.info(`Test '${this.name}' passed. ✔︎`);
      return true;
    },
  },
  {
    name: "serperAutosuggestEmptyInputTest",
    input: {
      inputTerm: "", // Empty term should cause an error
    },
    validate(result) {
      const validation = errorResultSchema.safeParse(result);
      if (!validation.success) {
        console.info(
          `Test '${this.name}' failed. Expected an error result, but got: ${JSON.stringify(result)}`,
        );
        return false;
      }

      console.info(`Test '${this.name}' passed. ✔︎`);
      return true;
    },
  },
];

// Test cases for enrich-keywords task
const enrichKeywordsTestCases: TestCase<typeof enrichKeywordsTask>[] = [
  {
    name: "enrichKeywordsBasicTest",
    input: {
      keywords: [
        {
          keyword: "MIME types",
          source: "llm_extracted",
          confidence: 1,
          context: "The main topic of the search results",
        },
      ],
    },
    validate(result) {
      const validation = okResultSchema.safeParse(result);
      if (!validation.success) {
        console.info(
          `Test '${this.name}' failed. Expected a valid result, but got: ${JSON.stringify(result)}`,
        );
        console.info(validation.error.errors.map((error: ZodIssue) => error.message).join("\n"));
        return false;
      }

      const outputValidation = EnrichKeywordsTaskOutputSchema.safeParse(validation.data.output);
      if (!outputValidation.success) {
        console.warn(
          `Test '${this.name}' failed. Expected a valid output format, but got: ${JSON.stringify(validation.data.output)}`,
        );
        console.warn(
          outputValidation.error.errors.map((error: ZodIssue) => error.message).join("\n"),
        );
        return false;
      }

      const output = outputValidation.data;

      // Verify enriched data structure
      const firstKeyword = output.enrichedKeywords[0];
      if (
        typeof firstKeyword.volume !== "number" ||
        typeof firstKeyword.cpc !== "number" ||
        typeof firstKeyword.competition !== "number"
      ) {
        console.warn(
          `Test '${this.name}' failed. Missing or invalid enrichment metrics in response: ${JSON.stringify(firstKeyword)}`,
        );
        return false;
      }

      // Verify source attribution is preserved
      if (firstKeyword.source !== "llm_extracted" || firstKeyword.confidence !== 1) {
        console.warn(`Test '${this.name}' failed. Source attribution not preserved`);
        return false;
      }

      console.info(`Test '${this.name}' passed. ✔︎`);
      return true;
    },
  },
  {
    name: "enrichKeywordsAutosuggestTest",
    input: {
      keywords: [
        {
          keyword: "mime types list",
          source: "autosuggest",
          confidence: 1,
          context: "Direct Google autocomplete suggestion",
        },
      ],
    },
    validate(result) {
      const validation = okResultSchema.safeParse(result);
      if (!validation.success) {
        console.info(
          `Test '${this.name}' failed. Expected a valid result, but got: ${JSON.stringify(result)}`,
        );
        return false;
      }

      const outputValidation = EnrichKeywordsTaskOutputSchema.safeParse(validation.data.output);
      if (!outputValidation.success) {
        console.warn(
          `Test '${this.name}' failed. Expected a valid output format, but got: ${JSON.stringify(validation.data.output)}`,
        );
        return false;
      }

      const output = outputValidation.data;

      // Verify source is preserved
      if (output.enrichedKeywords[0].source !== "autosuggest") {
        console.warn(`Test '${this.name}' failed. Source not preserved correctly`);
        return false;
      }

      console.info(`Test '${this.name}' passed. ✔︎`);
      return true;
    },
  },
  {
    name: "enrichKeywordsLargeTest",
    input: {
      keywords: Array.from({ length: 101 }, (_, i) => ({
        keyword: `test keyword ${i}`,
        source: "autosuggest",
        confidence: 1,
        context: "Test context",
      })),
    },
    validate(result) {
      const validation = errorResultSchema.safeParse(result);
      if (!validation.success) {
        console.info(
          `Test '${this.name}' failed. Expected an error result, but got: ${JSON.stringify(result)}`,
        );
        return false;
      }

      const error = validation.data.error;
      if (typeof error !== "object" || !error || !("message" in error)) {
        console.warn(`Test '${this.name}' failed. Expected error to have a message property`);
        return false;
      }

      const message = (error as { message: unknown }).message;
      if (
        typeof message !== "string" ||
        !message.includes("Cannot process more than 100 keywords")
      ) {
        console.warn(`Test '${this.name}' failed. Expected error about keyword limit`);
        return false;
      }

      console.info(`Test '${this.name}' passed. ✔︎`);
      return true;
    },
  },
  {
    name: "enrichKeywordsInvalidTest",
    input: {
      keywords: [
        {
          keyword: "this_keyword_should_not_exist_in_api_response",
          source: "autosuggest",
          confidence: 1,
          context: "Invalid keyword test",
        },
      ],
    },
    validate(result) {
      const validation = okResultSchema.safeParse(result);
      if (!validation.success) {
        console.info(
          `Test '${this.name}' failed. Expected a valid result, but got: ${JSON.stringify(result)}`,
        );
        return false;
      }

      const outputValidation = EnrichKeywordsTaskOutputSchema.safeParse(validation.data.output);
      if (!outputValidation.success) {
        console.warn(
          `Test '${this.name}' failed. Expected a valid output format, but got: ${JSON.stringify(validation.data.output)}`,
        );
        return false;
      }

      const output = outputValidation.data;

      // Verify that the keyword was skipped
      if (output.metadata.skippedCount !== 1) {
        console.warn(
          `Test '${this.name}' failed. Expected 1 skipped keyword, got ${output.metadata.skippedCount}`,
        );
        return false;
      }

      // Verify the specific keyword was skipped
      if (
        !output.metadata.skippedKeywords.includes("this_keyword_should_not_exist_in_api_response")
      ) {
        console.warn(`Test '${this.name}' failed. Expected keyword to be in skipped list`);
        return false;
      }

      // Verify no enriched keywords were returned
      if (output.enrichedKeywords.length !== 0) {
        console.warn(
          `Test '${this.name}' failed. Expected no enriched keywords, got ${output.enrichedKeywords.length}`,
        );
        return false;
      }

      console.info(`Test '${this.name}' passed. ✔︎`);
      return true;
    },
  },
];

// Export individual test runners for each task
export const researchKeywordsTest = createTestRunner({
  id: "research_keywords_test",
  task: researchKeywords,
  testCases: researchKeywordsTestCases,
});

export const relatedKeywordsTest = createTestRunner({
  id: "related_keywords_test",
  task: relatedKeywordsTask,
  testCases: relatedKeywordsTestCases,
});

export const serperSearchTest = createTestRunner({
  id: "serper_search_test",
  task: serperSearchTask,
  testCases: serperSearchTestCases,
});

export const serperAutosuggestTest = createTestRunner({
  id: "serper_autosuggest_test",
  task: serperAutosuggestTask,
  testCases: serperAutosuggestTestCases,
});

export const enrichKeywordsTest = createTestRunner({
  id: "enrich_keywords_test",
  task: enrichKeywordsTask,
  testCases: enrichKeywordsTestCases,
});
