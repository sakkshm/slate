export interface User {
  id: number
  username: string
  name: string
  email: string
  avatar_url: string
  github_installation_id: number
  created_at: string
  updated_at: string
}

export interface GithubInstallationRepo {
  id: number
  name: string
  full_name: string
  private: boolean
  html_url: string
  default_branch: string
}

export interface GithubInstallationReposResponse {
  total_count: number
  repositories: GithubInstallationRepo[]
}

export interface GithubRepoBranch {
  name: string
  commit: {
    sha: string
  }
  protected: boolean
}

export interface RepoBranchesResponse {
  branches: GithubRepoBranch[]
}

export interface GithubRepoContentEntry {
  name: string
  path: string
  sha: string
  size: number
  type: string
  html_url: string
  download_url: string
}

export interface RepoContentsResponse {
  entries: GithubRepoContentEntry[]
}

export interface Project {
  id: string
  owner_id: number
  name: string
  slug: string
  repo_id: number
  repo_url: string
  repo_name: string
  prod_branch: string
  framework: string
  root_dir: string
  install_cmd: string
  build_cmd: string
  out_dir: string
  created_at: string
  updated_at: string
}

export type BuildStatus = "queued" | "building" | "ready" | "failed" | "cancelled"

export interface Build {
  id: string
  project_id: string
  commit_sha: string
  commit_message: string
  status: BuildStatus
  duration: number
  log_location: string
  log_content: string
  asset_location: string
  created_at: string
}

export interface BuildDetail extends Build {
  deployment_url: string
  asset_url: string
}

export interface ListBuildsResponse {
  builds: Build[]
  total: number
}

export interface TriggerBuildResponse {
  build_id: string
  status: string
}

export interface CancelBuildResponse {
  build_id: string
  status: string
}

export interface EnvVar {
  key: string
  value: string
  updated_at: string
}

export interface CreateProjectRequest {
  name: string
  repo_url: string
  repo_id: number
  repo_name: string
  full_name: string
  prod_branch: string
  framework?: string
  root_dir?: string
  install_cmd?: string
  build_cmd?: string
  out_dir?: string
}

export interface UpdateProjectRequest {
  name?: string | null
  prod_branch?: string | null
  framework?: string | null
  root_dir?: string | null
  install_cmd?: string | null
  build_cmd?: string | null
  out_dir?: string | null
}

export interface AuthRedirectResponse {
  url: string
}
