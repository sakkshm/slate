import type { BuildStatus } from "@/types"

export function shortSHA(sha: string, length = 7): string {
  if (!sha) return ""
  return sha.length > length ? sha.slice(0, length) : sha
}

export function formatDate(iso: string): string {
  if (!iso) return ""
  const date = new Date(iso)
  return date.toLocaleString(undefined, {
    month: "short",
    day: "numeric",
    year: "numeric",
    hour: "numeric",
    minute: "2-digit",
  })
}

export function formatDuration(ms: number): string {
  if (!ms) return "—"
  const totalSeconds = Math.floor(ms / 1000)
  if (totalSeconds < 1) return `${ms}ms`
  const minutes = Math.floor(totalSeconds / 60)
  const seconds = totalSeconds % 60
  if (minutes < 1) return `${seconds}s`
  return `${minutes}m ${seconds}s`
}

export interface StatusMeta {
  label: string
  dot: string
  badge: string
}

const statusMeta: Record<BuildStatus, StatusMeta> = {
  queued: {
    label: "Queued",
    dot: "bg-muted-foreground",
    badge:
      "border-border bg-muted text-muted-foreground",
  },
  building: {
    label: "Building",
    dot: "bg-blue-500 animate-pulse",
    badge: "border-blue-500/30 bg-blue-500/10 text-blue-600 dark:text-blue-400",
  },
  ready: {
    label: "Ready",
    dot: "bg-emerald-500",
    badge: "border-emerald-500/30 bg-emerald-500/10 text-emerald-600 dark:text-emerald-400",
  },
  failed: {
    label: "Failed",
    dot: "bg-red-500",
    badge: "border-red-500/30 bg-red-500/10 text-red-600 dark:text-red-400",
  },
  cancelled: {
    label: "Cancelled",
    dot: "bg-muted-foreground",
    badge: "border-border bg-muted text-muted-foreground",
  },
}

export function buildStatusMeta(status: BuildStatus): StatusMeta {
  return statusMeta[status] ?? statusMeta.queued
}
