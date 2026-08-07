import { useEffect, useState } from "react"
import { Link, useNavigate, useParams } from "react-router-dom"
import { ArrowLeft, Rocket, ChevronLeft, ChevronRight } from "lucide-react"
import { toast } from "sonner"

import { apiClient, type APIError } from "@/shared/api"
import { BuildHistoryTable } from "@/components/custom/BuildHistoryTable"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"
import type {
  Build,
  ListBuildsResponse,
  Project,
  TriggerBuildResponse,
} from "@/types"

const PAGE_SIZE = 20

function DeploymentsPage() {
  const { projectID } = useParams()
  const navigate = useNavigate()

  const [project, setProject] = useState<Project | null>(null)
  const [builds, setBuilds] = useState<Build[] | null>(null)
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(0)
  const [triggering, setTriggering] = useState(false)

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
    if (!projectID) return

    let active = true
    apiClient
      .get<ListBuildsResponse>(
        `/api/projects/${projectID}/builds?limit=${PAGE_SIZE}&offset=${page * PAGE_SIZE}`
      )
      .then((data) => {
        if (active) {
          setBuilds(data.builds)
          setTotal(data.total)
        }
      })
      .catch((err: unknown) => {
        const e = err as APIError
        toast.error(`Failed to load deployments (${e.code})`, {
          description: e.message,
        })
      })

    return () => {
      active = false
    }
  }, [projectID, page])

  const handleDeploy = async () => {
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

  const pageCount = Math.max(1, Math.ceil(total / PAGE_SIZE))

  return (
    <div className="mx-auto max-w-4xl px-6 py-10">
      <Button
        variant="ghost"
        size="sm"
        className="mb-4 -ml-2 text-muted-foreground"
        render={
          <Link to={`/dashboard/${projectID}`}>
            <ArrowLeft data-icon="inline-start" />
            {project ? project.name : "Project"}
          </Link>
        }
      >
      </Button>

      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="font-heading text-2xl font-semibold tracking-tight">
            Deployments
          </h1>
          <p className="text-sm text-muted-foreground">
            {total} deployment{total === 1 ? "" : "s"}
          </p>
        </div>
        <Button onClick={handleDeploy} disabled={triggering}>
          <Rocket data-icon="inline-start" />
          Deploy now
        </Button>
      </div>

      <Card>
        <CardHeader className="border-b">
          <CardTitle className="text-sm">Build history</CardTitle>
        </CardHeader>
        <CardContent className="px-0 py-0">
          {builds === null ? (
            <div className="flex flex-col gap-2 p-4">
              {Array.from({ length: 5 }).map((_, i) => (
                <Skeleton key={i} className="h-8 w-full" />
              ))}
            </div>
          ) : builds.length === 0 ? (
            <p className="p-8 text-center text-sm text-muted-foreground">
              No deployments yet. Trigger the first one.
            </p>
          ) : (
            <BuildHistoryTable projectID={projectID ?? ""} builds={builds} />
          )}
        </CardContent>
      </Card>

      {builds !== null && builds.length > 0 && pageCount > 1 ? (
        <div className="mt-4 flex items-center justify-end gap-2">
          <Button
            variant="outline"
            size="sm"
            disabled={page === 0}
            onClick={() => setPage((p) => p - 1)}
          >
            <ChevronLeft />
            Prev
          </Button>
          <span className="text-xs text-muted-foreground">
            Page {page + 1} of {pageCount}
          </span>
          <Button
            variant="outline"
            size="sm"
            disabled={page >= pageCount - 1}
            onClick={() => setPage((p) => p + 1)}
          >
            Next
            <ChevronRight />
          </Button>
        </div>
      ) : null}
    </div>
  )
}

export default DeploymentsPage
