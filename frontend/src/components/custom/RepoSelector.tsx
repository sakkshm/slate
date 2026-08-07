import { useEffect, useState } from "react"
import { Search, ChevronsUpDown, Lock } from "lucide-react"
import { IconBrandGithub } from "@tabler/icons-react"
import { toast } from "sonner"

import { apiClient, type APIError } from "@/shared/api"
import type { GithubInstallationRepo } from "@/types"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog"

export function RepoSelector({
  value,
  onSelect,
}: {
  value: GithubInstallationRepo | null
  onSelect: (repo: GithubInstallationRepo) => void
}) {
  const [open, setOpen] = useState(false)
  const [repos, setRepos] = useState<GithubInstallationRepo[] | null>(null)
  const [query, setQuery] = useState("")

  useEffect(() => {
    if (!open) return

    let active = true
    apiClient
      .get<{ repositories: GithubInstallationRepo[] }>("/api/user/repos")
      .then((data) => {
        if (active) setRepos(data.repositories)
      })
      .catch((err: unknown) => {
        const e = err as APIError
        toast.error(`Failed to load repositories (${e.code})`, {
          description: e.message,
        })
      })

    return () => {
      active = false
    }
  }, [open])

  const filtered =
    repos?.filter((repo) =>
      `${repo.full_name} ${repo.name}`
        .toLowerCase()
        .includes(query.toLowerCase())
    ) ?? []

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger
        render={
          <Button variant="outline" className="w-full justify-between font-normal">
            {value ? (
              <span className="flex min-w-0 items-center gap-2">
                <IconBrandGithub className="size-4 shrink-0" />
                <span className="truncate">{value.full_name}</span>
              </span>
            ) : (
              <span className="text-muted-foreground">Choose a repository</span>
            )}
            <ChevronsUpDown className="size-4 shrink-0 text-muted-foreground" />
          </Button>
        }
      />
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>Select a repository</DialogTitle>
          <DialogDescription>
            Pick a GitHub repo you have granted Slate access to.
          </DialogDescription>
        </DialogHeader>
        <div className="relative">
          <Search className="absolute top-1/2 left-2.5 size-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            className="pl-8"
            placeholder="Search repositories…"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
          />
        </div>
        <div className="flex max-h-64 flex-col gap-1 overflow-y-auto">
          {repos === null ? (
            <p className="p-4 text-center text-sm text-muted-foreground">
              Loading…
            </p>
          ) : filtered.length === 0 ? (
            <p className="p-4 text-center text-sm text-muted-foreground">
              No repositories found.
            </p>
          ) : (
            filtered.map((repo) => (
              <button
                key={repo.id}
                type="button"
                onClick={() => {
                  onSelect(repo)
                  setOpen(false)
                }}
                className="flex w-full items-center gap-2 rounded-lg px-2 py-2 text-left text-sm transition-colors hover:bg-muted"
              >
                <IconBrandGithub className="size-4 shrink-0 text-muted-foreground" />
                <span className="min-w-0 flex-1 truncate font-medium">
                  {repo.full_name}
                </span>
                {repo.private ? (
                  <Lock className="size-3.5 shrink-0 text-muted-foreground" />
                ) : null}
              </button>
            ))
          )}
        </div>
      </DialogContent>
    </Dialog>
  )
}
