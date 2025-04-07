import { relations } from "drizzle-orm";
import { index, int, mysqlTable, text, timestamp, varchar } from "drizzle-orm/mysql-core";
import { createInsertSchema, createSelectSchema } from "drizzle-zod";
import { z } from "zod";
import { entries } from "./entries";

export const evalTypes = ["technical", "seo", "editorial", "brand_bias"] as const;
export type EvalType = (typeof evalTypes)[number];

export const evals = mysqlTable(
  "evals",
  {
    id: int("id").primaryKey().autoincrement(),
    entryId: int("entry_id")
      .notNull()
      .references(() => entries.id),
    type: varchar("type", { enum: evalTypes, length: 1024 }),
    ratings: text("ratings").notNull(), // JSON stringified ratings
    recommendations: text("recommendations").notNull().default("[]"), // Add default empty array
    outline: text("outline").default("[]"), // Add outline field
    createdAt: timestamp("created_at").notNull().defaultNow(),
    updatedAt: timestamp("updated_at")
      .notNull()
      .defaultNow()
      .$onUpdate(() => new Date()),
  },
  (table) => ({
    entryIdIdx: index("entry_id_idx").on(table.entryId),
  }),
);

export const evalsRelations = relations(evals, ({ one }) => ({
  entry: one(entries, {
    fields: [evals.entryId],
    references: [entries.id],
  }),
}));

// Schema for LLM interaction - accepts string or number
const ratingValueSchema = z
  .union([
    z.number().int().min(0).max(10),
    z
      .string()
      .regex(/^\d+$/)
      .transform((val) => Number.parseInt(val, 10)),
  ])
  .pipe(z.number().int().min(0).max(10));

// Schema for ratings that the LLM will use
export const ratingsSchema = z
  .object({
    accuracy: ratingValueSchema,
    completeness: ratingValueSchema,
    clarity: ratingValueSchema,
  })
  .transform((val) => ({
    accuracy: Number(val.accuracy),
    completeness: Number(val.completeness),
    clarity: Number(val.clarity),
  }));

export const recommendationsSchema = z.object({
  recommendations: z.array(
    z.object({
      type: z.enum(["add", "modify", "merge", "remove"]),
      description: z.string(),
      suggestion: z.string(),
    }),
  ),
});

// schemas for brand bias evaluation to be used with LLM
export const brandBiasRatingSchema = z.object({
  commercialBias: z.number().min(0).max(10),
  neutralityScore: z.number().min(0).max(10),
  educationalValue: z.number().min(0).max(10),
});

export const brandBiasRecommendationSchema = z.object({
  recommendation: z.enum(["use_current", "fetch_neutral"]),
  dominantBrands: z.array(z.string()),
  reasoning: z.string(),
});

// DB schemas
export const insertEvalSchema = createInsertSchema(evals).extend({}).omit({ id: true });
export const selectEvalSchema = createSelectSchema(evals);

// Types
export type InsertEval = z.infer<typeof insertEvalSchema>;
export type SelectEval = typeof evals.$inferSelect;
export type Rating = z.infer<typeof ratingsSchema>;
export type Recommendation = z.infer<typeof recommendationsSchema>;
