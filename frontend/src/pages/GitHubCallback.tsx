import { useEffect, useState } from 'react';
import { useSearchParams, useNavigate } from 'react-router-dom';
import { apiClient, type APIError } from '@/shared/api';
import { toast } from 'sonner';
import { Button } from '@/components/ui/button';
import { IconBrandGithub } from '@tabler/icons-react';

export const GitHubCallback = () => {
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const [needsInstall, setNeedsInstall] = useState(false);  

  const code = searchParams.get('code');
  const state = searchParams.get('state');
  const invalidURL = !code || !state;

  useEffect(() => {
    if (!code || !state) {
      toast.error("Malformed Callback URL", {
            description: "The URL does not contain either code or state params..",
      });
      return;
    }

    apiClient.post('/api/auth/github/callback', { code, state })
      .then(() => {
        toast.success("GitHub Integration successful!");
        navigate('/dashboard');
      })
      .catch((err) => {
        const error = err as APIError;
        if (error.code === 'NO_INSTALLATION') {
          setNeedsInstall(true);
          toast.warning("GitHub App not installed", {
            description: "Please install the GitHub App to continue.",
          });
        } else {
          toast.error(`Authentication Failed (${error.code})`, {
            description: `${error.message} (Status: ${error.status})`,
          });
          navigate('/');
        }
      });
  }, [code, state, navigate]);

  const handleInstall = async () => {
    try {
      const data = await apiClient.get<{ url: string }>('/api/auth/github/install-url');
      window.location.href = data.url;
    } catch (err) {
      const error = err as APIError;
      toast.error(`Failed to get install URL: ${error.code}`);
    }
  };

  if (needsInstall) {
    return (
      <div className="flex h-screen items-center justify-center">
        <div className="flex flex-col items-center gap-4">
          <p className="text-sm text-muted-foreground">
            You need to install the GitHub App first.
          </p>
          <Button onClick={handleInstall}>
            <IconBrandGithub /> Install GitHub App
          </Button>
        </div>
      </div>
    );
  }

  else if(invalidURL){
    return (
      <div className="flex h-screen items-center justify-center">
        <div className="flex flex-col items-center gap-4">
          <p className="text-sm text-muted-foreground">
            Invalid Callback URL, retry to authenticate.
          </p>
        </div>
      </div>
    );
  }

  return (
    <div className="flex h-screen items-center justify-center">
      <p className="text-sm text-muted-foreground animate-pulse">
        Connecting your GitHub account...
      </p>
    </div>
  );
};