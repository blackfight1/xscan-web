<template>
  <div class="task-list">
    <el-card class="create-card" shadow="hover">
      <div class="create-form">
        <el-select v-model="scanMode" size="large" class="mode-select">
          <el-option label="Domain mode" value="domain" />
          <el-option label="Single URL mode" value="url" />
        </el-select>

        <el-input
          v-model="targetInput"
          :placeholder="inputPlaceholder"
          size="large"
          clearable
          @keyup.enter="handleCreate"
        >
          <template #prepend>
            <el-icon><Search /></el-icon>
          </template>
        </el-input>

        <el-button type="primary" size="large" :loading="creating" @click="handleCreate">
          <el-icon><VideoPlay /></el-icon>
          Start
        </el-button>
      </div>
    </el-card>

    <div class="stats-row">
      <el-card shadow="hover" class="stat-card">
        <div class="stat-content">
          <div class="stat-number">{{ tasks.length }}</div>
          <div class="stat-label">Total Tasks</div>
        </div>
      </el-card>

      <el-card shadow="hover" class="stat-card">
        <div class="stat-content">
          <div class="stat-number running">{{ runningCount }}</div>
          <div class="stat-label">Running</div>
        </div>
      </el-card>

      <el-card shadow="hover" class="stat-card">
        <div class="stat-content">
          <div class="stat-number success">{{ completedCount }}</div>
          <div class="stat-label">Completed</div>
        </div>
      </el-card>

      <el-card shadow="hover" class="stat-card">
        <div class="stat-content">
          <div class="stat-number danger">{{ totalXss }}</div>
          <div class="stat-label">XSS Found</div>
        </div>
      </el-card>
    </div>

    <el-card shadow="hover">
      <template #header>
        <div class="card-header">
          <span>Scan Tasks</span>
          <el-button text @click="loadTasks">
            <el-icon><Refresh /></el-icon>
            Refresh
          </el-button>
        </div>
      </template>

      <el-table :data="tasks" stripe v-loading="loading" empty-text="No tasks">
        <el-table-column prop="id" label="ID" width="100" />

        <el-table-column label="Mode" width="130" align="center">
          <template #default="{ row }">
            <el-tag :type="row.scan_mode === 'url' ? 'success' : 'info'" effect="light">
              {{ row.scan_mode === 'url' ? 'URL' : 'Domain' }}
            </el-tag>
          </template>
        </el-table-column>

        <el-table-column prop="root_domain" label="Target" min-width="240" />

        <el-table-column label="Status" width="160">
          <template #default="{ row }">
            <el-tag :type="statusType(row.status)" effect="light">
              {{ statusText(row.status) }}
            </el-tag>
          </template>
        </el-table-column>

        <el-table-column label="Step" min-width="220">
          <template #default="{ row }">
            <span class="step-text">{{ row.current_step }}</span>
          </template>
        </el-table-column>

        <el-table-column prop="subdomain_count" label="Sub" width="80" align="center" />
        <el-table-column prop="alive_count" label="Alive" width="80" align="center" />

        <el-table-column label="XSS" width="80" align="center">
          <template #default="{ row }">
            <el-tag v-if="row.xss_count > 0" type="danger" effect="dark" size="small">
              {{ row.xss_count }}
            </el-tag>
            <span v-else>0</span>
          </template>
        </el-table-column>

        <el-table-column label="Created" width="170">
          <template #default="{ row }">
            {{ formatTime(row.created_at) }}
          </template>
        </el-table-column>

        <el-table-column label="Actions" width="150" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" @click="viewDetail(row.id)">
              <el-icon><View /></el-icon>
              Detail
            </el-button>

            <el-popconfirm title="Delete this task?" @confirm="handleDelete(row.id)">
              <template #reference>
                <el-button link type="danger">
                  <el-icon><Delete /></el-icon>
                  Delete
                </el-button>
              </template>
            </el-popconfirm>
          </template>
        </el-table-column>
      </el-table>
    </el-card>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { getTasks, createTask, deleteTask } from '../api'
import { ElMessage } from 'element-plus'

