<template>
  <div class="task-list">
    <!-- Create Task Area -->
    <div class="create-section">
      <div class="create-glow"></div>
      <div class="create-inner">
        <div class="create-title">
          <h2>New Scan</h2>
          <p>Enter targets to start XSS vulnerability scanning</p>
        </div>
        <div class="create-form">
          <div class="mode-switch">
            <button :class="['mode-btn', { active: scanMode === 'domain' }]" @click="scanMode = 'domain'">
              <el-icon><Globe /></el-icon>
              Domain Mode
            </button>
            <button :class="['mode-btn', { active: scanMode === 'url' }]" @click="scanMode = 'url'">
              <el-icon><Link /></el-icon>
              URL Mode
            </button>
          </div>
          <div class="mode-desc">
            <template v-if="scanMode === 'domain'">
              <span class="desc-icon">🔍</span>
              <span>Subfinder → Httpx → XScan full pipeline. Enter one domain per line. Each domain creates a separate task.</span>
            </template>
            <template v-else>
              <span class="desc-icon">⚡</span>
              <span>Direct XSS scan, skip subdomain &amp; probing. Enter one URL per line (must start with http/https). All URLs go into one task. Single URL uses <code>xscan spider -u</code>, multiple uses <code>xscan spider -f</code>.</span>
            </template>
          </div>
          <div class="input-area">
            <textarea
              v-model="targetInput"
              :placeholder="inputPlaceholder"
              rows="4"
              class="target-textarea"
              @keydown.ctrl.enter="handleCreate"
            ></textarea>
            <div class="input-footer">
              <span class="input-hint">{{ targetCount }} target{{ targetCount !== 1 ? 's' : '' }} · Ctrl+Enter to submit</span>
              <el-button type="primary" size="large" :loading="creating" @click="handleCreate" class="scan-btn">
                <el-icon><VideoPlay /></el-icon>
                Start Scan
              </el-button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Stats -->
    <div class="stats-row">
      <div class="stat-card">
        <div class="stat-icon total">
          <el-icon :size="20"><Tickets /></el-icon>
        </div>
        <div class="stat-info">
          <div class="stat-number">{{ tasks.length }}</div>
          <div class="stat-label">Total</div>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon running">
          <el-icon :size="20"><Loading /></el-icon>
        </div>
        <div class="stat-info">
          <div class="stat-number">{{ runningCount }}</div>
          <div class="stat-label">Running</div>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon completed">
          <el-icon :size="20"><CircleCheck /></el-icon>
        </div>
        <div class="stat-info">
          <div class="stat-number">{{ completedCount }}</div>
          <div class="stat-label">Done</div>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon danger">
          <el-icon :size="20"><Warning /></el-icon>
        </div>
        <div class="stat-info">
          <div class="stat-number">{{ totalXss }}</div>
          <div class="stat-label">XSS</div>
        </div>
      </div>
    </div>

    <!-- Task Table -->
    <el-card shadow="never" class="table-card">
      <template #header>
        <div class="card-header">
          <span class="card-title">Scan Tasks</span>
          <el-button text class="refresh-btn" @click="loadTasks">
            <el-icon><Refresh /></el-icon>
            Refresh
          </el-button>
        </div>
      </template>

      <el-table :data="tasks" v-loading="loading" empty-text="No tasks yet" stripe>
        <el-table-column prop="id" label="ID" width="80" />

        <el-table-column label="Mode" width="110" align="center">
          <template #default="{ row }">
            <span :class="['mode-badge', row.scan_mode === 'url' ? 'mode-url' : 'mode-domain']">
              {{ row.scan_mode === 'url' ? 'URL' : 'Domain' }}
            </span>
          </template>
        </el-table-column>

        <el-table-column prop="root_domain" label="Target" min-width="240">
          <template #default="{ row }">
            <span class="target-text">{{ row.root_domain }}</span>
          </template>
        </el-table-column>

        <el-table-column label="Status" width="170">
          <template #default="{ row }">
            <span :class="['status-badge', `status-${statusClass(row.status)}`]">
              <span class="status-dot"></span>
              {{ statusText(row.status) }}
            </span>
          </template>
        </el-table-column>

        <el-table-column label="Progress" min-width="200">
          <template #default="{ row }">
            <span class="step-text">{{ row.current_step || '-' }}</span>
          </template>
        </el-table-column>

        <el-table-column prop="subdomain_count" label="Sub" width="70" align="center" />
        <el-table-column prop="alive_count" label="Alive" width="70" align="center" />

        <el-table-column label="XSS" width="70" align="center">
          <template #default="{ row }">
            <span v-if="row.xss_count > 0" class="xss-badge">{{ row.xss_count }}</span>
            <span v-else class="text-muted">0</span>
          </template>
        </el-table-column>

        <el-table-column label="Created" width="160">
          <template #default="{ row }">
            <span class="time-text">{{ formatTime(row.created_at) }}</span>
          </template>
        </el-table-column>

        <el-table-column label="" width="140" fixed="right">
          <template #default="{ row }">
            <div class="action-btns">
              <button class="action-btn view" @click="viewDetail(row.id)" title="View">
                <el-icon><View /></el-icon>
              </button>
              <el-popconfirm title="Delete this task?" @confirm="handleDelete(row.id)">
                <template #reference>
                  <button class="action-btn delete" title="Delete">
                    <el-icon><Delete /></el-icon>
                  </button>
                </template>
              </el-popconfirm>
            </div>
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

