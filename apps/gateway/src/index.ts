type Env = {};

// TODO load from the databse

export default {
	async fetch(request: Request, env: Env): Promise<Response> {
		console.log("request", request.url);
		console.log("env", env);
		const APEX_DOMAIN = "unkey.app";
		const subdomain = new URL(request.url).hostname.replace(`.${APEX_DOMAIN}`, "");
		if (!subdomain) {
			return new Response("not found", { status: 404 });
		}
		try {
			const worker = env.DISPATCHER.get(subdomain);
			console.log(worker);
			return worker.fetch(request);
		} catch (e) {
			if (e.message.startsWith("Worker not found")) {
				return new Response("not found", { status: 404 });
			}
		}
	},
} satisfies ExportedHandler<Env>;
