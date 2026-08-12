import { ref } from 'vue'
import type { CallLogRecord, JobRunRecord } from '@/types'

export type JobDetail = {
  job: JobRunRecord
  calls: CallLogRecord[]
}

type JobDetailDrawerOptions = {
  loadDetail: (jobId: number) => Promise<JobDetail>
  onError: (error: unknown) => void
}

export function useJobDetailDrawer({ loadDetail, onError }: JobDetailDrawerOptions) {
  const jobDrawerVisible = ref(false)
  const jobDetailLoading = ref(false)
  const jobDetail = ref<JobDetail | null>(null)
  const callDrawerVisible = ref(false)
  const selectedCall = ref<CallLogRecord | null>(null)

  async function openJobDetail(jobId: number) {
    callDrawerVisible.value = false
    jobDrawerVisible.value = true
    jobDetailLoading.value = true
    try {
      jobDetail.value = await loadDetail(jobId)
    } catch (error) {
      onError(error)
      jobDetail.value = null
    } finally {
      jobDetailLoading.value = false
    }
  }

  function openCallDetail(call: CallLogRecord) {
    selectedCall.value = call
    callDrawerVisible.value = true
  }

  function clearJobDetail() {
    jobDrawerVisible.value = false
    jobDetail.value = null
  }

  return {
    callDrawerVisible,
    clearJobDetail,
    jobDetail,
    jobDetailLoading,
    jobDrawerVisible,
    openCallDetail,
    openJobDetail,
    selectedCall
  }
}
