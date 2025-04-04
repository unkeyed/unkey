import { type TestCase, createTestRunner, errorResultSchema, okResultSchema } from "@/lib/test";
import { RelatedKeywordsOutputSchema, relatedKeywordsTask } from "./related-keywords";
import { serperSearchTask, TaskOutputSchema, KeywordSchema } from "./serper-search";
import { serperAutosuggestTask, TaskOutputSchema as SerperAutosuggestTaskOutputSchema } from "./serper-autosuggest";

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
        (k) => k.keyword.toLowerCase().includes("mime") || k.keyword.toLowerCase().includes("type"),
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
        (k) => k.keyword.toLowerCase().includes("mime") || k.keyword.toLowerCase().includes("type"),
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
        (k) => k.source !== "autosuggest" || k.confidence !== 1.0
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
        (k) => k.keyword.toLowerCase().includes("mime") || k.keyword.toLowerCase().includes("type"),
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

// Export individual test runners for each task
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

// Combined test runner for all keyword research tests
export const keywordResearchTest = createTestRunner({
  id: "keyword_research_test",
  task: serperSearchTask,
  testCases: [...serperSearchTestCases, ...serperAutosuggestTestCases],
});

// Future test cases will be added as we implement more tasks:
// 1. Serper Autosuggest test cases
// 2. Enrich Keywords test cases
// 3. Parent Task (_research-keywords) test cases
