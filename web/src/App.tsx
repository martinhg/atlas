import { useEffect, useState } from "react";

interface VersionInfo {
  version: string;
  name: string;
}

function App() {
  const [info, setInfo] = useState<VersionInfo | null>(null);

  useEffect(() => {
    fetch("/api/v1/version")
      .then((r) => r.json())
      .then(setInfo)
      .catch(() => setInfo(null));
  }, []);

  return (
    <div className="min-h-screen bg-zinc-950 text-zinc-100 flex items-center justify-center">
      <div className="text-center space-y-6">
        <h1 className="text-5xl font-bold tracking-tight">AtlasOS</h1>
        <p className="text-zinc-400 text-lg">
          Engineering Intelligence Platform
        </p>
        {info && (
          <p className="text-zinc-500 text-sm font-mono">v{info.version}</p>
        )}
      </div>
    </div>
  );
}

export default App;
