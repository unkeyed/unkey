import { Email } from "./client";



async function main(){
    const email = new Email({apiKey: process.env.RESEND_API_KEY!});

    const to = "dev@chronark.com"
    const inviteLink = `https://unkey.dev/early-access/${Buffer.from(to).toString("base64url")}`

    await email.sendEarlyAccessInvitation({to, inviteLink})
    console.log("sent invitation to", to)
}

main()