import { createGoogleGenerativeAI } from "@ai-sdk/google";

// Call Gemini for evaluation
export const google = createGoogleGenerativeAI({
  apiKey: process.env.GEMINI_API_KEY,
});
