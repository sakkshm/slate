import { useCallback, useEffect, useRef, useState } from "react"
import { Link, useNavigate, useParams } from "react-router-dom"
import {
  ArrowLeft,
  GitCommitHorizontal,
  ExternalLink,
  Rocket,
  Square,
  Loader2,
} from "lucide-react"
import { toast } from "sonner"

import { apiClient, type APIError } from "@/shared/api"
import { LogViewer } from "@/components/custom/LogViewer"
import { StatusBadge } from "@/components/custom/StatusBadge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"
import { formatDate, formatDuration, shortSHA } from "@/lib/format"
import type {
  BuildDetail,
  BuildStatus,
  Project,
  TriggerBuildResponse,
} from "@/types"

function isTerminal(status: BuildStatus): boolean {
  return status === "ready" || status === "failed" || status === "cancelled"
}

function BuildDetailPage() {
  const { projectID, buildID } = useParams()
  const navigate = useNavigate()

  const [project, setProject] = useState<Project | null>(null)
  const [build, setBuild] = useState<BuildDetail | null>(null)
  const [notFound, setNotFound] = useState(false)
  const [triggering, setTriggering] = useState(false)
  const statusRef = useRef<BuildStatus | null>(null)

  const fetchBuild = useCallback(() => {
    if (!projectID || !buildID) return
    apiClient
      .get<BuildDetail>(`/api/projects/${projectID}/builds/${buildID}`)
      .then((b) => {
        statusRef.current = b.status
        setBuild(b)
        setNotFound(false)
      })
      .catch((err: unknown) => {
        const e = err as APIError
        if (e.status === 404) {
          setNotFound(true)
        } else {
          toast.error(`Failed to load build (${e.code})`, {
            description: e.message,
          })
        }
      })
  }, [projectID, buildID])

  useEffect(() => {
    if (!projectID) return
    let active = true
    apiClient
      .get<Project>(`/api/projects/${projectID}`)
      .then((proj) => {
        if (active) setProject(proj)
      })
      .catch((err: unknown) => {
        const e = err as APIError
        toast.error(`Failed to load project (${e.code})`, {
          description: e.message,
        })
      })
    return () => {
      active = false
    }
  }, [projectID])

  useEffect(() => {
    statusRef.current = null
    fetchBuild()

    const interval = setInterval(() => {
      if (statusRef.current && isTerminal(statusRef.current)) {
        clearInterval(interval)
        return
      }
      fetchBuild()
    }, 3000)

    return () => clearInterval(interval)
  }, [fetchBuild])

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

  const handleCancel = async () => {
    if (!projectID || !buildID) return
    try {
      await apiClient.post(
        `/api/projects/${projectID}/builds/${buildID}/cancel`
      )
      toast.success("Cancellation requested")
      fetchBuild()
    } catch (err) {
      const e = err as APIError
      toast.error(`Failed to cancel build (${e.code})`, {
        description: e.message,
      })
    }
  }

  if (notFound) {
    return (
      <div className="mx-auto max-w-3xl px-6 py-10 text-center">
        <p className="text-muted-foreground">Build not found.</p>
        <Button
          className="mt-4"
          render={
            <Link to={`/dashboard/${projectID}/deployments`}>
              <ArrowLeft data-icon="inline-start" />
              Back to deployments
            </Link>
          }
        />
      </div>
    )
  }

  if (!build) {
    return (
      <div className="mx-auto max-w-3xl px-6 py-10">
        <Skeleton className="mb-6 h-10 w-72" />
        <Skeleton className="h-[52vh] w-full" />
      </div>
    )
  }

  const canCancel = build.status === "queued" || build.status === "building"
  const canVisit = build.status === "ready" && build.deployment_url

  return (
    <div className="mx-auto max-w-4xl px-6 py-10">
      <Button
        variant="ghost"
        size="sm"
        className="mb-4 -ml-2 text-muted-foreground"
        render={
          <Link to={`/dashboard/${projectID}/deployments`}>
            <ArrowLeft data-icon="inline-start" />
            Back to deployments
          </Link>
        }
      />

      <div className="mb-6 flex flex-wrap items-end justify-between gap-4">
        <div>
          <div className="flex items-center gap-3">
            <h1 className="font-heading text-2xl font-semibold tracking-tight">
              Deployment
            </h1>
            <StatusBadge status={build.status} />
          </div>
          <p className="mt-1 flex items-center gap-1.5 text-sm text-muted-foreground">
            <GitCommitHorizontal className="size-3.5" />
            <span className="font-mono text-xs">
              {shortSHA(build.commit_sha)}
            </span>
            <span className="truncate">{build.commit_message}</span>
          </p>
        </div>
        <div className="flex gap-2">
          {canVisit ? (
            <Button
              variant="outline"
              render={
                <a href={build.deployment_url} target="_blank" rel="noreferrer">
                  <ExternalLink data-icon="inline-start" />
                  Visit site
                </a>
              }
            />
          ) : null}
          <Button onClick={handleRedeploy} disabled={triggering}>
            <Rocket data-icon="inline-start" />
            Redeploy
          </Button>
          {canCancel ? (
            <Button variant="destructive" onClick={handleCancel}>
              <Square data-icon="inline-start" />
              Cancel
            </Button>
          ) : null}
        </div>
      </div>

      <Card size="sm" className="mb-6">
        <CardHeader>
          <CardTitle className="text-sm">Details</CardTitle>
        </CardHeader>
        <CardContent className="grid grid-cols-2 gap-x-4 gap-y-3 sm:grid-cols-4">
          <DetailItem label="Status" value={build.status} />
          <DetailItem label="Duration" value={formatDuration(build.duration)} />
          <DetailItem label="Started" value={formatDate(build.created_at)} />
          <DetailItem
            label="Project"
            value={project?.name ?? build.project_id.slice(0, 8)}
          />
        </CardContent>
      </Card>

      <h2 className="mb-2 text-sm font-medium">Build logs</h2>
      <LogViewer
        endpoint={`/api/projects/${projectID}/builds/${buildID}/logs`}
        onDone={fetchBuild}
      />

      {!isTerminal(build.status) ? (
        <div className="mt-3 flex items-center gap-2 text-xs text-muted-foreground">
          <Loader2 className="size-3.5 animate-spin" />
          Live streaming — logs update in real time.
        </div>
      ) : null}
    </div>
  )
}

function DetailItem({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <p className="text-xs text-muted-foreground">{label}</p>
      <p className="mt-0.5 text-sm font-medium capitalize">{value}</p>
    </div>
  )
}

export default BuildDetailPage
