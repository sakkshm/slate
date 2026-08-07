import { useEffect, useState } from "react"
import { apiClient } from "@/shared/api"
import type { User } from "@/types"

export interface UseAuthResult {
  user: User | null
  loading: boolean
  authed: boolean
}

export function useAuth(): UseAuthResult {
  const [user, setUser] = useState<User | null>(null)
  const [loading, setLoading] = useState(true)
  const [authed, setAuthed] = useState(false)

  useEffect(() => {
    let active = true

    apiClient
      .get<User>("/api/user")
      .then((profile) => {
        if (!active) return
        setUser(profile)
        setAuthed(true)
      })
      .catch(() => {
        if (active) setAuthed(false)
      })
      .finally(() => {
        if (active) setLoading(false)
      })

    return () => {
      active = false
    }
  }, [])

  return { user, loading, authed }
}
