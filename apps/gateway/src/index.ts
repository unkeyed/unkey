type Env = {};

// TODO load from the databse

export default {
	async fetch(request: Request, env: Env): Promise<Response> {
		console.log("request", request.url);
		const APEX_DOMAIN = "unkey.app";
		let subdomain = new URL(request.url).hostname.replace(`.${APEX_DOMAIN}`, "");
		if (!subdomain) {
			return new Response("not found", { status: 404 });
		}
		try {
			const worker = env.DISPATCHER.get(subdomain);

			return await worker.fetch(request);
		} catch (e) {
			if (e.message.startsWith("Worker not found")) {
				return new Response("not found", { status: 404 });
			}
		}
	},
} satisfies ExportedHandler<Env>;
