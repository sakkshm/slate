import { useEffect } from 'react';
import { useSearchParams, useNavigate } from 'react-router-dom';
import { apiClient } from '@/shared/api';
import { toast } from 'sonner';

export const GitHubCallback = () => {
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();

  useEffect(() => {
    const code = searchParams.get('code');
    const installationId = searchParams.get('installation_id');
    const state = searchParams.get('state');

    if (code && installationId) {
      apiClient.post('/api/auth/github/callback', { code, installationId, state })
        .then(() => {
          toast.success("GitHub Integration successful!");
          navigate('/dashboard'); 
        })
        .catch((err: any) => {
          // Gracefully toast the structured error thrown out of your apiClient
          toast.error(`Authentication Failed (${err.code})`, {
            description: `${err.message} (Status: ${err.status})`,
          });
          navigate('/login'); // Redirect to allow the user to retry
        });
    }
  }, [searchParams, navigate]);

  return (
    <div className="flex h-screen items-center justify-center">
      <p className="text-sm text-muted-foreground animate-pulse">
        Connecting your GitHub account and workspaces...
      </p>
    </div>
  );
};