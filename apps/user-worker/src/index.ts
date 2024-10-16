import app from "../bundle/hono";
import secrets from "../bundle/secrets";
export default {
	async fetch(request, env, ctx): Promise<Response> {
		env = Object.assign(env, secrets);
		console.log("env", env);
		return app.fetch(request, env, ctx);
	},
} satisfies ExportedHandler<Env>;
