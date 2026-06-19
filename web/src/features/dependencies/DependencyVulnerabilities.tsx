import { Link } from "react-router-dom";
import { useVulnerabilities } from "@/features/vulnerabilities/useVulnerabilities";
import { SeverityBadge } from "@/features/vulnerabilities/SeverityBadge";

interface Props {
  slug: string;
  name: string;
}

// DependencyVulnerabilities renders the "Known Vulnerabilities" section on the
// dependency detail page, listing every vulnerability that affects the package.
export function DependencyVulnerabilities({ slug, name }: Props) {
  const { data, isPending, isError } = useVulnerabilities(slug, 1, 100, "", name);

  const vulns = data?.data ?? [];

  return (
    <section className="space-y-3">
      <h3 className="text-lg font-medium">Known Vulnerabilities</h3>

      {isPending && <p className="text-zinc-500 text-sm animate-pulse">Loading vulnerabilities...</p>}
      {isError && <p className="text-red-400 text-sm">Failed to load vulnerabilities.</p>}

      {data && vulns.length === 0 && (
        <p className="text-zinc-500 text-sm">No known vulnerabilities.</p>
      )}

      {vulns.length > 0 && (
        <div className="border border-zinc-800 rounded-lg overflow-hidden">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-zinc-800 bg-zinc-900/50">
                <th className="text-left px-4 py-3 font-medium text-zinc-400">Advisory</th>
                <th className="text-left px-4 py-3 font-medium text-zinc-400">Severity</th>
                <th className="text-right px-4 py-3 font-medium text-zinc-400">CVSS</th>
              </tr>
            </thead>
            <tbody>
              {vulns.map((vuln) => (
                <tr
                  key={vuln.id}
                  className="border-b border-zinc-800/50 last:border-0 hover:bg-zinc-900/30"
                >
                  <td className="px-4 py-3 font-medium">
                    <Link
                      to={`/orgs/${slug}/vulnerabilities/${vuln.id}`}
                      className="text-zinc-100 hover:text-zinc-300 underline"
                    >
                      {vuln.osv_id}
                    </Link>
                    {vuln.cve_id && (
                      <span className="ml-2 text-xs text-zinc-500">{vuln.cve_id}</span>
                    )}
                  </td>
                  <td className="px-4 py-3">
                    <SeverityBadge severity={vuln.severity} />
                  </td>
                  <td className="px-4 py-3 text-right text-zinc-400">
                    {vuln.cvss_score != null ? vuln.cvss_score.toFixed(1) : "—"}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </section>
  );
}
