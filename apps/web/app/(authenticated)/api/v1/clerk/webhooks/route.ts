import { Webhook } from 'svix'
import { headers } from 'next/headers'
import { WebhookEvent } from '@clerk/nextjs/server'
 
export async function POST(req: Request) : Promise<any> {
 
  const WEBHOOK_SECRET = process.env.WEBHOOK_SECRET
  const loopsAPIKey = process.env.LOOPS_API_KEY
  if (!WEBHOOK_SECRET) {
    throw new Error('Please add WEBHOOK_SECRET from Clerk Dashboard to .env or .env.local')
  }
 
  // Get the headers
  const headerPayload = headers();
  const svix_id = headerPayload.get("svix-id");
  const svix_timestamp = headerPayload.get("svix-timestamp");
  const svix_signature = headerPayload.get("svix-signature");
 
  // If there are no headers, error out
  if (!svix_id || !svix_timestamp || !svix_signature) {
    return new Response('Error occured -- no svix headers', {
      status: 400
    })
  }
 
  // Get the body
  const payload = await req.json()
  const body = JSON.stringify(payload);
 
  // Create a new SVIX instance with your secret.
  const wh = new Webhook(WEBHOOK_SECRET);
 
  let evt: WebhookEvent
 
  // Verify the payload with the headers
  try {
    evt = wh.verify(body, {
      "svix-id": svix_id,
      "svix-timestamp": svix_timestamp,
      "svix-signature": svix_signature,
    }) as WebhookEvent
  } catch (err) {
    console.error('Error verifying webhook:', err);
    return new Response('Error occured', {
      status: 400
    })
  }
 
  // Get the eventType
  const eventType = evt.type;
  if (eventType === "user.created") {
    // we only care about the first email address, so we can just grab the first one
    const email = evt.data.email_addresses[0].email_address;
    if (!email) {
      return new Response('Error occured -- no email address found', {
        status: 400
      });
    }
    try {
      const loopsResponse = await fetch("https://app.loops.so/api/v1/contacts/create", {
        method: "POST",
        headers: {
          Authorization: `Bearer ${loopsAPIKey}`,
          "Content-Type": "application/json",
        },
        body: JSON.stringify({ email: email, source: "clerk-signup" }),
      });

      if (loopsResponse.status < 200 || loopsResponse.status >= 300) {
        console.error('Error creating a user in loops:', await loopsResponse.json());
        return new Response('Error occured -- Creating a user in loops' , {
          status: 500
        });
      }
      return new Response('Success', {
        status: 200
      });
    } catch (err) {
      console.error('Error sending to loops:', err);
      return new Response('Error occured -- loops API error', {
        status: 400
      });
    }
  }
}