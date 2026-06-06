import { SharedArray } from 'k6/data';
import exec from 'k6/execution';

// Real public repos — small, stable, always pass GitHub existence check.
// Using SharedArray so the array is built once and shared across all VUs.
const REPOS = new SharedArray('repos', () => [
  'golang/go',
  'k6io/k6',
  'grafana/grafana',
  'prometheus/prometheus',
  'chi-middleware/chi',
  'jackc/pgx',
  'go-chi/chi',
  'google/uuid',
  'stretchr/testify',
  'spf13/viper',
]);

// Unique email per VU per iteration — prevents 409 conflicts between parallel VUs.
// Format: k6-{vuId}-{iterationId}-{timestamp}@loadtest.local
export function uniqueEmail() {
  const vu = exec.vu.idInTest;
  const iter = exec.scenario.iterationInTest;
  return `k6-${vu}-${iter}-${Date.now()}@loadtest.local`;
}

// Round-robin repo selection — each VU gets a different repo so GitHub
// rate limit hits are spread across repos instead of hammering one.
export function randomRepo() {
  const vu = exec.vu.idInTest;
  const iter = exec.scenario.iterationInTest;
  return REPOS[(vu + iter) % REPOS.length];
}
