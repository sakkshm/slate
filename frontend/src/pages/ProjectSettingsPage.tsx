import { useEffect, useState } from "react"
import { Link, useNavigate, useParams } from "react-router-dom"
import {
  ArrowLeft,
  Eye,
  EyeOff,
  GitBranch,
  History,
  KeyRound,
  Loader2,
  Pencil,
  Plus,
  Save,
  Settings,
  Trash2,
} from "lucide-react"
import { toast } from "sonner"

import { apiClient, type APIError } from "@/shared/api"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Skeleton } from "@/components/ui/skeleton"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import { formatDate } from "@/lib/format"
import type { EnvVar, Project } from "@/types"

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

function ProjectSettingsPage() {
  const { projectID } = useParams()
  const navigate = useNavigate()

  const [project, setProject] = useState<Project | null>(null)
  const [envVars, setEnvVars] = useState<EnvVar[]>([])
  const [loading, setLoading] = useState(true)

  const [form, setForm] = useState({
    name: "",
    prod_branch: "",
    root_dir: "",
    out_dir: "",
    install_cmd: "",
    build_cmd: "",
  })
  const [saving, setSaving] = useState(false)

  const [envOpen, setEnvOpen] = useState(false)
  const [editingKey, setEditingKey] = useState<string | null>(null)
  const [envKey, setEnvKey] = useState("")
  const [envValue, setEnvValue] = useState("")
  const [showValue, setShowValue] = useState(false)
  const [savingEnv, setSavingEnv] = useState(false)

  const [deleteOpen, setDeleteOpen] = useState(false)
  const [deleting, setDeleting] = useState(false)

  useEffect(() => {
    if (!projectID) return

    let active = true

    Promise.all([
      apiClient.get<Project>(`/api/projects/${projectID}`),
      apiClient.get<EnvVar[]>(`/api/projects/${projectID}/env-vars`),
    ])
      .then(([proj, vars]) => {
        if (!active) return
        setProject(proj)
        setEnvVars(vars)
        setForm({
          name: proj.name ?? "",
          prod_branch: proj.prod_branch ?? "",
          root_dir: proj.root_dir ?? "",
          out_dir: proj.out_dir ?? "",
          install_cmd: proj.install_cmd ?? "",
          build_cmd: proj.build_cmd ?? "",
        })
      })
      .catch((err: unknown) => {
        const e = err as APIError
        toast.error(`Failed to load settings (${e.code})`, {
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

  const refetchEnvVars = async () => {
    if (!projectID) return
    const vars = await apiClient.get<EnvVar[]>(
      `/api/projects/${projectID}/env-vars`
    )
    setEnvVars(vars)
  }

  const openAddEnvVar = () => {
    setEditingKey(null)
    setEnvKey("")
    setEnvValue("")
    setShowValue(false)
    setEnvOpen(true)
  }

  const openEditEnvVar = (v: EnvVar) => {
    setEditingKey(v.key)
    setEnvKey(v.key)
    setEnvValue("")
    setShowValue(false)
    setEnvOpen(true)
  }

  const handleSaveEnvVar = async () => {
    if (!projectID || !envKey.trim() || !envValue) return
    setSavingEnv(true)
    try {
      await apiClient.put(`/api/projects/${projectID}/env-vars`, {
        key: envKey.trim(),
        value: envValue,
      })
      toast.success(
        editingKey
          ? "Environment variable updated"
          : "Environment variable added"
      )
      setEnvOpen(false)
      await refetchEnvVars()
    } catch (err) {
      const e = err as APIError
      toast.error(
        `Failed to save environment variable (${e.code})`,
        { description: e.message }
      )
    } finally {
      setSavingEnv(false)
    }
  }

  const handleDeleteEnvVar = async (key: string) => {
    if (!projectID) return
    try {
      await apiClient.del(
        `/api/projects/${projectID}/env-vars/${encodeURIComponent(key)}`
      )
      toast.success("Environment variable deleted")
      await refetchEnvVars()
    } catch (err) {
      const e = err as APIError
      toast.error(
        `Failed to delete environment variable (${e.code})`,
        { description: e.message }
      )
    }
  }

  const handleSaveProject = async () => {
    if (!projectID) return
    if (!form.name.trim() || !form.prod_branch.trim()) {
      toast.error("Name and production branch are required")
      return
    }
    setSaving(true)
    try {
      await apiClient.put(`/api/projects/${projectID}`, {
        name: form.name.trim(),
        prod_branch: form.prod_branch.trim(),
        root_dir: form.root_dir.trim(),
        out_dir: form.out_dir.trim(),
        install_cmd: form.install_cmd.trim(),
        build_cmd: form.build_cmd.trim(),
      })
      const proj = await apiClient.get<Project>(`/api/projects/${projectID}`)
      setProject(proj)
      setForm({
        name: proj.name ?? "",
        prod_branch: proj.prod_branch ?? "",
        root_dir: proj.root_dir ?? "",
        out_dir: proj.out_dir ?? "",
        install_cmd: proj.install_cmd ?? "",
        build_cmd: proj.build_cmd ?? "",
      })
      toast.success("Project settings saved")
    } catch (err) {
      const e = err as APIError
      toast.error(`Failed to save project settings (${e.code})`, {
        description: e.message,
      })
    } finally {
      setSaving(false)
    }
  }

  const handleDeleteProject = async () => {
    if (!projectID) return
    setDeleting(true)
    try {
      await apiClient.del(`/api/projects/${projectID}`)
      toast.success("Project deleted")
      navigate("/dashboard")
    } catch (err) {
      const e = err as APIError
      toast.error(`Failed to delete project (${e.code})`, {
        description: e.message,
      })
      setDeleteOpen(false)
    } finally {
      setDeleting(false)
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
            render={<Link to={`/dashboard/${project.id}`} />}
          >
            <Settings data-icon="inline-start" />
            Overview
          </Button>
          <Button
            variant="outline"
            render={<Link to={`/dashboard/${project.id}/deployments`} />}
          >
            <History data-icon="inline-start" />
            Deployments
          </Button>
        </div>
      </div>

      <div className="flex flex-col gap-6">
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center justify-between gap-3">
              <span>Environment variables</span>
              <Button size="sm" onClick={openAddEnvVar}>
                <Plus data-icon="inline-start" />
                Add variable
              </Button>
            </CardTitle>
            <CardDescription>
              Injected into every build. Values are encrypted at rest and never
              shown again after saving.
            </CardDescription>
          </CardHeader>
          <CardContent>
            {envVars.length === 0 ? (
              <div className="flex flex-col items-center gap-3 rounded-lg border border-dashed py-12 text-center">
                <KeyRound className="size-6 text-muted-foreground" />
                <div>
                  <p className="font-medium">No environment variables yet</p>
                  <p className="text-sm text-muted-foreground">
                    Add secrets like API keys and connection strings.
                  </p>
                </div>
                <Button variant="outline" size="sm" onClick={openAddEnvVar}>
                  Add your first variable
                </Button>
              </div>
            ) : (
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Key</TableHead>
                    <TableHead>Value</TableHead>
                    <TableHead>Updated</TableHead>
                    <TableHead className="text-right">Actions</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {envVars.map((v) => (
                    <TableRow key={v.key}>
                      <TableCell className="font-medium">
                        <KeyRound className="mr-1.5 inline size-3.5 text-muted-foreground" />
                        {v.key}
                      </TableCell>
                      <TableCell className="font-mono text-xs tracking-widest text-muted-foreground">
                        {v.value}
                      </TableCell>
                      <TableCell className="text-xs text-muted-foreground">
                        {formatDate(v.updated_at)}
                      </TableCell>
                      <TableCell className="text-right">
                        <div className="flex justify-end gap-1">
                          <Button
                            size="icon-sm"
                            variant="ghost"
                            onClick={() => openEditEnvVar(v)}
                            aria-label={`Edit ${v.key}`}
                          >
                            <Pencil />
                          </Button>
                          <Button
                            size="icon-sm"
                            variant="ghost"
                            className="text-destructive hover:text-destructive"
                            onClick={() => handleDeleteEnvVar(v.key)}
                            aria-label={`Delete ${v.key}`}
                          >
                            <Trash2 />
                          </Button>
                        </div>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            )}
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Build configuration</CardTitle>
            <CardDescription>
              Controls how Slate builds and deploys this project.
            </CardDescription>
          </CardHeader>
          <CardContent className="flex flex-col gap-4">
            <div className="grid gap-4 sm:grid-cols-2">
              <Field label="Name">
                <Input
                  value={form.name}
                  onChange={(e) =>
                    setForm({ ...form, name: e.target.value })
                  }
                />
              </Field>
              <Field label="Production branch">
                <Input
                  value={form.prod_branch}
                  onChange={(e) =>
                    setForm({ ...form, prod_branch: e.target.value })
                  }
                />
              </Field>
            </div>
            <Field label="Root directory" hint="Where your app lives. Defaults to the repository root.">
              <Input
                value={form.root_dir}
                onChange={(e) =>
                  setForm({ ...form, root_dir: e.target.value })
                }
                placeholder="/"
              />
            </Field>
            <div className="grid gap-4 sm:grid-cols-3">
              <Field label="Output directory">
                <Input
                  value={form.out_dir}
                  onChange={(e) =>
                    setForm({ ...form, out_dir: e.target.value })
                  }
                  placeholder="auto-detected"
                />
              </Field>
              <Field label="Install command">
                <Input
                  value={form.install_cmd}
                  onChange={(e) =>
                    setForm({ ...form, install_cmd: e.target.value })
                  }
                  placeholder="auto-detected"
                />
              </Field>
              <Field label="Build command">
                <Input
                  value={form.build_cmd}
                  onChange={(e) =>
                    setForm({ ...form, build_cmd: e.target.value })
                  }
                  placeholder="auto-detected"
                />
              </Field>
            </div>
            <div className="flex justify-end">
              <Button onClick={handleSaveProject} disabled={saving}>
                {saving ? (
                  <Loader2 className="animate-spin" />
                ) : (
                  <Save data-icon="inline-start" />
                )}
                Save changes
              </Button>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Danger zone</CardTitle>
            <CardDescription>
              Deleting this project is permanent and cannot be undone.
            </CardDescription>
          </CardHeader>
          <CardContent className="flex items-center justify-between gap-4">
            <div>
              <p className="text-sm font-medium">Delete project</p>
              <p className="text-xs text-muted-foreground">
                Removes the project, all builds, and environment variables.
              </p>
            </div>
            <Button variant="destructive" onClick={() => setDeleteOpen(true)}>
              <Trash2 data-icon="inline-start" />
              Delete project
            </Button>
          </CardContent>
        </Card>
      </div>

      <Dialog open={envOpen} onOpenChange={setEnvOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>
              {editingKey ? "Edit variable" : "Add variable"}
            </DialogTitle>
            <DialogDescription>
              {editingKey
                ? `Update the value of ${editingKey}.`
                : "Add a variable to inject into every build."}
            </DialogDescription>
          </DialogHeader>
          <div className="flex flex-col gap-3">
            <label className="flex flex-col gap-1.5">
              <span className="text-sm font-medium">Key</span>
              <Input
                value={envKey}
                onChange={(e) => setEnvKey(e.target.value)}
                disabled={editingKey !== null}
                placeholder="DATABASE_URL"
              />
            </label>
            <label className="flex flex-col gap-1.5">
              <span className="text-sm font-medium">Value</span>
              <div className="relative">
                <Input
                  type={showValue ? "text" : "password"}
                  value={envValue}
                  onChange={(e) => setEnvValue(e.target.value)}
                  placeholder={
                    editingKey ? "Type a new value to overwrite" : "secret-value"
                  }
                  className="pr-9"
                />
                <button
                  type="button"
                  onClick={() => setShowValue((s) => !s)}
                  className="absolute inset-y-0 right-0 flex items-center px-2 text-muted-foreground transition-colors hover:text-foreground"
                  aria-label={showValue ? "Hide value" : "Show value"}
                >
                  {showValue ? (
                    <EyeOff className="size-4" />
                  ) : (
                    <Eye className="size-4" />
                  )}
                </button>
              </div>
            </label>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setEnvOpen(false)}>
              Cancel
            </Button>
            <Button
              onClick={handleSaveEnvVar}
              disabled={!envKey.trim() || !envValue || savingEnv}
            >
              {savingEnv ? <Loader2 className="animate-spin" /> : null}
              Save variable
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={deleteOpen} onOpenChange={setDeleteOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Delete project?</DialogTitle>
            <DialogDescription>
              This will permanently delete "{project.name}" along with all of
              its builds and environment variables. This action cannot be
              undone.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setDeleteOpen(false)}>
              Cancel
            </Button>
            <Button
              variant="destructive"
              onClick={handleDeleteProject}
              disabled={deleting}
            >
              {deleting ? <Loader2 className="animate-spin" /> : null}
              Delete project
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}

export default ProjectSettingsPage
