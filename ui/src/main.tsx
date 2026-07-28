import React, { useEffect } from 'react';
import { createRoot } from 'react-dom/client';
import { QueryClientProvider } from '@tanstack/react-query';
import { RouterProvider } from '@tanstack/react-router';
import { createAppQueryClient } from './query-client';
import { connectRuntimeEvents } from './runtime-events';
import { router } from './router';
import './styles.css';

const queryClient = createAppQueryClient();

function RuntimeEventBridge() {
  useEffect(() => connectRuntimeEvents(queryClient), []);
  return null;
}

createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <QueryClientProvider client={queryClient}>
      <RuntimeEventBridge />
      <RouterProvider router={router} />
    </QueryClientProvider>
  </React.StrictMode>
);
