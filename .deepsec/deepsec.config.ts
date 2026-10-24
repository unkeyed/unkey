import { defineConfig } from "deepsec/config";
import { generatedMatchersPlugin } from "./generated-matchers.js";

export default defineConfig({
	defaultAgent: "codex",
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
