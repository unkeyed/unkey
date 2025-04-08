import { takeawaysSchema } from "@/lib/db-marketing/schemas/takeaways-schema";
import { type TestCase, createTestRunner, errorResultSchema, okResultSchema } from "@/lib/test";
import { z } from "zod";
import { generateTakeawaysTask } from "./generate-takeaways";

const testCases: TestCase<typeof generateTakeawaysTask>[] = [
  {
    name: "fullGenerationTest",
    input: {
      term: "MIME types",
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
      const expectedOutputSchema = z.object({
        term: z.literal("MIME types"),
        takeaways: takeawaysSchema,
      });
      const output = validation.data.output;
      const outputValidation = expectedOutputSchema.safeParse(output);
      if (!outputValidation.success) {
        console.warn(
          `Test '${this.name}' failed. Expected a valid result, but got: ${JSON.stringify(result)}`,
        );
        console.warn(outputValidation.error.errors.map((e) => e.message).join("\n"));
        return false;
      }
      console.info(`Test '${this.name}' passed. ✔︎`);
      return true;
    },
  },
  {
    name: "partialGenerationTest",
    input: {
      term: "MIME types",
      fields: {
        tldr: true,
        definitionAndStructure: [0, 1],
        usageInAPIs: {
          description: true,
        },
      },
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
        term: z.literal("MIME types"),
        takeaways: z.object({
          tldr: takeawaysSchema.shape.tldr,
          definitionAndStructure: takeawaysSchema.shape.definitionAndStructure,
          usageInAPIs: z.object({
            description: z.string(),
          }),
        }),
        fields: z.object({
          tldr: z.boolean(),
          definitionAndStructure: z.array(z.number()),
          usageInAPIs: z.object({
            description: z.boolean(),
          }),
        }),
      });
      const outputValidation = expectedOutputSchema.safeParse(validation.data.output);
      if (!outputValidation.success) {
        console.warn(
          `Test '${this.name}' failed on output validation. Expected a valid result, but got: ${JSON.stringify(validation.data.output)}`,
        );
        console.warn(
          outputValidation.error.errors.map((e) => `${e.path}: ${e.message}`).join("\n"),
        );
        return false;
      }
      console.info(`Test '${this.name}' passed. ✔︎`);
      return true;
    },
  },
  {
    name: "invalidTermTest",
    input: {
      term: "",
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

export const generateTakeawaysTest = createTestRunner({
  id: "generate_takeaways_test",
  task: generateTakeawaysTask,
  testCases,
});
