<script setup lang="ts">
import { onBeforeUnmount, onMounted, reactive, ref } from 'vue';
import { useRouter } from 'vue-router';
import PageSection from '@/components/common/PageSection.vue';
import StatusTag from '@/components/common/StatusTag.vue';
import type { JobDetail, JobSummary } from '@/interfaces/jobs';
import { showSuccess } from '@/lib/feedback';
import { getJobDetail, listJobs, triggerJob } from '@/services/jobs';
import { formatDateTimeDisplay } from '@/utils/time';

const router = useRouter();

const loading = ref(false);
const detailLoading = ref(false);
const triggering = ref(false);
const detailOpen = ref(false);
const jobs = ref<JobSummary[]>([]);
const selectedJob = ref<JobDetail | null>(null);
const triggerForm = reactive({
  source: 'manual',
  triggerId: '',
});

let pollTimer: ReturnType<typeof setInterval> | null = null;

async function load() {
  loading.value = true;
  try {
    jobs.value = await listJobs().catch(() => []);
  } finally {
    loading.value = false;
  }
}

async function openDetail(record: JobSummary) {
  detailOpen.value = true;
  detailLoading.value = true;
  try {
    selectedJob.value = await getJobDetail(record.name);
  } finally {
    detailLoading.value = false;
  }
}

function startPolling() {
  stopPolling();
  pollTimer = setInterval(() => {
    void load();
    if (detailOpen.value && selectedJob.value?.name) {
      void openDetail(selectedJob.value);
    }
  }, 5000);
}

function stopPolling() {
  if (pollTimer) {
    clearInterval(pollTimer);
    pollTimer = null;
  }
}

async function submitTrigger() {
  if (!selectedJob.value) {
    return;
  }
  triggering.value = true;
  try {
    selectedJob.value = await triggerJob(selectedJob.value.name, {
      source: triggerForm.source.trim() || 'manual',
      triggerId: triggerForm.triggerId.trim() || undefined,
    });
    triggerForm.triggerId = '';
    await load();
    showSuccess('任务已触发');
  } finally {
    triggering.value = false;
  }
}

onMounted(() => {
  void load();
  startPolling();
});

onBeforeUnmount(() => {
  stopPolling();
});
</script>

<template>
  <div class="system-jobs-page">
    <PageSection title="Jobs 运维">
      <template #actions>
        <a-button @click="router.push('/system')">返回系统设置</a-button>
        <a-button @click="load">刷新</a-button>
      </template>

      <a-table :data-source="jobs" :loading="loading" row-key="name" :scroll="{ x: 1080 }">
        <a-table-column key="name" data-index="name" title="Job" width="260" />
        <a-table-column key="description" data-index="description" title="描述" />
        <a-table-column key="schedule" data-index="schedule" title="Cron" width="180" />
        <a-table-column key="manualOnly" title="Manual Only" width="120">
          <template #default="{ record }">{{ record.manualOnly ? '是' : '否' }}</template>
        </a-table-column>
        <a-table-column key="running" title="运行中" width="120">
          <template #default="{ record }">
            <StatusTag :value="record.running ? 'running' : 'idle'" :text="record.running ? '运行中' : '空闲'" />
          </template>
        </a-table-column>
        <a-table-column key="lastRunStatus" title="最近状态" width="140">
          <template #default="{ record }">
            <StatusTag :value="record.lastRun?.status || 'unknown'" :text="record.lastRun?.status || '-'" />
          </template>
        </a-table-column>
        <a-table-column key="lastRunAt" title="最近完成时间" width="180">
          <template #default="{ record }">{{ formatDateTimeDisplay(record.lastRun?.finishedAt || record.lastRun?.startedAt) }}</template>
        </a-table-column>
        <a-table-column key="actions" title="操作" width="120" fixed="right">
          <template #default="{ record }">
            <a-button type="link" size="small" @click="openDetail(record)">详情</a-button>
          </template>
        </a-table-column>
      </a-table>
    </PageSection>

    <a-drawer v-model:open="detailOpen" width="860" title="Job 详情" destroy-on-close>
      <a-skeleton :loading="detailLoading" active>
        <a-empty v-if="!selectedJob" description="暂无 Job 详情" />
        <div v-else class="system-jobs-page__detail">
          <a-descriptions bordered size="small" :column="2">
            <a-descriptions-item label="Job">{{ selectedJob.name }}</a-descriptions-item>
            <a-descriptions-item label="描述">{{ selectedJob.description || '-' }}</a-descriptions-item>
            <a-descriptions-item label="Cron">{{ selectedJob.schedule || '-' }}</a-descriptions-item>
            <a-descriptions-item label="Manual Only">{{ selectedJob.manualOnly ? '是' : '否' }}</a-descriptions-item>
            <a-descriptions-item label="运行中">
              <StatusTag :value="selectedJob.running ? 'running' : 'idle'" :text="selectedJob.running ? '运行中' : '空闲'" />
            </a-descriptions-item>
            <a-descriptions-item label="最近状态">
              <StatusTag :value="selectedJob.lastRun?.status || 'unknown'" :text="selectedJob.lastRun?.status || '-'" />
            </a-descriptions-item>
            <a-descriptions-item label="最近消息" :span="2">{{ selectedJob.lastRun?.message || '-' }}</a-descriptions-item>
            <a-descriptions-item label="最近错误" :span="2">{{ selectedJob.lastRun?.errorText || '-' }}</a-descriptions-item>
          </a-descriptions>

          <PageSection title="手动触发" subtle>
            <a-form layout="vertical">
              <a-form-item label="Source">
                <a-input v-model:value="triggerForm.source" />
              </a-form-item>
              <a-form-item label="Trigger ID">
                <a-input v-model:value="triggerForm.triggerId" placeholder="留空则由后端生成" />
              </a-form-item>
              <div class="system-jobs-page__actions">
                <a-button type="primary" :loading="triggering" @click="submitTrigger">触发任务</a-button>
              </div>
            </a-form>
          </PageSection>

          <PageSection title="最近运行记录" subtle>
            <a-table :data-source="selectedJob.recentRuns || []" row-key="id" size="small" :pagination="false" :scroll="{ x: 1200 }">
              <a-table-column key="id" data-index="id" title="ID" width="80" />
              <a-table-column key="triggerSource" data-index="triggerSource" title="Source" width="120" />
              <a-table-column key="triggerId" data-index="triggerId" title="Trigger ID" width="180" />
              <a-table-column key="status" title="状态" width="120">
                <template #default="{ record }">
                  <StatusTag :value="record.status" :text="record.status" />
                </template>
              </a-table-column>
              <a-table-column key="targetVersion" data-index="targetVersion" title="Target Version" width="180" />
              <a-table-column key="durationMs" data-index="durationMs" title="耗时(ms)" width="120" />
              <a-table-column key="startedAt" title="开始时间" width="180">
                <template #default="{ record }">{{ formatDateTimeDisplay(record.startedAt) }}</template>
              </a-table-column>
              <a-table-column key="finishedAt" title="结束时间" width="180">
                <template #default="{ record }">{{ formatDateTimeDisplay(record.finishedAt) }}</template>
              </a-table-column>
              <a-table-column key="message" data-index="message" title="消息" />
            </a-table>
          </PageSection>
        </div>
      </a-skeleton>
    </a-drawer>
  </div>
</template>

<style scoped>
.system-jobs-page {
  display: grid;
  gap: 18px;
}

.system-jobs-page__detail {
  display: grid;
  gap: 18px;
}

.system-jobs-page__actions {
  display: flex;
  justify-content: flex-end;
}
</style>
