import { Badge } from "@/components/ui/badge"
import { buildStatusMeta } from "@/lib/format"
import { cn } from "@/lib/utils"
import type { BuildStatus } from "@/types"

export function StatusBadge({ status }: { status: BuildStatus }) {
  const meta = buildStatusMeta(status)
  return (
    <Badge
      variant="outline"
      className={cn("gap-1.5 border", meta.badge)}
    >
      <span className={cn("size-1.5 shrink-0 rounded-full", meta.dot)} />
      {meta.label}
    </Badge>
  )
}