const inputPlaceholder = computed(() =>
  scanMode.value === 'url'
    ? 'https://example.com/search?q=test\nhttps://example.com/page?id=1\nhttps://other.com/path'
    : 'example.com\nexample.org\ntest.com'
)

const targetCount = computed(() => {
  return targetInput.value.split('\n').filter(l => l.trim()).length
})

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
  } catch {
    ElMessage.error('Failed to load tasks')
  } finally {
    loading.value = false
  }
}

async function handleCreate() {
  const lines = targetInput.value.split('\n').map(l => l.trim()).filter(l => l)
  if (lines.length === 0) {
    ElMessage.warning(scanMode.value === 'url' ? 'Enter at least one URL' : 'Enter at least one domain')
    return
  }

  // Validate URLs in url mode
  if (scanMode.value === 'url') {
    for (const line of lines) {
      if (!/^https?:\/\//i.test(line)) {
        ElMessage.warning(`Invalid URL (must start with http/https): ${line}`)
        return
      }
    }
  }

  const payload = {
    mode: scanMode.value,
    targets: lines
  }

  creating.value = true
  try {
    await createTask(payload)
    if (scanMode.value === 'domain' && lines.length > 1) {
      ElMessage.success(`${lines.length} tasks created`)
    } else {
      ElMessage.success('Task created')
    }
    targetInput.value = ''
    await loadTasks()
  } catch (err) {
    ElMessage.error('Failed: ' + (err.response?.data?.error || err.message))
  } finally {
    creating.value = false
  }
}

async function handleDelete(id) {
  try {
    await deleteTask(id)
    ElMessage.success('Deleted')
    await loadTasks()
  } catch {
    ElMessage.error('Delete failed')
  }
}

function viewDetail(id) {
  router.push(`/task/${id}`)
}

function statusClass(status) {
  const map = {
    pending: 'pending',
    subdomain_collecting: 'running',
    httpx_probing: 'running',
    xss_scanning: 'running',
    completed: 'completed',
    failed: 'failed',
    cancelled: 'cancelled',
  }
  return map[status] || 'pending'
}

function statusText(status) {
  const map = {
    pending: 'Pending',
    subdomain_collecting: 'Subdomains',
    httpx_probing: 'Probing',
    xss_scanning: 'XSS Scan',
    completed: 'Completed',
    failed: 'Failed',
    cancelled: 'Cancelled',
  }
  return map[status] || status
}

