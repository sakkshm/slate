import { useState } from "react"
import { useNavigate, Link } from "react-router-dom"
import { ArrowLeft, Loader2 } from "lucide-react"
import { toast } from "sonner"

import { apiClient, type APIError } from "@/shared/api"
import { RepoSelector } from "@/components/custom/RepoSelector"
import { BranchSelector } from "@/components/custom/BranchSelector"
import { RootDirPicker } from "@/components/custom/RootDirPicker"
import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import type { CreateProjectRequest, GithubInstallationRepo, Project } from "@/types"

const errorMessages: Record<string, string> = {
  BRANCH_NOT_FND: "Branch not found in this repository.",
  ROOT_DIR_ERR: "Root directory not found in the repository.",
  GH_BRANCHES_ERR: "Unable to fetch repository branches from GitHub.",
  INST_TKN_ERR: "GitHub installation token could not be obtained.",
  GH_CONTENTS_ERR: "Unable to inspect the repository contents on GitHub.",
}

function Field({
  label,
  hint,
  children,
}: {
  label: string
  hint?: string
  children: React.ReactNode
}) {
  return (
    <label className="flex flex-col gap-1.5">
      <span className="text-sm font-medium">{label}</span>
      {children}
      {hint ? <span className="text-xs text-muted-foreground">{hint}</span> : null}
    </label>
  )
}

function NewProjectPage() {
  const navigate = useNavigate()

  const [repo, setRepo] = useState<GithubInstallationRepo | null>(null)
  const [name, setName] = useState("")
  const [branch, setBranch] = useState("")
  const [rootDir, setRootDir] = useState("")
  const [installCmd, setInstallCmd] = useState("")
  const [buildCmd, setBuildCmd] = useState("")
  const [outDir, setOutDir] = useState("")
  const [submitting, setSubmitting] = useState(false)

  const handleSelectRepo = (selected: GithubInstallationRepo) => {
    setRepo(selected)
    setName(selected.name)
    setBranch(selected.default_branch)
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()

    if (!repo) {
      toast.error("Choose a repository")
      return
    }
    if (!branch) {
      toast.error("Choose a production branch")
      return
    }
    if (!name.trim()) {
      toast.error("Project name is required")
      return
    }

    const payload: CreateProjectRequest = {
      name: name.trim(),
      repo_url: repo.html_url,
      repo_id: repo.id,
      repo_name: repo.full_name,
      full_name: repo.full_name,
      prod_branch: branch,
      root_dir: rootDir.trim() || undefined,
      install_cmd: installCmd.trim() || undefined,
      build_cmd: buildCmd.trim() || undefined,
      out_dir: outDir.trim() || undefined,
    }

    setSubmitting(true)
    try {
      const project = await apiClient.post<Project>("/api/projects", payload)
      toast.success("Project created")
      navigate(`/dashboard/${project.id}`)
    } catch (err) {
      const e = err as APIError
      toast.error(
        errorMessages[e.code] ?? `Failed to create project (${e.code})`,
        { description: e.message }
      )
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div className="mx-auto max-w-2xl px-6 py-10">
      <Button
        variant="ghost"
        size="sm"
        className="mb-4 -ml-2 text-muted-foreground"
        render={<Link to="/dashboard" />}
      >
        <ArrowLeft data-icon="inline-start" />
        Back to projects
      </Button>

      <div className="mb-6">
        <h1 className="font-heading text-2xl font-semibold tracking-tight">
          New Project
        </h1>
        <p className="text-sm text-muted-foreground">
          Connect a repository and Slate will deploy it on every push.
        </p>
      </div>

      <form onSubmit={handleSubmit} className="flex flex-col gap-4">
        <Card>
          <CardHeader>
            <CardTitle>Repository</CardTitle>
            <CardDescription>
              The GitHub repository to build and deploy.
            </CardDescription>
          </CardHeader>
          <CardContent className="flex flex-col gap-4">
            <Field label="Repository">
              <RepoSelector value={repo} onSelect={handleSelectRepo} />
            </Field>
            <Field label="Production branch">
              <BranchSelector
                key={repo?.full_name ?? "none"}
                fullName={repo?.full_name ?? ""}
                value={branch}
                onValueChange={setBranch}
              />
            </Field>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Project</CardTitle>
            <CardDescription>
              How Slate should refer to this site.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <Field label="Name">
              <Input
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder="my-project"
              />
            </Field>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Build configuration</CardTitle>
            <CardDescription>
              Optional overrides. Leave blank to auto-detect the framework.
            </CardDescription>
          </CardHeader>
          <CardContent className="flex flex-col gap-4">
            <Field label="Root directory" hint="Where your app lives. Defaults to the repository root.">
              <RootDirPicker
                key={`${repo?.full_name ?? "none"}-${branch}`}
                fullName={repo?.full_name ?? ""}
                branch={branch}
                value={rootDir}
                onChange={setRootDir}
              />
            </Field>
            <div className="grid gap-4 sm:grid-cols-3">
              <Field label="Output directory">
                <Input
                  value={outDir}
                  onChange={(e) => setOutDir(e.target.value)}
                  placeholder="auto-detected"
                />
              </Field>
              <Field label="Install command">
                <Input
                  value={installCmd}
                  onChange={(e) => setInstallCmd(e.target.value)}
                  placeholder="auto-detected"
                />
              </Field>
              <Field label="Build command">
                <Input
                  value={buildCmd}
                  onChange={(e) => setBuildCmd(e.target.value)}
                  placeholder="auto-detected"
                />
              </Field>
            </div>
          </CardContent>
        </Card>

        <div className="flex justify-end gap-2">
          <Button
            type="button"
            variant="outline"
            render={<Link to="/dashboard" />}
          >
            Cancel
          </Button>
          <Button type="submit" disabled={submitting}>
            {submitting ? (
              <Loader2 className="animate-spin" />
            ) : null}
            Create Project
          </Button>
        </div>
      </form>
    </div>
  )
}

export default NewProjectPage