const router = useRouter()
const tasks = ref([])
const loading = ref(false)
const creating = ref(false)
const scanMode = ref('domain')
const targetInput = ref('')
let refreshTimer = null

const inputPlaceholder = computed(() => {
  if (scanMode.value === 'url') {
    return 'Enter a full URL, e.g. https://example.com/search?q=1'
  }
  return 'Enter root domain, e.g. example.com'
})

const runningCount = computed(() =>
  tasks.value.filter((t) =>
    ['pending', 'subdomain_collecting', 'httpx_probing', 'xss_scanning'].includes(t.status)
  ).length
)

const completedCount = computed(() => tasks.value.filter((t) => t.status === 'completed').length)

const totalXss = computed(() => tasks.value.reduce((sum, t) => sum + (t.xss_count || 0), 0))

async function loadTasks() {
  loading.value = true
  try {
    const data = await getTasks()
    tasks.value = data.tasks || []
  } catch {
    ElMessage.error('Failed to load tasks')
  } finally {
    loading.value = false
  }
}

async function handleCreate() {
  const value = targetInput.value.trim()
  if (!value) {
    ElMessage.warning(scanMode.value === 'url' ? 'Please enter URL' : 'Please enter root domain')
    return
  }

  if (scanMode.value === 'url' && !/^https?:\/\//i.test(value)) {
    ElMessage.warning('URL must start with http:// or https://')
    return
  }

  const payload =
    scanMode.value === 'url'
      ? { mode: 'url', target_url: value }
      : { mode: 'domain', root_domain: value }

  creating.value = true
  try {
    await createTask(payload)
    ElMessage.success('Task created')
    targetInput.value = ''
    await loadTasks()
  } catch (err) {
    ElMessage.error('Create task failed: ' + (err.response?.data?.error || err.message))
  } finally {
    creating.value = false
  }
}

async function handleDelete(id) {
  try {
    await deleteTask(id)
    ElMessage.success('Task deleted')
    await loadTasks()
  } catch {
    ElMessage.error('Delete failed')
  }
}

function viewDetail(id) {
  router.push(`/task/${id}`)
}

function statusType(status) {
  const map = {
    pending: 'info',
    subdomain_collecting: 'warning',
    httpx_probing: 'warning',
    xss_scanning: 'warning',
    completed: 'success',
    failed: 'danger',
    cancelled: 'info',
  }
  return map[status] || 'info'
}

function statusText(status) {
  const map = {
    pending: 'Pending',
    subdomain_collecting: 'Collecting subdomains',
    httpx_probing: 'Probing alive URLs',
    xss_scanning: 'Scanning XSS',
    completed: 'Completed',
    failed: 'Failed',
    cancelled: 'Cancelled',
  }
  return map[status] || status
}

function formatTime(time) {
  if (!time) return '-'
  return new Date(time).toLocaleString('zh-CN')
}

onMounted(() => {
  loadTasks()
  refreshTimer = setInterval(loadTasks, 10000)
})

onUnmounted(() => {
  if (refreshTimer) clearInterval(refreshTimer)
})
</script>

<style scoped>
.task-list {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.create-card {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
}

.create-card :deep(.el-card__body) {
  padding: 24px;
}

.create-form {
  display: flex;
  gap: 12px;
}

.mode-select {
  width: 180px;
}

.create-form .el-input {
  flex: 1;
}

.stats-row {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 16px;
}

.stat-content {
  text-align: center;
  padding: 8px 0;
}

.stat-number {
  font-size: 32px;
  font-weight: 700;
  color: #303133;
}

.stat-number.running {
  color: #e6a23c;
}

.stat-number.success {
  color: #67c23a;
}

.stat-number.danger {
  color: #f56c6c;
}

.stat-label {
  font-size: 14px;
  color: #909399;
  margin-top: 4px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-weight: 600;
}

.step-text {
  font-size: 13px;
  color: #606266;
}

@media (max-width: 768px) {
  .stats-row {
    grid-template-columns: repeat(2, 1fr);
  }

  .create-form {
    flex-direction: column;
  }

  .mode-select {
    width: 100%;
  }
}
</style>
