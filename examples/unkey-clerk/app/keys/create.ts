"use server";
import { auth } from "@clerk/nextjs";
import { Unkey } from "@unkey/api";
export async function create(formData: FormData) {
    "use server";
    const { userId } = auth();
    if (!userId) {
        return null;
    }
    const token = process.env.UNKEY_ROOT_KEY;
    const apiId = process.env.UNKEY_API_ID;

    if (!token || !apiId) {
        return null;
    }

    const name = (formData.get("name") as string) ?? "My Awesome API";
    const unkey = new Unkey({ token });
    const key = await unkey.keys.create({
        name: name,
        ownerId: userId,
        apiId,
    });
    return { key: key.result };
}