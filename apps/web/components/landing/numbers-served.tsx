import { Container } from "@/components/landing/container";
import { FadeIn, FadeInStagger } from "@/components/landing/fade-in";
import { StatList, StatListItem } from "@/components/landing/stat-list";
import { db, schema, sql } from "@/lib/db";
import { getTotalVerifications } from "@/lib/tinybird";

const [workspaces, apis, keys, totalVerifications] = await Promise.all([
  db
    .select({ count: sql<number>`count(*)` })
    .from(schema.workspaces)
    .then((res) => res.at(0)?.count ?? 0)
    .catch((err) => {
      console.error(err);
      return 0;
    }),
  db
    .select({ count: sql<number>`count(*)` })
    .from(schema.apis)
    .then((res) => res.at(0)?.count ?? 0)
    .catch((err) => {
      console.error(err);
      return 0;
    }),
  db
    .select({ count: sql<number>`count(*)` })
    .from(schema.keys)
    .then((res) => res.at(0)?.count ?? 0)
    .catch((err) => {
      console.error(err);
      return 0;
    }),
  getTotalVerifications({})
    .then((res) => {
      return res.data.reduce((acc, curr) => acc + curr.verifications, 0);
    })
    .catch((err) => {
      console.error(err);
      return 0;
    }),
  { next: { revalidate: 3600 } },
]);

export function NumbersServed() {
  return (
    <div className="rounded-4xl mt-24 sm:mt-32 lg:mt-32">
      <Container className="">
        <FadeIn className="flex items-center gap-x-8">
          <h2 className="font-display mb-8 text-center text-2xl font-semibold tracking-wider text-black sm:text-left">
            We serve
          </h2>
          <div className="h-px flex-auto" />
        </FadeIn>
        <FadeInStagger faster>
          <StatList>
            <StatListItem
              value={Intl.NumberFormat("en", { notation: "compact" }).format(workspaces)}
              label="Workspaces"
            />
            <StatListItem
              value={Intl.NumberFormat("en", { notation: "compact" }).format(apis)}
              label="APIs"
            />
            <StatListItem
              value={Intl.NumberFormat("en", { notation: "compact" }).format(keys)}
              label="Keys"
            />
            <StatListItem
              value={Intl.NumberFormat("en", { notation: "compact" }).format(totalVerifications)}
              label="Verifications"
            />
          </StatList>
        </FadeInStagger>
      </Container>
    </div>
  );
}
