import { z } from "zod";

export const envVarKeySchema = z
  .string()
  .trim()
// .regex(
//   /^[-._a-zA-Z0-9]+$/,
//   "No spaces or special characters, only letters, numbers, hyphens, underscores, and dots",
// );
