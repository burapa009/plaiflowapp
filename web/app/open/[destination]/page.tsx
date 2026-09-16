import Link from "next/link";
import { redirect } from "next/navigation";
import { sessionGET } from "@/lib/session-api";

const destinations = {
  dashboard: "tasks",
  tasks: "tasks",
  vendors: "vendors",
  imports: "vendors?focus=import",
  exports: "vendors?focus=export",
  settings: "connections",
} as const;

export default async function OpenDestination({ params }: { params: Promise<{ destination: string }> }) {
  const { destination } = await params;
  const target = destinations[destination as keyof typeof destinations];
  if (!target) redirect("/organizations");
  const response = await sessionGET("/v1/organizations");
  if (response?.status === 401) redirect("/");
  if (!response?.ok) return <section className="error-state" role="alert"><h1>ยังเปิดพื้นที่งานไม่ได้</h1><Link className="button inline-button" href={`/open/${encodeURIComponent(destination)}`}>ลองใหม่</Link></section>;
  const { organizations } = await response.json() as { organizations: { id: string; name: string; role: string }[] };
  if (organizations.length === 0) redirect("/organizations");
  if (organizations.length === 1) redirect(targetURL(organizations[0].id, target));
  return <section><p className="eyebrow">เลือก Organization</p><h1>เปิด {destination === "settings" ? "Connections" : "พื้นที่งาน"}</h1><div className="organization-list">{organizations.map((organization) => <Link className="task-card" key={organization.id} href={targetURL(organization.id, target)}><strong>{organization.name}</strong><span className="field-help">{organization.role}</span></Link>)}</div></section>;
}

function targetURL(organization: string, target: string) {
  const [path, query] = target.split("?");
  return `/o/${encodeURIComponent(organization)}/${path}${query ? `?${query}` : ""}`;
}
