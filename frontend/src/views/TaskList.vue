<template>
  <div class="task-list">
    <!-- Create Task Card -->
    <el-card class="create-card" shadow="hover">
      <div class="create-form">
        <el-input
          v-model="newDomain"
          placeholder="输入根域名，例如 example.com"
          size="large"
          clearable
          @keyup.enter="handleCreate"
        >
          <template #prepend>
            <el-icon><Search /></el-icon>
          </template>
        </el-input>
        <el-button type="primary" size="large" @click="handleCreate" :loading="creating">
          <el-icon><VideoPlay /></el-icon>
          开始扫描
        </el-button>
      </div>
    </el-card>

    <!-- Stats -->
    <div class="stats-row">
      <el-card shadow="hover" class="stat-card">
        <div class="stat-content">
          <div class="stat-number">{{ tasks.length }}</div>
          <div class="stat-label">总任务数</div>
        </div>
      </el-card>
      <el-card shadow="hover" class="stat-card">
        <div class="stat-content">
          <div class="stat-number running">{{ runningCount }}</div>
          <div class="stat-label">运行中</div>
        </div>
      </el-card>
      <el-card shadow="hover" class="stat-card">
        <div class="stat-content">
          <div class="stat-number success">{{ completedCount }}</div>
          <div class="stat-label">已完成</div>
        </div>
      </el-card>
      <el-card shadow="hover" class="stat-card">
        <div class="stat-content">
          <div class="stat-number danger">{{ totalXss }}</div>
          <div class="stat-label">发现XSS</div>
        </div>
      </el-card>
    </div>

    <!-- Task Table -->
    <el-card shadow="hover">
      <template #header>
        <div class="card-header">
          <span>扫描任务列表</span>
          <el-button text @click="loadTasks">
            <el-icon><Refresh /></el-icon>
            刷新
          </el-button>
        </div>
      </template>

      <el-table :data="tasks" stripe v-loading="loading" empty-text="暂无任务">
        <el-table-column prop="id" label="ID" width="100" />
        <el-table-column prop="root_domain" label="根域名" min-width="180" />
        <el-table-column label="状态" width="160">
          <template #default="{ row }">
            <el-tag :type="statusType(row.status)" effect="light">
              {{ statusText(row.status) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="进度" min-width="200">
          <template #default="{ row }">
            <span class="step-text">{{ row.current_step }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="subdomain_count" label="子域名" width="90" align="center" />
        <el-table-column prop="alive_count" label="存活" width="80" align="center" />
        <el-table-column label="XSS" width="80" align="center">
          <template #default="{ row }">
            <el-tag v-if="row.xss_count > 0" type="danger" effect="dark" size="small">
              {{ row.xss_count }}
            </el-tag>
            <span v-else>0</span>
          </template>
        </el-table-column>
        <el-table-column label="创建时间" width="170">
          <template #default="{ row }">
            {{ formatTime(row.created_at) }}
          </template>
        </el-table-column>
        <el-table-column label="操作" width="150" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" @click="viewDetail(row.id)">
              <el-icon><View /></el-icon>
              详情
            </el-button>
            <el-popconfirm title="确定删除此任务？" @confirm="handleDelete(row.id)">
              <template #reference>
                <el-button link type="danger">
                  <el-icon><Delete /></el-icon>
                  删除
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
const newDomain = ref('')
let refreshTimer = null

const runningCount = computed(() =>
  tasks.value.filter(t =>
    ['pending', 'subdomain_collecting', 'httpx_probing', 'xss_scanning'].includes(t.status)
  ).length
)

const completedCount = computed(() =>
  tasks.value.filter(t => t.status === 'completed').length
)

const totalXss = computed(() =>
  tasks.value.reduce((sum, t) => sum + (t.xss_count || 0), 0)
)

async function loadTasks() {
  loading.value = true
  try {
    const data = await getTasks()
    tasks.value = data.tasks || []
  } catch (err) {
    ElMessage.error('获取任务列表失败')
  } finally {
    loading.value = false
  }
}

async function handleCreate() {
  const domain = newDomain.value.trim()
  if (!domain) {
    ElMessage.warning('请输入根域名')
    return
  }

  creating.value = true
  try {
    await createTask(domain)
    ElMessage.success('任务创建成功')
    newDomain.value = ''
    await loadTasks()
  } catch (err) {
    ElMessage.error('创建任务失败: ' + (err.response?.data?.error || err.message))
  } finally {
    creating.value = false
  }
}

async function handleDelete(id) {
  try {
    await deleteTask(id)
    ElMessage.success('任务删除成功')
    await loadTasks()
  } catch (err) {
    ElMessage.error('删除失败')
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
    pending: '等待中',
    subdomain_collecting: '收集子域名',
    httpx_probing: '探测存活',
    xss_scanning: 'XSS扫描中',
    completed: '已完成',
    failed: '失败',
    cancelled: '已取消',
  }
  return map[status] || status
}

function formatTime(time) {
  if (!time) return '-'
  return new Date(time).toLocaleString('zh-CN')
}

onMounted(() => {
  loadTasks()
  // Auto refresh every 10 seconds
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
}
</style>
