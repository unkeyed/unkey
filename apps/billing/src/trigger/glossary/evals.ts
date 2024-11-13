import { db } from "@/lib/db-marketing/client";
import { entries } from "@/lib/db-marketing/schemas";
import {
  type EvalType,
  evals,
  ratingsSchema,
  recommendationsSchema,
} from "@/lib/db-marketing/schemas/evals";
import { openai } from "@ai-sdk/openai";
import { AbortTaskRunError, task } from "@trigger.dev/sdk/v3";
import { generateObject } from "ai";
import { eq } from "drizzle-orm";
import type { CacheStrategy } from "./_generate-glossary-entry";

type TaskInput = {
  input: string;
  onCacheHit?: CacheStrategy;
};

type RatingOptions = {
  type: EvalType;
  content: string;
};

type EvalOptions = {
  content: string;
};

// Base task for getting or creating ratings
export const getOrCreateRatingsTask = task({
  id: "get_or_create_ratings",
  run: async ({ input, onCacheHit = "stale", ...options }: TaskInput & RatingOptions) => {
    console.info(`Getting/Creating ${options.type} ratings for term: ${input}`);

    const entry = await db.query.entries.findFirst({
      where: eq(entries.inputTerm, input),
      orderBy: (entries, { desc }) => [desc(entries.createdAt)],
    });

    if (!entry) {
      throw new AbortTaskRunError(`Entry not found for term: ${input}`);
    }

    const existing = await db.query.evals.findFirst({
      where: eq(evals.entryId, entry.id),
    });

    if (existing && onCacheHit === "stale") {
      console.info(`Found existing ${options.type} ratings for term: ${input}`);
      const ratings = JSON.parse(existing.ratings);
      return { ratings };
    }

    console.info(`Generating new ${options.type} ratings for term: ${input}`);

    const systemPrompt = `You are a Senior Technical Content Evaluator with expertise in API development and technical documentation.

Your task is to evaluate the ${options.type} aspects of the content provided. Rate each aspect from 0-10:

- Accuracy (0-10): How factually correct and technically precise is the content?
- Completeness (0-10): How well does it cover all necessary aspects of the topic?
- Clarity (0-10): How clear and understandable is the content for the target audience?

Guidelines:
- Be strict but fair in your evaluation
- Consider the technical accuracy especially for API-related content
- Focus on developer experience and understanding
- Provide whole numbers only
- Ensure all ratings have clear justification`;

    const result = await generateObject({
      model: openai("gpt-4o-mini"),
      system: systemPrompt,
      prompt: `Review this content and provide numerical ratings:\n${options.content}`,
      schema: ratingsSchema,
    });

    return result;
  },
});

// Base task for getting or creating recommendations
export const getOrCreateRecommendationsTask = task({
  id: "get_or_create_recommendations",
  run: async ({ input, onCacheHit = "stale", ...options }: TaskInput & RatingOptions) => {
    console.info(`Getting/Creating ${options.type} recommendations for term: ${input}`);

    const entry = await db.query.entries.findFirst({
      where: eq(entries.inputTerm, input),
      orderBy: (entries, { desc }) => [desc(entries.createdAt)],
    });

    if (!entry) {
      throw new AbortTaskRunError(`Entry not found for term: ${input}`);
    }

    const existing = await db.query.evals.findFirst({
      where: eq(evals.entryId, entry.id),
    });

    if (existing && onCacheHit === "stale") {
      console.info(`Found existing ${options.type} recommendations for term: ${input}`);
      const recommendations = JSON.parse(existing.recommendations);
      return { recommendations };
    }

    console.info(`Generating new ${options.type} recommendations for term: ${input}`);

    const systemPrompt = `You are a Senior Technical Content Strategist specializing in API documentation.

Your task is to provide specific, actionable recommendations for improving the ${options.type} aspects of the content.

For each recommendation:
1. Identify the type of change needed (add/modify/merge/remove)
2. Provide a clear description of what needs to be changed
3. Give a specific suggestion for implementation

Guidelines:
- Focus on technical accuracy and completeness
- Consider the developer experience
- Be specific and actionable
- Avoid vague suggestions
- Ensure recommendations are practical and implementable
- Return between 2-5 recommendations`;

    const result = await generateObject({
      model: openai("gpt-4o-mini"),
      system: systemPrompt,
      prompt: `Review this content and provide recommendations:\n${options.content}`,
      schema: recommendationsSchema,
    });

    return result;
  },
});

// Technical Review Task
export const performTechnicalEvalTask = task({
  id: "perform_technical_eval",
  run: async ({ input, onCacheHit = "stale", ...options }: TaskInput & EvalOptions) => {
    console.info(`Starting technical evaluation for term: ${input}`);

    const entry = await db.query.entries.findFirst({
      where: eq(entries.inputTerm, input),
      orderBy: (entries, { desc }) => [desc(entries.createdAt)],
    });

    if (!entry) {
      throw new AbortTaskRunError(`Entry not found for term: ${input}`);
    }

    const existing = await db.query.evals.findFirst({
      where: eq(evals.entryId, entry.id),
    });

    if (existing && onCacheHit === "stale") {
      console.info(`Found existing technical evaluation for term: ${input}`);
      return {
        ratings: JSON.parse(existing.ratings),
        recommendations: JSON.parse(existing.recommendations),
      };
    }

    console.info(`Performing new technical evaluation for term: ${input}`);

    const ratingsResult = await getOrCreateRatingsTask.triggerAndWait({
      input,
      type: "technical",
      content: options.content,
      onCacheHit,
    });

    if (!ratingsResult.ok) {
      throw new AbortTaskRunError("Failed to get ratings");
    }
    console.info(`Generated technical ratings for term: ${input}`, ratingsResult.output);

    const recommendationsResult = await getOrCreateRecommendationsTask.triggerAndWait({
      input,
      type: "technical",
      content: options.content,
      onCacheHit,
    });

    if (!recommendationsResult.ok) {
      throw new AbortTaskRunError("Failed to get recommendations");
    }
    console.info(
      `Generated technical recommendations for term: ${input}`,
      recommendationsResult.output,
    );

    await db.insert(evals).values({
      entryId: entry.id,
      type: "technical",
      ratings: JSON.stringify(ratingsResult.output),
      recommendations: JSON.stringify(recommendationsResult.output.recommendations || []),
    });
    console.info(`Stored technical evaluation for term: ${input}`);

    return {
      ratings: ratingsResult.output,
      recommendations: recommendationsResult.output.recommendations,
    };
  },
});

