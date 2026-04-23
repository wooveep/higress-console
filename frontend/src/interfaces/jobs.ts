export interface JobRunRecord {
  id: number;
  jobName: string;
  triggerSource?: string;
  triggerId?: string;
  status: string;
  idempotencyKey?: string;
  targetVersion?: string;
  message?: string;
  errorText?: string;
  startedAt?: string;
  finishedAt?: string;
  durationMs?: number;
}

export interface JobSummary {
  name: string;
  description: string;
  schedule?: string;
  manualOnly: boolean;
  running: boolean;
  lastRun?: JobRunRecord | null;
}

export interface JobDetail extends JobSummary {
  recentRuns: JobRunRecord[];
}

export interface TriggerJobInput {
  source?: string;
  triggerId?: string;
}
