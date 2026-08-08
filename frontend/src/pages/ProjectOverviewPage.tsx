import { useEffect, useState } from "react"
import { Link, useNavigate, useParams } from "react-router-dom"
import { GitBranch, Rocket, ExternalLink, History, Settings, ArrowLeft } from "lucide-react"
import { toast } from "sonner"

import { apiClient, type APIError } from "@/shared/api"
import { StatusBadge } from "@/components/custom/StatusBadge"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"
import { formatDate, formatDuration, shortSHA } from "@/lib/format"
import type {
  Build,
  BuildDetail,
  ListBuildsResponse,
  Project,
  TriggerBuildResponse,
} from "@/types"

function ProjectOverviewPage() {
  const { projectID } = useParams()
  const navigate = useNavigate()

  const [project, setProject] = useState<Project | null>(null)
  const [latest, setLatest] = useState<Build | null>(null)
  const [detail, setDetail] = useState<BuildDetail | null>(null)
  const [loading, setLoading] = useState(true)
  const [triggering, setTriggering] = useState(false)

  useEffect(() => {
    if (!projectID) return

    let active = true

    apiClient
      .get<Project>(`/api/projects/${projectID}`)
      .then((proj) => {
        if (!active) return
        setProject(proj)
        return apiClient.get<ListBuildsResponse>(
          `/api/projects/${projectID}/builds?limit=1`
        )
      })
      .then(async (builds) => {
        if (!active) return
        const b = builds?.builds[0]
        if (!b) return
        setLatest(b)
        if (b.status === "ready") {
          const d = await apiClient.get<BuildDetail>(
            `/api/projects/${projectID}/builds/${b.id}`
          )
          if (active) setDetail(d)
        }
      })
      .catch((err: unknown) => {
        const e = err as APIError
        toast.error(`Failed to load project (${e.code})`, {
          description: e.message,
        })
      })
      .finally(() => {
        if (active) setLoading(false)
      })

    return () => {
      active = false
    }
  }, [projectID])

  const handleRedeploy = async () => {
    if (!projectID) return
    setTriggering(true)
    try {
      const res = await apiClient.post<TriggerBuildResponse>(
        `/api/projects/${projectID}/builds`
      )
      toast.success("Deployment started")
      navigate(`/dashboard/${projectID}/deployments/${res.build_id}`)
    } catch (err) {
      const e = err as APIError
      toast.error(`Failed to start deployment (${e.code})`, {
        description: e.message,
      })
    } finally {
      setTriggering(false)
    }
  }

  if (loading) {
    return (
      <div className="mx-auto max-w-3xl px-6 py-10">
        <Skeleton className="mb-6 h-10 w-64" />
        <Skeleton className="h-40 w-full" />
      </div>
    )
  }

  if (!project) {
    return (
      <div className="mx-auto max-w-3xl px-6 py-10 text-center">
        <p className="text-muted-foreground">Project not found.</p>
        <Button className="mt-4" render={<Link to="/dashboard" />}>
          <ArrowLeft data-icon="inline-start" />
          Back to projects
        </Button>
      </div>
    )
  }

  const busy = latest?.status === "queued" || latest?.status === "building"

  return (
    <div className="mx-auto max-w-3xl px-6 py-10">
      <Button
        variant="ghost"
        size="sm"
        className="mb-4 -ml-2 text-muted-foreground"
        render={<Link to="/dashboard" />}
      >
        <ArrowLeft data-icon="inline-start" />
        Back to projects
      </Button>

      <div className="mb-6 flex flex-wrap items-end justify-between gap-4">
        <div>
          <div className="flex items-center gap-3">
            <h1 className="font-heading text-2xl font-semibold tracking-tight">
              {project.name}
            </h1>
            {project.framework ? (
              <Badge variant="secondary" className="capitalize">
                {project.framework}
              </Badge>
            ) : null}
          </div>
          <p className="mt-1 flex items-center gap-1.5 text-sm text-muted-foreground">
            <GitBranch className="size-3.5" />
            {project.repo_name} · {project.prod_branch}
          </p>
        </div>
        <div className="flex gap-2">
          <Button
            variant="outline"
            render={<Link to={`/dashboard/${project.id}/settings`} />}
          >
            <Settings data-icon="inline-start" />
            Settings
          </Button>
          <Button
            variant="outline"
            render={<Link to={`/dashboard/${project.id}/deployments`} />}
          >
            <History data-icon="inline-start" />
            Deployments
          </Button>
          <Button onClick={handleRedeploy} disabled={busy || triggering}>
            <Rocket data-icon="inline-start" />
            {busy ? "Deploying…" : "Redeploy"}
          </Button>
        </div>
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center justify-between">
            <span>Latest deployment</span>
            {latest ? (
              <StatusBadge status={latest.status} />
            ) : (
              <span className="text-xs font-normal text-muted-foreground">
                No deployments yet
              </span>
            )}
          </CardTitle>
          <CardDescription>
            Auto-deploys on every push to {project.prod_branch}
          </CardDescription>
        </CardHeader>
        <CardContent className="flex flex-col gap-3">
          {latest ? (
            <>
              <div className="flex items-center gap-2 text-sm">
                <span className="font-mono text-xs text-muted-foreground">
                  {shortSHA(latest.commit_sha)}
                </span>
                <span className="truncate">{latest.commit_message}</span>
              </div>
              <div className="flex flex-wrap gap-x-4 gap-y-1 text-xs text-muted-foreground">
                <span>{formatDate(latest.created_at)}</span>
                <span>Duration {formatDuration(latest.duration)}</span>
              </div>
              {detail?.deployment_url && latest.status === "ready" ? (
                <div className="mt-1 flex items-center gap-2">
                  <span className="text-xs text-muted-foreground">
                    Deployment URL
                  </span>
                  <a
                    href={detail.deployment_url}
                    target="_blank"
                    rel="noreferrer"
                    className="inline-flex items-center gap-1 text-sm text-primary underline-offset-4 hover:underline"
                  >
                    {detail.deployment_url.replace(/^https?:\/\//, "")}
                    <ExternalLink className="size-3.5" />
                  </a>
                </div>
              ) : null}
            </>
          ) : (
            <p className="text-sm text-muted-foreground">
              Deploy this project to see its production URL here.
            </p>
          )}
        </CardContent>
      </Card>
    </div>
  )
}

export default ProjectOverviewPage
