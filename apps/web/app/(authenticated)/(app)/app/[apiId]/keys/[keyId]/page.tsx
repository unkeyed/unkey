import { db, eq, schema } from "@unkey/db";

export default async function Page(props: { params: { keyId: string } }) {
  const key = await db.query.keys.findFirst({ where: eq(schema.keys.id, props.params.keyId) });

  return <pre>{JSON.stringify(key, null, 2)}</pre>;
}
