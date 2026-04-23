import type { JobDetail, JobSummary, TriggerJobInput } from '@/interfaces/jobs';
import request from './request';

export const listJobs = () => {
  return request.get<any, JobSummary[]>('/internal/jobs');
};

export const getJobDetail = (name: string) => {
  return request.get<any, JobDetail>(`/internal/jobs/${name}`);
};

export const triggerJob = (name: string, payload: TriggerJobInput) => {
  return request.post<TriggerJobInput, JobDetail>(`/internal/jobs/${name}/trigger`, payload);
};
