import { useEffect, useState } from "react"
import { toast } from "sonner"

import { apiClient, type APIError } from "@/shared/api"
import type { RepoBranchesResponse } from "@/types"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"

export function BranchSelector({
  fullName,
  value,
  onValueChange,
}: {
  fullName: string
  value: string
  onValueChange: (branch: string) => void
}) {
  const [branches, setBranches] = useState<string[] | null>(null)

  useEffect(() => {
    if (!fullName) return

    let active = true
    apiClient
      .get<RepoBranchesResponse>(
        `/api/repos/branches?fullName=${encodeURIComponent(fullName)}`
      )
      .then((data) => {
        if (active) setBranches(data.branches.map((branch) => branch.name))
      })
      .catch((err: unknown) => {
        const e = err as APIError
        if (active) setBranches([])
        toast.error(`Failed to load branches (${e.code})`, {
          description: e.message,
        })
      })

    return () => {
      active = false
    }
  }, [fullName])

  return (
    <Select
      value={value}
      onValueChange={(next) => onValueChange(next ?? "")}
    >
      <SelectTrigger className="w-full">
        <SelectValue
          placeholder={branches === null ? "Loading branches…" : "Select a branch"}
        />
      </SelectTrigger>
      <SelectContent>
        {branches === null ? (
          <p className="px-3 py-2 text-sm text-muted-foreground">Loading…</p>
        ) : branches.length === 0 ? (
          <p className="px-3 py-2 text-sm text-muted-foreground">No branches found</p>
        ) : (
          branches.map((branch) => (
            <SelectItem key={branch} value={branch}>
              {branch}
            </SelectItem>
          ))
        )}
      </SelectContent>
    </Select>
  )
}