// SEO Eval Task
export const performSEOEvalTask = task({
  id: "perform_seo_eval",
  run: async ({ input, onCacheHit = "stale", ...options }: TaskInput & EvalOptions) => {
    console.info(`Starting SEO evaluation for term: ${input}`);

    const entry = await db.query.entries.findFirst({
      where: eq(entries.inputTerm, input),
      orderBy: (entries, { desc }) => [desc(entries.createdAt)],
    });

    if (!entry) {
      throw new AbortTaskRunError(`Entry not found for term: ${input}`);
    }

    const existing = await db.query.evals.findFirst({
      where: eq(evals.entryId, entry.id),
    });

    if (existing && onCacheHit === "stale") {
      console.info(`Found existing SEO evaluation for term: ${input}`);
      return {
        ratings: JSON.parse(existing.ratings),
        recommendations: JSON.parse(existing.recommendations),
      };
    }

    console.info(`Performing new SEO evaluation for term: ${input}`);

    const ratingsResult = await getOrCreateRatingsTask.triggerAndWait({
      input,
      type: "seo",
      content: options.content,
      onCacheHit,
    });

    if (!ratingsResult.ok) {
      throw new AbortTaskRunError("Failed to get SEO ratings");
    }
    console.info(`Generated SEO ratings for term: ${input}`, ratingsResult.output);

    const recommendationsResult = await getOrCreateRecommendationsTask.triggerAndWait({
      input,
      type: "seo",
      content: options.content,
      onCacheHit,
    });

    if (!recommendationsResult.ok) {
      throw new AbortTaskRunError("Failed to get SEO recommendations");
    }
    console.info(`Generated SEO recommendations for term: ${input}`, recommendationsResult.output);

    await db.insert(evals).values({
      entryId: entry.id,
      type: "seo",
      ratings: JSON.stringify(ratingsResult.output),
      recommendations: JSON.stringify(recommendationsResult.output.recommendations || []),
    });
    console.info(`Stored SEO evaluation for term: ${input}`);

    return {
      ratings: ratingsResult.output,
      recommendations: recommendationsResult.output.recommendations,
    };
  },
});

// Editorial Eval Task
export const performEditorialEvalTask = task({
  id: "perform_editorial_eval",
  run: async ({ input, onCacheHit = "stale", ...options }: TaskInput & EvalOptions) => {
    console.info(`[workflow=glossary] [task=editorial_eval] Starting for term: ${input}`);

    const entry = await db.query.entries.findFirst({
      where: eq(entries.inputTerm, input),
      orderBy: (entries, { desc }) => [desc(entries.createdAt)],
    });

    if (!entry) {
      throw new AbortTaskRunError(`Entry not found for term: ${input}`);
    }

    const existing = await db.query.evals.findFirst({
      where: eq(evals.entryId, entry.id),
    });

    if (existing && onCacheHit === "stale") {
      console.info(
        `[workflow=glossary] [task=editorial_eval] Found existing evaluation for term: ${input}`,
      );
      return {
        ratings: JSON.parse(existing.ratings),
        recommendations: JSON.parse(existing.recommendations),
        outline: JSON.parse(existing.outline || "[]"),
      };
    }

    console.info(
      `[workflow=glossary] [task=editorial_eval] Performing new evaluation for term: ${input}`,
    );

    const ratingsResult = await getOrCreateRatingsTask.triggerAndWait({
      input,
      type: "editorial",
      content: options.content,
      onCacheHit,
    });

    if (!ratingsResult.ok) {
      throw new AbortTaskRunError(
        "[workflow=glossary] [task=editorial_eval] Failed to get editorial ratings",
      );
    }
    console.info(
      `[workflow=glossary] [task=editorial_eval] Generated ratings for term: ${input}`,
      ratingsResult.output,
    );

    const recommendationsResult = await getOrCreateRecommendationsTask.triggerAndWait({
      input,
      type: "editorial",
      content: options.content,
      onCacheHit,
    });

    if (!recommendationsResult.ok) {
      throw new AbortTaskRunError(
        "[workflow=glossary] [task=editorial_eval] Failed to get editorial recommendations",
      );
    }
    console.info(
      `[workflow=glossary] [task=editorial_eval] Generated recommendations for term: ${input}`,
      recommendationsResult.output,
    );

    await db.insert(evals).values({
      entryId: entry.id,
      type: "editorial",
      ratings: JSON.stringify(ratingsResult.output),
      recommendations: JSON.stringify(recommendationsResult.output.recommendations || []),
      outline: JSON.stringify(options.content),
    });
    console.info(`[workflow=glossary] [task=editorial_eval] Stored evaluation for term: ${input}`);

    return {
      ratings: ratingsResult.output,
      recommendations: recommendationsResult.output.recommendations,
      outline: options.content,
    };
  },
});