function formatTime(time) {
  if (!time) return '-'
  return new Date(time).toLocaleString('en-US', { hour12: false, month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' })
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
  gap: 24px;
}

/* Create Section */
.create-section {
  position: relative;
  border-radius: 16px;
  overflow: hidden;
  background: var(--bg-card);
  border: 1px solid var(--border-color);
}

.create-glow {
  position: absolute;
  inset: 0;
  background: linear-gradient(135deg, rgba(0, 212, 255, 0.06) 0%, rgba(99, 102, 241, 0.06) 50%, rgba(239, 68, 68, 0.03) 100%);
  pointer-events: none;
}

.create-inner {
  position: relative;
  padding: 32px;
}

.create-title h2 {
  font-size: 22px;
  font-weight: 700;
  color: var(--text-primary);
  margin-bottom: 4px;
}

.create-title p {
  font-size: 14px;
  color: var(--text-muted);
  margin-bottom: 20px;
}

.create-form {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.mode-switch {
  display: flex;
  gap: 8px;
}

.mode-btn {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 10px 20px;
  border-radius: 8px;
  border: 1px solid var(--border-color);
  background: var(--bg-input);
  color: var(--text-muted);
  cursor: pointer;
  font-size: 13px;
  font-weight: 500;
  transition: all 0.2s;
}

.mode-btn:hover {
  border-color: var(--border-light);
  color: var(--text-secondary);
}

.mode-btn.active {
  border-color: var(--accent);
  color: var(--accent);
  background: var(--accent-glow);
}

.mode-desc {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  padding: 12px 16px;
  background: rgba(0, 212, 255, 0.04);
  border: 1px solid rgba(0, 212, 255, 0.1);
  border-radius: 10px;
  font-size: 13px;
  color: var(--text-secondary);
  line-height: 1.5;
}

.mode-desc .desc-icon {
  flex-shrink: 0;
  font-size: 16px;
}

.mode-desc code {
  background: rgba(0, 212, 255, 0.1);
  color: #7dd3fc;
  padding: 1px 5px;
  border-radius: 3px;
  font-size: 12px;
}

.input-area {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.target-textarea {
  width: 100%;
  min-height: 100px;
  padding: 14px 16px;
  background: var(--bg-input);
  border: 1px solid var(--border-color);
  border-radius: 10px;
  color: var(--text-primary);
  font-size: 14px;
  font-family: 'JetBrains Mono', monospace, sans-serif;
  line-height: 1.6;
  resize: vertical;
  outline: none;
  transition: border-color 0.2s;
}

.target-textarea::placeholder {
  color: var(--text-muted);
  opacity: 0.6;
}

.target-textarea:focus {
  border-color: var(--accent);
  box-shadow: 0 0 0 1px var(--accent);
}

.input-footer {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.input-hint {
  font-size: 12px;
  color: var(--text-muted);
}

.scan-btn {
  min-width: 140px;
  font-weight: 600;
  border-radius: 10px;
  background: linear-gradient(135deg, #00d4ff, #6366f1) !important;
  border: none !important;
}

/* Stats */
.stats-row {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 16px;
}

.stat-card {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 20px;
  background: var(--bg-card);
  border: 1px solid var(--border-color);
  border-radius: 12px;
  transition: border-color 0.2s;
}

.stat-card:hover {
  border-color: var(--border-light);
}

.stat-icon {
  width: 40px;
  height: 40px;
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.stat-icon.total { background: rgba(99, 102, 241, 0.15); color: #818cf8; }
.stat-icon.running { background: rgba(245, 158, 11, 0.15); color: #fbbf24; }
.stat-icon.completed { background: rgba(16, 185, 129, 0.15); color: #34d399; }
.stat-icon.danger { background: rgba(239, 68, 68, 0.15); color: #f87171; }

.stat-number {
  font-size: 24px;
  font-weight: 700;
  color: var(--text-primary);
  line-height: 1;
}

.stat-label {
  font-size: 12px;
  color: var(--text-muted);
  margin-top: 2px;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

/* Table card */
.table-card { overflow: hidden; }

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.card-title {
  font-size: 16px;
  font-weight: 600;
  color: var(--text-primary);
}

.refresh-btn { color: var(--text-muted) !important; }
.refresh-btn:hover { color: var(--accent) !important; }

/* Mode badge */
.mode-badge {
  display: inline-block;
  padding: 3px 10px;
  border-radius: 6px;
  font-size: 11px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.mode-domain { background: rgba(99, 102, 241, 0.15); color: #818cf8; }
.mode-url { background: rgba(16, 185, 129, 0.15); color: #34d399; }

.target-text {
  color: var(--text-primary);
  font-weight: 500;
  word-break: break-all;
}

/* Status badge */
.status-badge {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 4px 12px;
  border-radius: 20px;
  font-size: 12px;
  font-weight: 500;
}

.status-dot { width: 6px; height: 6px; border-radius: 50%; flex-shrink: 0; }
.status-pending { background: rgba(100,116,139,.15); color: #94a3b8; }
.status-pending .status-dot { background: #94a3b8; }
.status-running { background: rgba(245,158,11,.15); color: #fbbf24; }
.status-running .status-dot { background: #fbbf24; animation: pulse 1.5s infinite; }
.status-completed { background: rgba(16,185,129,.15); color: #34d399; }
.status-completed .status-dot { background: #34d399; }
.status-failed { background: rgba(239,68,68,.15); color: #f87171; }
.status-failed .status-dot { background: #f87171; }
.status-cancelled { background: rgba(100,116,139,.15); color: #64748b; }
.status-cancelled .status-dot { background: #64748b; }

@keyframes pulse { 0%,100%{opacity:1} 50%{opacity:.4} }

.step-text { font-size: 13px; color: var(--text-secondary); }

.xss-badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 22px;
  height: 22px;
  padding: 0 6px;
  border-radius: 6px;
  background: rgba(239,68,68,.2);
  color: #f87171;
  font-size: 12px;
  font-weight: 700;
}

.text-muted { color: var(--text-muted); }
.time-text { font-size: 13px; color: var(--text-muted); }

/* Action buttons */
.action-btns { display: flex; gap: 6px; }

.action-btn {
  width: 32px;
  height: 32px;
  border-radius: 8px;
  border: 1px solid var(--border-color);
  background: transparent;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s;
  font-size: 14px;
}

.action-btn.view { color: var(--accent); }
.action-btn.view:hover { background: var(--accent-glow); border-color: var(--accent); }
.action-btn.delete { color: var(--text-muted); }
.action-btn.delete:hover { color: #f87171; background: rgba(239,68,68,.1); border-color: rgba(239,68,68,.3); }

@media (max-width: 768px) {
  .stats-row { grid-template-columns: repeat(2, 1fr); }
  .input-footer { flex-direction: column; gap: 8px; align-items: flex-start; }
  .scan-btn { min-width: auto; width: 100%; }
}
</style>
