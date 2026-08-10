import { Route, Routes, useParams } from "react-router-dom"
import { Toaster } from "@/components/ui/sonner"
import { useSessionGuard } from "@/hooks/useSessionGuard"
import { DashboardLayout } from "@/components/layout/DashboardLayout"
import { GitHubCallback } from "./pages/GitHubCallback"
import HomePage from "./pages/HomePage"
import NotFound from "./pages/NotFound"
import ProjectsPage from "./pages/ProjectsPage"
import NewProjectPage from "./pages/NewProjectPage"
import ProjectOverviewPage from "./pages/ProjectOverviewPage"
import ProjectSettingsPage from "./pages/ProjectSettingsPage"
import DeploymentsPage from "./pages/DeploymentsPage"
import BuildDetailPage from "./pages/BuildDetailPage"

function ProjectOverviewRoute() {
  const { projectID } = useParams()
  return <ProjectOverviewPage key={projectID} />
}

function ProjectSettingsRoute() {
  const { projectID } = useParams()
  return <ProjectSettingsPage key={projectID} />
}

function DeploymentsRoute() {
  const { projectID } = useParams()
  return <DeploymentsPage key={projectID} />
}

function BuildDetailRoute() {
  const { projectID, buildID } = useParams()
  return <BuildDetailPage key={`${projectID}-${buildID}`} />
}

export default function App() {
  useSessionGuard()

  return (
    <>
      <Routes>
        <Route path="/" element={<HomePage />} />
        <Route path="/auth/github/callback" element={<GitHubCallback />} />

        <Route path="/dashboard" element={<DashboardLayout />}>
          <Route index element={<ProjectsPage />} />
          <Route path="new" element={<NewProjectPage />} />
          <Route path=":projectID" element={<ProjectOverviewRoute />} />
          <Route path=":projectID/settings" element={<ProjectSettingsRoute />} />
          <Route path=":projectID/deployments" element={<DeploymentsRoute />} />
          <Route
            path=":projectID/deployments/:buildID"
            element={<BuildDetailRoute />}
          />
        </Route>

        <Route path="*" element={<NotFound />} />
      </Routes>
      <Toaster richColors position="bottom-right" />
    </>
  )
}
