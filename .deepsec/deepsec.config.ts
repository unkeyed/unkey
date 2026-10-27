import { defineConfig } from "deepsec/config";
import { generatedMatchersPlugin } from "./generated-matchers.js";

export default defineConfig({
	defaultAgent: "pi",
	defaultModel: "deepseek/deepseek-v4-flash",
	defaultThinkingLevel: "xhigh",
	ai: {
		mode: "gateway",
		provider: "vercel",
	}, // <deepsec:model-route>
	projects: [
		{ id: "unkey", root: ".." },
		// <deepsec:projects-insert-above>
	],
	plugins: [generatedMatchersPlugin],
});
