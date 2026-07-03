import type { TrendsMessages } from './zh-Hans';

const messages: TrendsMessages = {
  title: 'Trend Pipeline',
  description: 'daily.dev trend sync, AI pre-filtering, and article topic intake',
  syncNow: 'Sync now',
  refreshing: 'Syncing',
  searchPlaceholder: 'Search title, summary, or channel',
  source: 'Source',
  lastPolled: 'Last polled',
  interval: 'Every {count} min',
  noSource: 'No source configured yet',
  tabs: {
    all: 'All',
    candidate: 'Candidates',
    new: 'New',
    selected: 'Selected',
    rejected: 'Rejected',
  },
  stats: {
    total: 'Total trends',
    candidate: 'Candidates',
    new: 'Watching',
    queued: 'Score queue',
  },
  table: {
    topic: 'Trend',
    channel: 'Channel',
    score: 'Score',
    audience: 'Audience',
    status: 'Status',
    time: 'Time',
    action: 'Link',
  },
  score: {
    h: 'Heat',
    k: 'Knowledge',
    r: 'Relevance',
    brand: 'Brand',
    risk: 'Risk',
    total: 'Total',
  },
  status: {
    new: 'New',
    candidate: 'Candidate',
    selected: 'Selected',
    rejected: 'Rejected',
    archived: 'Archived',
  },
  empty: {
    title: 'No matching trends',
    description: 'Change the filters or trigger a fresh daily.dev sync.',
  },
  errors: {
    load: 'Failed to load trend data',
    sync: 'Sync failed',
  },
  toast: {
    syncSuccess: 'Sync complete: {inserted} inserted, {candidates} candidates',
  },
};

export default messages;
