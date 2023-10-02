import { serve } from "https://deno.land/std@0.168.0/http/server.ts";
import { verifyKey } from "https://esm.sh/@unkey/api@0.10.0";

serve(async (req) => {
  try {
    console.log(req.headers);
    const token = req.headers.get("x-unkey-api-key");
    if (!token) {
      return new Response("No API Key provided", { status: 401 });
    }
    const { result, error } = await verifyKey(token);
    if (error) {
      // handle potential network or bad request error
      // a link to our docs will be in the `error.docs` field
      console.error(error.message);
      return new Response(JSON.stringify({ error: error.message }), {
        status: 400,
      });
    }
    if (!result.valid) {
      // do not grant access
      return new Response(JSON.stringify({ error: "API Key is not valid for this request" }), {
        status: 401,
      });
    }
    return new Response(JSON.stringify({ result }), { status: 200 });
  } catch (error) {
    return new Response(JSON.stringify({ error: error.message }), {
      status: 500,
    });
  }
});

// To invoke this you can use the following command

/*
curl -XPOST -H 'Authorization: Bearer <SUPBASE_BEARER_TOKEN>' -H 'x-unkey-api-key: <UNKEY_API_KEY>' -H "Content-type: application/json" 'http://localhost:54321/functions/v1/hello-world'
*/
