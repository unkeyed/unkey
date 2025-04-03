import { type TestCase, createTestRunner, errorResultSchema, okResultSchema } from "@/lib/test";
import { z } from "zod";
import { updateGlossaryContentTask } from "./update-content";
import { cleanupGlossaryUpdateTask } from "./update-content-cleanup";

const testCases: TestCase<typeof updateGlossaryContentTask>[] = [
  {
    name: "basicUpdateTest",
    input: {
      inputTerm: "MIME types",
      content:
        "# MIME Types\n\nMIME types (Multipurpose Internet Mail Extensions) are identifiers for file formats and content transmitted on the Internet. They consist of a type and subtype, such as `text/html` or `application/json`.\n\n## Common MIME Types\n\n- `text/html`: HTML documents\n- `application/json`: JSON data\n- `image/jpeg`: JPEG images\n- `application/pdf`: PDF documents\n\n## Usage in APIs\n\nMIME types are crucial in API development for specifying the format of request and response data. They're typically set in the `Content-Type` header.",
    },
    validate(result) {
      const validation = okResultSchema.safeParse(result);
      if (!validation.success) {
        console.info(
          `Test '${this.name}' failed on result validation. Expected a valid result, but got: ${JSON.stringify(result)}`,
        );
        console.info(validation.error.errors.map((e) => e.message).join("\n"));
        return false;
      }

      const expectedOutputSchema = z.object({
        updated: z.literal(true),
        prUrl: z.string().url(),
        branch: z.string(),
      });

      const outputValidation = expectedOutputSchema.safeParse(validation.data.output);
      if (!outputValidation.success) {
        console.warn(
          `Test '${this.name}' failed on output validation. Expected a valid result, but got: ${JSON.stringify(result)}`,
        );
        console.warn(outputValidation.error.errors.map((e) => e.message).join("\n"));
        return false;
      }
      console.info(`Test '${this.name}' passed. ✔︎`);
      return true;
    },
    async cleanup(result) {
      if (!result.ok) {
        console.info(`Test '${this.name}' didn't succeed, so skipping the cleanup.`);
        return;
      }

      const { prUrl, branch } = result.output;
      await cleanupGlossaryUpdateTask.triggerAndWait({ prNumber: prUrl, branch });
    },
  },
  {
    name: "emptyTermTest",
    input: {
      inputTerm: "",
      content: "This content should not be updated because the term is empty.",
    },
    validate(result) {
      const validation = errorResultSchema.safeParse(result);
      if (!validation.success) {
        console.info(
          `Test '${this.name}' failed on result validation. Expected an error result, but got: ${JSON.stringify(result)}`,
        );
        return false;
      }

      if (!validation.data.error?.message?.includes("Input term is required")) {
        console.info(
          `Test '${this.name}' failed on error message validation. Expected error message to include "Input term is required", but got: ${validation.data.error?.message}`,
        );
        return false;
      }

      console.info(`Test '${this.name}' passed. ✔︎`);
      return true;
    },
  },
  {
    name: "emptyContentTest",
    input: {
      inputTerm: "MIME types",
      content: "",
    },
    validate(result) {
      const validation = errorResultSchema.safeParse(result);
      if (!validation.success) {
        console.info(
          `Test '${this.name}' failed on result validation. Expected an error result, but got: ${JSON.stringify(result)}`,
        );
        return false;
      }

      if (!validation.data.error?.message?.includes("Content is required")) {
        console.info(
          `Test '${this.name}' failed on error message validation. Expected error message to include "Content is required", but got: ${validation.data.error?.message}`,
        );
        return false;
      }

      console.info(`Test '${this.name}' passed. ✔︎`);
      return true;
    },
  },
  {
    name: "nonExistentFileTest",
    input: {
      inputTerm: `non-existent-term-${Date.now()}`,
      content: "This content should not be updated because the file doesn't exist.",
    },
    validate(result) {
      const validation = errorResultSchema.safeParse(result);
      if (!validation.success) {
        console.info(
          `Test '${this.name}' failed on result validation. Expected an error result, but got: ${JSON.stringify(result)}`,
        );
        return false;
      }

      if (!result.error?.message?.includes("File not found")) {
        console.info(
          `Test '${this.name}' failed on error message validation. Expected error message to include "File not found", but got: ${validation.data.error?.message}`,
        );
        return false;
      }

      console.info(`Test '${this.name}' passed. ✔︎`);
      return true;
    },
  },
];

export const updateContentTest = createTestRunner({
  id: "update_content_test",
  task: updateGlossaryContentTask,
  testCases,
});
