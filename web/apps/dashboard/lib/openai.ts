import OpenAI from "openai";
import { env } from "./env";

export const openai = env().OPENAI_API_KEY ? new OpenAI({ apiKey: env().OPENAI_API_KEY }) : null;
