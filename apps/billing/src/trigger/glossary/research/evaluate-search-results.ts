import { google } from "@/lib/google";
import { type TaskOutput, task } from "@trigger.dev/sdk/v3";
import { generateObject } from "ai";
import { z } from "zod";
import type { exaDomainSearchTask } from "./exa-domain-search";

// Evaluation schema for content quality and relevance
const evaluationSchema = z.object({
  rating: z.number().min(1).max(10),
  justification: z.string(),
});

type EvaluateSearchOptions = {
  searchResults: TaskOutput<typeof exaDomainSearchTask>["results"];
  inputTerm: string;
};

export const evaluateSearchResults = task({
  id: "evaluate-search-results",
  run: async ({ searchResults, inputTerm }: EvaluateSearchOptions) => {
    // Set up the evaluation schema
    const batchEvaluationSchema = z.object({
      url: z.string(),
      evaluation: evaluationSchema,
    });

    const geminiResponse = await generateObject({
      model: google("gemini-2.0-flash-lite-preview-02-05") as any,
      schema: batchEvaluationSchema,
      output: "array",
      prompt: `
        Evaluate these search results for relevance to: "${inputTerm}"
        
        For each result below, return an evaluation with:
        - resultId: The ID number shown in brackets
        - evaluation:
          - rating: 1-10 scale (10 = highly relevant, 1 = irrelevant)
          - justification: Brief explanation why, including noting if content is outdated
        
        GUIDANCE ON EVALUATING CONTENT:
        - Generally prioritize content from recent years (2020-present)
        - Be cautious with older content (pre-2020)
        - Only give high ratings (7+) to older content if it's truly foundational
        - Consider the source quality
        - The ideal content is both highly relevant AND reasonably current
        
        Here are the results:
        
        ${searchResults
          .map(
            (r) => `[Result ID: ${r.id}]
        Title: ${r.title}
        URL: ${r.url}
        Published: ${r.publishedDate || "Unknown date"}
        Summary: ${r.summary}
        `,
          )
          .join("\n\n")}
        
        IMPORTANT: You must return evaluations for ALL ${searchResults.length} results.
        CRITICAL: Return a flat array of objects, not an array of arrays.
      `,
      experimental_telemetry: {
        isEnabled: true,
        functionId: "evaluate-search-results",
      },
    });

    const costs = {
      input: geminiResponse.usage.promptTokens * (0.075 / 1000000),
      output: geminiResponse.usage.completionTokens * (0.3 / 1000000),
      total:
        geminiResponse.usage.promptTokens * (0.075 / 1000000) +
        geminiResponse.usage.completionTokens * (0.3 / 1000000),
    };

    // Log token usage
    console.info(`💸 Token usage: ${geminiResponse.usage.totalTokens} tokens
      INPUT: $${costs.input}
      OUTPUT: $${costs.output}
      TOTAL: $${costs.total}
      `);

    const evaluations = geminiResponse.object;
    if (!Array.isArray(evaluations)) {
      throw new Error("Invalid evaluation response from Gemini: Not an array");
    }
    if (evaluations.length === 0) {
      throw new Error("No evaluations returned from Gemini");
    }

    // Return the original search results with evaluations attached
    return {
      costs: {
        total:
          geminiResponse.usage.promptTokens * (0.075 / 1000000) +
          geminiResponse.usage.completionTokens * (0.3 / 1000000),
        input: geminiResponse.usage.promptTokens * (0.075 / 1000000),
        output: geminiResponse.usage.completionTokens * (0.3 / 1000000),
      },
      inputTerm,
      evaluationSummary: {
        totalEvaluated: evaluations.length,
        totalIncluded: evaluations.filter(
          (evaluation) => evaluation.evaluation?.rating && evaluation.evaluation?.rating >= 7,
        ).length,
        totalExcluded: evaluations.filter(
          (evaluation) => evaluation.evaluation?.rating && evaluation.evaluation?.rating < 7,
        ).length,
      },
      evaluations,
      included: evaluations.filter(
        (evaluation) => evaluation.evaluation?.rating && evaluation.evaluation?.rating >= 7,
      ),
      excluded: evaluations.filter(
        (evaluation) => evaluation.evaluation?.rating && evaluation.evaluation?.rating < 7,
      ),
    };
  },
});
