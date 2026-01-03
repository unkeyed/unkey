import { z } from "zod";

/*
 * An identifier is any string the user gives us to be used as a lookup key.
 * It must be URL safe and fit into our database (varchar 256)
 */
const identifier = z
  .string()
  .min(3)
  .max(256)
  .regex(
    /^[a-zA-Z0-9_\.:\-]*$/,
    "Only alphanumeric, underscores, periods, colons and hyphens are allowed",
  );

/**
 * A name is a user given human-readable string.
 *
 * It must not be used in URLs.
 *
 * @example the name of a key
 */
const name = z.string().min(3).max(256);

/**
 * A description is a user given human-readable string.
 *
 * It must not be used in URLs.
 *
 * @example The description of a permission
 */
const description = z.string().min(3).max(256).optional().or(z.literal(""));

const unkeyId = z
  .string()
  .regex(
    /^[a-z]{3,4}_[a-zA-Z0-9]{8,}$/,
    "Unkey IDs must include a prefix, separated by an underscore: key_abcdefg123",
  );

export const validation = {
  identifier,
  name,
  description,
  unkeyId,
};
