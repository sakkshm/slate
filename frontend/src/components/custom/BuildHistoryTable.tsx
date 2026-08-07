import { useNavigate } from "react-router-dom"
import { GitCommitHorizontal } from "lucide-react"

import { StatusBadge } from "@/components/custom/StatusBadge"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import { formatDate, formatDuration, shortSHA } from "@/lib/format"
import type { Build } from "@/types"

export function BuildHistoryTable({
  projectID,
  builds,
}: {
  projectID: string
  builds: Build[]
}) {
  const navigate = useNavigate()

  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>Status</TableHead>
          <TableHead>Commit</TableHead>
          <TableHead>Duration</TableHead>
          <TableHead>Created</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {builds.map((build) => (
          <TableRow
            key={build.id}
            className="cursor-pointer"
            onClick={() =>
              navigate(`/dashboard/${projectID}/deployments/${build.id}`)
            }
          >
            <TableCell>
              <StatusBadge status={build.status} />
            </TableCell>
            <TableCell className="max-w-[22rem]">
              <div className="flex items-center gap-2 overflow-hidden">
                <GitCommitHorizontal className="size-4 shrink-0 text-muted-foreground" />
                <span className="font-mono text-xs text-muted-foreground">
                  {shortSHA(build.commit_sha)}
                </span>
                <span className="truncate">{build.commit_message}</span>
              </div>
            </TableCell>
            <TableCell>{formatDuration(build.duration)}</TableCell>
            <TableCell>{formatDate(build.created_at)}</TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  )
}
