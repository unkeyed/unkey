import { Container } from "@/components/container";
import { FadeInStagger } from "@/components/fade-in";
import { StatList, StatListItem } from "@/components/stat-list";
import { db, schema, sql } from "@/lib/db";
import { getTotalVerifications } from "@/lib/tinybird";

const [workspaces, apis, keys, totalVerifications] = await Promise.all([
  db
    .select({ count: sql<number>`count(*)` })
    .from(schema.workspaces)
    .then((res) => res.at(0)?.count ?? 0),
  db
    .select({ count: sql<number>`count(*)` })
    .from(schema.apis)
    .then((res) => res.at(0)?.count ?? 0),
  db
    .select({ count: sql<number>`count(*)` })
    .from(schema.keys)
    .then((res) => res.at(0)?.count ?? 0),
  getTotalVerifications({}).then((res) => {
    return res.data.reduce((acc, curr) => acc + curr.verifications, 0);
  }),
  { next: { revalidate: 3600 } },
]);

export function Stats() {
  return (
    <div className="rounded-4xl my-20 py-8 lg:pl-12 lg:py-12 border-[.75px] backdrop-filter backdrop-blur stats-border-gradient text-white">
      <Container>
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
