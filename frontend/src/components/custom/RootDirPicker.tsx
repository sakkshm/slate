import { useEffect, useState } from "react"
import { Check, ChevronRight, Folder, Loader2, RefreshCw } from "lucide-react"
import { toast } from "sonner"

import { apiClient, type APIError } from "@/shared/api"
import type { GithubRepoContentEntry, RepoContentsResponse } from "@/types"
import { Button } from "@/components/ui/button"
import { cn } from "@/lib/utils"

function contentsQuery(fullName: string, branch: string, path: string): string {
  const params = new URLSearchParams({
    fullName,
    ref: branch,
    path,
  })
  return `/api/repos/contents?${params.toString()}`
}

export function RootDirPicker({
  fullName,
  branch,
  value,
  onChange,
}: {
  fullName: string
  branch: string
  value: string
  onChange: (rootDir: string) => void
}) {
  const [path, setPath] = useState("")
  const [dirs, setDirs] = useState<GithubRepoContentEntry[] | null>(null)
  const [error, setError] = useState<string | null>(null)

  const segments = path ? path.split("/") : []

  const load = async (nextPath: string) => {
    setPath(nextPath)
    setDirs(null)
    setError(null)
    try {
      const data = await apiClient.get<RepoContentsResponse>(
        contentsQuery(fullName, branch, nextPath)
      )
      setDirs(data.entries.filter((entry) => entry.type === "dir"))
    } catch (err) {
      const e = err as APIError
      setDirs([])
      setError(`${e.code}: ${e.message}`)
      toast.error(`Failed to load directories (${e.code})`, {
        description: e.message,
      })
    }
  }

  useEffect(() => {
    if (!fullName || !branch) return
    let active = true
    apiClient
      .get<RepoContentsResponse>(contentsQuery(fullName, branch, ""))
      .then((data) => {
        if (active) setDirs(data.entries.filter((entry) => entry.type === "dir"))
      })
      .catch((err: unknown) => {
        const e = err as APIError
        if (active) {
          setDirs([])
          setError(`${e.code}: ${e.message}`)
          toast.error(`Failed to load directories (${e.code})`, {
            description: e.message,
          })
        }
      })
    return () => {
      active = false
    }
  }, [fullName, branch])

  if (!fullName || !branch) {
    return (
      <p className="rounded-lg border border-dashed bg-muted/30 p-3 text-xs text-muted-foreground">
        Choose a repository and production branch to browse its directories.
      </p>
    )
  }

  return (
    <div className="overflow-hidden rounded-lg border bg-background">
      <div className="flex items-center gap-2 border-b bg-muted/30 px-3 py-2">
        <nav className="flex min-w-0 flex-1 flex-wrap items-center gap-1 text-xs text-muted-foreground">
          <button
            type="button"
            onClick={() => load("")}
            className={cn(
              "rounded px-1 py-0.5 transition-colors hover:bg-muted hover:text-foreground",
              path === "" && "text-foreground"
            )}
          >
            Repo root
          </button>
          {segments.map((segment, index) => {
            const segmentPath = segments.slice(0, index + 1).join("/")
            return (
              <span key={segmentPath} className="flex min-w-0 items-center gap-1">
                <ChevronRight className="size-3 shrink-0" />
                <button
                  type="button"
                  onClick={() => load(segmentPath)}
                  className={cn(
                    "truncate rounded px-1 py-0.5 transition-colors hover:bg-muted hover:text-foreground",
                    index === segments.length - 1 && "font-medium text-foreground"
                  )}
                >
                  {segment}
                </button>
              </span>
            )
          })}
        </nav>
        <Button
          type="button"
          size="sm"
          variant="ghost"
          className="h-6 shrink-0 px-2 text-muted-foreground"
          onClick={() => load(path)}
          disabled={dirs === null}
        >
          <RefreshCw className={cn("size-3.5", dirs === null && "animate-spin")} />
          Reload
        </Button>
      </div>

      <div className="max-h-52 overflow-y-auto p-1.5">
        {dirs === null ? (
          <div className="flex items-center gap-2 p-3 text-xs text-muted-foreground">
            <Loader2 className="size-3.5 animate-spin" />
            Loading directories…
          </div>
        ) : error !== null ? (
          <p className="p-3 text-xs text-destructive">{error}</p>
        ) : dirs.length === 0 ? (
          <p className="p-3 text-xs text-muted-foreground">
            No subdirectories found here.
          </p>
        ) : (
          <ul className="flex flex-col">
            {dirs.map((dir) => (
              <li
                key={dir.path}
                className="flex items-center justify-between gap-2 rounded-md px-2 py-1.5 transition-colors hover:bg-muted/50"
              >
                <button
                  type="button"
                  onClick={() => load(dir.path)}
                  className="flex min-w-0 flex-1 items-center gap-2 text-left text-sm"
                >
                  <Folder className="size-4 shrink-0 text-amber-500/70" />
                  <span className="truncate">{dir.name}</span>
                </button>
                {value === dir.path ? (
                  <span className="flex shrink-0 items-center gap-1 rounded bg-muted px-1.5 py-0.5 text-xs font-medium text-muted-foreground">
                    <Check className="size-3" />
                    Selected
                  </span>
                ) : (
                  <Button
                    type="button"
                    size="sm"
                    variant="outline"
                    className="h-6 shrink-0 px-2 text-xs"
                    onClick={() => onChange(dir.path)}
                  >
                    Select
                  </Button>
                )}
              </li>
            ))}
          </ul>
        )}
      </div>

      <div className="flex items-center justify-between gap-2 border-t bg-muted/30 px-3 py-2">
        <span className="min-w-0 truncate text-xs text-muted-foreground">
          {value
            ? `Selected: ${value}`
            : "Selected: repository root"}
        </span>
        <Button
          type="button"
          size="sm"
          className="h-6 shrink-0 px-2 text-xs"
          onClick={() => onChange(path)}
          disabled={dirs === null || value === path}
        >
          {value === path ? (
            <>
              <Check className="size-3" />
              Selected
            </>
          ) : (
            "Use root dir"
          )}
        </Button>
      </div>
    </div>
  )
}
