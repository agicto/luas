// Luas API performance baseline. Each scenario drives a fixed arrival rate against one hot path and
// fails the run when its p95 latency or error rate leaves the budget. Run it through
// `make perf` (api/scripts/perf.sh), which prepares a disposable database and a release build.
import http from 'k6/http';
import { check, fail } from 'k6';

const baseURL = (__ENV.LUAS_PERF_BASE_URL || 'http://127.0.0.1:18080').replace(/\/$/, '');
const duration = __ENV.LUAS_PERF_DURATION || '30s';
const identifier = __ENV.LUAS_PERF_USER || 'user@example.com';
const password = __ENV.LUAS_PERF_PASSWORD || 'secret';
// Multiplies every p95 budget, for hardware slower than a CI runner. It never loosens error rates.
const budgetScale = Number(__ENV.LUAS_PERF_BUDGET_SCALE || '1');
if (!(budgetScale > 0)) throw new Error('LUAS_PERF_BUDGET_SCALE must be a positive number');

// Budgets are p95 milliseconds on a CI-class runner with PostgreSQL on the same host. Rates stay
// well below saturation so the numbers track per-request cost rather than queueing.
const budgets = {
  readiness: { rate: 50, p95: 50 },
  login: { rate: 5, p95: 300 },
  profile: { rate: 50, p95: 50 },
  api_keys: { rate: 30, p95: 75 },
  // Keyset history over the seeded large table. At this rate on half a million rows, keyset pages
  // measured p95 7 ms and the offset pages they replace 26 ms, so a reintroduced count fails here.
  audit_history: { rate: 20, p95: 20 },
};

const scenarios = {};
const thresholds = {
  http_req_failed: ['rate<0.01'],
  checks: ['rate>0.99'],
};
for (const [name, budget] of Object.entries(budgets)) {
  scenarios[name] = {
    executor: 'constant-arrival-rate',
    exec: name,
    rate: budget.rate,
    timeUnit: '1s',
    duration,
    preAllocatedVUs: Math.max(2, Math.ceil(budget.rate / 5)),
    maxVUs: budget.rate * 2,
  };
  thresholds[`http_req_duration{scenario:${name}}`] = [`p(95)<${budget.p95 * budgetScale}`];
}

export const options = {
  scenarios,
  thresholds,
  summaryTrendStats: ['avg', 'med', 'p(90)', 'p(95)', 'p(99)', 'max'],
};

function signIn() {
  const response = http.post(
    `${baseURL}/v1/login`,
    JSON.stringify({ username: identifier, password }),
    { headers: { 'Content-Type': 'application/json' }, tags: { name: 'POST /v1/login' } },
  );
  return response;
}

export function setup() {
  const response = signIn();
  if (response.status !== 200) {
    fail(`setup sign-in failed with HTTP ${response.status}: ${response.body}`);
  }
  return { token: response.json('data.access_token') };
}

function authorized(data, name) {
  return { headers: { Authorization: `Bearer ${data.token}` }, tags: { name } };
}

export function readiness() {
  const response = http.get(`${baseURL}/health/ready`, { tags: { name: 'GET /health/ready' } });
  check(response, { 'readiness 200': r => r.status === 200 });
}

export function login() {
  check(signIn(), { 'login 200': r => r.status === 200 });
}

export function profile(data) {
  const response = http.get(`${baseURL}/v1/users/profile`, authorized(data, 'GET /v1/users/profile'));
  check(response, { 'profile 200': r => r.status === 200 });
}

export function api_keys(data) {
  const response = http.get(`${baseURL}/v1/api-keys?page=1&per_page=20`, authorized(data, 'GET /v1/api-keys'));
  check(response, { 'api keys 200': r => r.status === 200 });
}

export function audit_history(data) {
  const first = http.get(`${baseURL}/v1/audit-logs?cursor=&per_page=20`, authorized(data, 'GET /v1/audit-logs (keyset)'));
  check(first, { 'audit history 200': r => r.status === 200 && r.json('meta.next_cursor') !== undefined });
  const next = first.json('meta.next_cursor');
  if (next) {
    const older = http.get(
      `${baseURL}/v1/audit-logs?cursor=${encodeURIComponent(next)}&per_page=20`,
      authorized(data, 'GET /v1/audit-logs (keyset)'),
    );
    check(older, { 'audit history next page 200': r => r.status === 200 });
  }
}
