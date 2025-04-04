import { type TestCase, createTestRunner, errorResultSchema, okResultSchema } from "@/lib/test";
import { RelatedKeywordsOutputSchema, relatedKeywordsTask } from "./related-keywords";

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
      // This is a simplified check - in a real scenario, you might want to do more sophisticated validation
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

// Only create and export the test runner for the tasks that are implemented
// As we implement more tasks, we'll add their test cases here
export const keywordResearchTest = createTestRunner({
  id: "keyword_research_test",
  task: relatedKeywordsTask,
  testCases: relatedKeywordsTestCases,
});

// Future test cases will be added as we implement more tasks:
// 1. Serper Search test cases
// 2. Serper Autosuggest test cases
// 3. Enrich Keywords test cases
// 4. Parent Task (_research-keywords) test cases
