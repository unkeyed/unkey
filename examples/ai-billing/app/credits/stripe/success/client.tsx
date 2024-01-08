"use client";
import { useRouter } from "next/navigation";
import { useEffect, useRef } from "react";
import { createKey, listKeys, updateKey } from "./keys";

export function Client({
  setCookie,
  revalidate,
}: {
  setCookie: any;
  getCookie: any;
  revalidate: any;
}) {
  // only run once in strict mode
  const initialized = useRef(false);
  const router = useRouter();

  // TODO: try redirecting the user to an API route instead of doing it within useEffect
  useEffect(() => {
    (async () => {
      if (!initialized.current) {
        initialized.current = true;
        const keys = await listKeys();
        if (!keys.length) {
          const { key, keyId } = await createKey();
          const data = { key, keyId };
          setCookie("unkey", JSON.stringify(data));
          revalidate("/credits");
          router.push("/credits");
        } else {
          const currentKey = keys[0];
          await updateKey(currentKey);
          revalidate("/credits");
          router.push("/credits");
        }
      }
    })();
  }, []);
  return null;
}
