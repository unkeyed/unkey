"use server";
import curl2Json from "@bany/curl-to-json";
import {
  CreateKeyCommand,
  DeleteKeyCommand,
  GetKeyCommand,
  GetVerificationsCommand,
  UpdateKeyCommand,
  VerifyKeyCommand,
} from "./unkey";

const urls = {
  createKey: "https://api.unkey.dev/v1/keys.createKey",
  getkey: "https://api.unkey.dev/v1/keys.getKey",
  verifyKey: "https://api.unkey.dev/v1/keys.verifyKey",
  updateKey: "https://api.unkey.dev/v1/keys.updateKey",
  getVerifications: "https://api.unkey.dev/v1/keys.getVerifications",
  deleteKey: "https://api.unkey.dev/v1/keys.deleteKey",
};
export const apiId = async () => {
  return process.env.PLAYGROUND_API_ID;
};
export async function handleCurlServer(curlString: string) {
  const res = await getDataFromString(curlString);
  return res;
}

export async function getDataFromString(curlString: string) {
  const reqString = curl2Json(curlString);
  const params = reqString.params;
  const data = reqString.data;
  const url = reqString.url;
  switch (url) {
    case urls.createKey:
      return await CreateKeyCommand(data.apiId);
    case urls.getkey:
      return await GetKeyCommand(params?.keyId ?? "");
    case urls.verifyKey:
      return await VerifyKeyCommand(data.key, data.apiId);
    case urls.updateKey:
      return await UpdateKeyCommand(
        data.keyId ?? undefined,
        data.ownerId ?? undefined,
        data.metaData ?? undefined,
        data.expires ? Number.parseInt(data.expires) : undefined,
        data.enabled ?? undefined,
      );
    case urls.getVerifications:
      return await GetVerificationsCommand(params?.keyId ?? "");
    case urls.deleteKey:
      return await DeleteKeyCommand(data.keyId);
    default:
  }
  return { error: "Invalid URL" };
}
