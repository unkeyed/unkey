import { getUserId, lucia } from "@/lib/auth/index";
import { db } from "@/lib/db";

// app/login/page.tsx
export default async function Page() {
  const userId = await getUserId();

  const user = await db.query.users.findFirst({
    where: (table, { eq }) => eq(table.id, userId),
  });

  return (
    <>
      <pre>{JSON.stringify(user, null, 2)}</pre>
    </>
  );
}
