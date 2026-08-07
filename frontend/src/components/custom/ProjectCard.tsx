import { Link } from "react-router-dom"
import { GitBranch, ChevronRight } from "lucide-react"

import { Badge } from "@/components/ui/badge"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { formatDate } from "@/lib/format"
import type { Project } from "@/types"

export function ProjectCard({ project }: { project: Project }) {
  return (
    <Link to={`/dashboard/${project.id}`}>
      <Card className="transition-colors hover:bg-muted/40">
        <CardHeader>
          <CardTitle className="flex items-center justify-between gap-2">
            <span className="truncate">{project.name}</span>
            <ChevronRight className="size-4 shrink-0 text-muted-foreground" />
          </CardTitle>
          <CardDescription className="truncate">
            {project.repo_name}
          </CardDescription>
        </CardHeader>
        <CardContent className="flex flex-wrap items-center gap-x-3 gap-y-2 text-xs text-muted-foreground">
          <span className="inline-flex items-center gap-1">
            <GitBranch className="size-3.5" />
            {project.prod_branch}
          </span>
          {project.framework ? (
            <Badge variant="secondary" className="capitalize">
              {project.framework}
            </Badge>
          ) : null}
          <span>{formatDate(project.created_at)}</span>
        </CardContent>
      </Card>
    </Link>
  )
}
