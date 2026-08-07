import { useEffect, useState } from "react"
import { Link } from "react-router-dom"
import { Plus, FolderGit2 } from "lucide-react"
import { toast } from "sonner"

import { apiClient, type APIError } from "@/shared/api"
import { ProjectCard } from "@/components/custom/ProjectCard"
import { Button } from "@/components/ui/button"
import { Skeleton } from "@/components/ui/skeleton"
import type { Project } from "@/types"

function ProjectsPage() {
  const [projects, setProjects] = useState<Project[] | null>(null)
  const [error, setError] = useState(false)

  useEffect(() => {
    let active = true
    apiClient
      .get<Project[]>("/api/projects")
      .then((data) => {
        if (active) setProjects(data)
      })
      .catch((err: unknown) => {
        const e = err as APIError
        setError(true)
        toast.error(`Failed to load projects (${e.code})`, {
          description: e.message,
        })
      })
    return () => {
      active = false
    }
  }, [])

  if (error) {
    return (
      <div className="mx-auto max-w-3xl px-6 py-10 text-center">
        <p className="text-muted-foreground">
          Something went wrong loading your projects.
        </p>
      </div>
    )
  }

  if (!projects) {
    return (
      <div className="mx-auto grid max-w-3xl gap-4 px-6 py-10">
        {Array.from({ length: 3 }).map((_, i) => (
          <Skeleton key={i} className="h-32 w-full" />
        ))}
      </div>
    )
  }

  return (
    <div className="mx-auto max-w-3xl px-6 py-10">
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="font-heading text-2xl font-semibold tracking-tight">
            Projects
          </h1>
          <p className="text-sm text-muted-foreground">
            Sites you have connected to Slate
          </p>
        </div>
        <Button render={<Link to="/dashboard/new" />}>
          <Plus data-icon="inline-start" />
          New Project
        </Button>
      </div>

      {projects.length === 0 ? (
        <div className="flex flex-col items-center gap-3 rounded-xl border border-dashed py-16 text-center">
          <FolderGit2 className="size-8 text-muted-foreground" />
          <div>
            <p className="font-medium">No projects yet</p>
            <p className="text-sm text-muted-foreground">
              Connect a repository to start deploying.
            </p>
          </div>
          <Button render={<Link to="/dashboard/new" />}>
            <Plus data-icon="inline-start" />
            Create your first project
          </Button>
        </div>
      ) : (
        <div className="grid gap-4">
          {projects.map((project) => (
            <ProjectCard key={project.id} project={project} />
          ))}
        </div>
      )}
    </div>
  )
}

export default ProjectsPage
