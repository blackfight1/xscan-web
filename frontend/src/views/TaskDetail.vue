<template>
  <div class="task-detail" v-loading="loading">
    <!-- Back Button -->
    <el-button @click="$router.push('/')" class="back-btn">
      <el-icon><ArrowLeft /></el-icon>
      返回列表
    </el-button>

    <template v-if="task">
      <!-- Task Info Card -->
      <el-card shadow="hover" class="info-card">
        <div class="task-header">
          <div class="task-title">
            <h2>{{ task.root_domain }}</h2>
            <el-tag :type="statusType(task.status)" size="large" effect="light">
              {{ statusText(task.status) }}
            </el-tag>
          </div>
          <div class="task-meta">
            <span>ID: {{ task.id }}</span>
            <el-divider direction="vertical" />
            <span>创建: {{ formatTime(task.created_at) }}</span>
            <el-divider direction="vertical" v-if="task.finished_at" />
            <span v-if="task.finished_at">完成: {{ formatTime(task.finished_at) }}</span>
          </div>
        </div>

        <!-- Progress -->
        <div class="progress-section">
          <el-steps :active="activeStep" finish-status="success" align-center>
            <el-step title="创建任务" />
            <el-step title="子域名收集" :description="`${task.subdomain_count} 个`" />
            <el-step title="存活探测" :description="`${task.alive_count} 个`" />
            <el-step title="XSS扫描" :description="`${task.xss_count} 个漏洞`" />
            <el-step title="完成" />
          </el-steps>
        </div>

        <div class="current-step" v-if="task.current_step">
          <el-icon class="is-loading" v-if="isRunning"><Loading /></el-icon>
          <span>{{ task.current_step }}</span>
        </div>

        <el-alert v-if="task.error_message" :title="task.error_message" type="error" show-icon :closable="false" class="error-alert" />
      </el-card>

      <!-- Stats Cards -->
      <div class="stats-row">
        <el-card shadow="hover">
          <div class="detail-stat">
            <el-icon :size="32" color="#409eff"><Connection /></el-icon>
            <div>
              <div class="stat-num">{{ task.subdomain_count }}</div>
              <div class="stat-label">子域名</div>
            </div>
          </div>
        </el-card>
        <el-card shadow="hover">
          <div class="detail-stat">
            <el-icon :size="32" color="#67c23a"><CircleCheck /></el-icon>
            <div>
              <div class="stat-num">{{ task.alive_count }}</div>
              <div class="stat-label">存活URL</div>
            </div>
          </div>
        </el-card>
        <el-card shadow="hover">
          <div class="detail-stat">
            <el-icon :size="32" color="#f56c6c"><Warning /></el-icon>
            <div>
              <div class="stat-num">{{ task.xss_count }}</div>
              <div class="stat-label">XSS漏洞</div>
            </div>
          </div>
        </el-card>
      </div>

      <!-- Tabs: Subdomains / Alive URLs / XSS Results -->
      <el-card shadow="hover">
        <el-tabs v-model="activeTab">
          <!-- Subdomains -->
          <el-tab-pane label="子域名" name="subdomains">
            <el-table :data="detail?.subdomains || []" stripe empty-text="暂无数据" max-height="400">
              <el-table-column type="index" width="60" />
              <el-table-column prop="domain" label="域名" />
              <el-table-column label="时间" width="180">
                <template #default="{ row }">
                  {{ formatTime(row.created_at) }}
                </template>
              </el-table-column>
            </el-table>
          </el-tab-pane>

          <!-- Alive URLs -->
          <el-tab-pane label="存活URL" name="alive">
            <el-table :data="detail?.alive_urls || []" stripe empty-text="暂无数据" max-height="400">
              <el-table-column type="index" width="60" />
              <el-table-column prop="url" label="URL" min-width="300">
                <template #default="{ row }">
                  <a :href="row.url" target="_blank" class="url-link">{{ row.url }}</a>
                </template>
              </el-table-column>
              <el-table-column prop="status_code" label="状态码" width="100" align="center" />
              <el-table-column prop="title" label="标题" min-width="200" />
            </el-table>
          </el-tab-pane>

          <!-- XSS Results -->
          <el-tab-pane name="xss">
            <template #label>
              <span>
                XSS漏洞
                <el-badge v-if="(detail?.xss_results || []).length > 0" :value="(detail?.xss_results || []).length" type="danger" />
              </span>
            </template>
            <div v-if="(detail?.xss_results || []).length === 0" class="empty-text">
              暂未发现XSS漏洞
            </div>
            <div v-else class="xss-results">
              <el-collapse>
                <el-collapse-item v-for="(xss, index) in detail.xss_results" :key="xss.id" :name="index">
                  <template #title>
                    <div class="xss-title">
                      <el-icon color="#f56c6c"><WarningFilled /></el-icon>
                      <span>{{ xss.url || `漏洞 #${index + 1}` }}</span>
                    </div>
                  </template>
                  <div class="report-content" v-html="renderMarkdown(xss.report_content)"></div>
                </el-collapse-item>
              </el-collapse>
            </div>
          </el-tab-pane>

          <!-- Full Report -->
          <el-tab-pane label="完整报告" name="report">
            <div v-if="!reportContent" class="empty-text">
              <el-button @click="loadReport" :loading="loadingReport">加载报告</el-button>
            </div>
            <div v-else class="report-content" v-html="renderMarkdown(reportContent)"></div>
          </el-tab-pane>
        </el-tabs>
      </el-card>
    </template>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRoute } from 'vue-router'
import { getTask, getReport } from '../api'
import { marked } from 'marked'
import { ElMessage } from 'element-plus'

const route = useRoute()
const taskId = route.params.id

const loading = ref(false)
const task = ref(null)
const detail = ref(null)
const activeTab = ref('subdomains')
const reportContent = ref('')
const loadingReport = ref(false)
let refreshTimer = null

const isRunning = computed(() =>
  task.value && ['pending', 'subdomain_collecting', 'httpx_probing', 'xss_scanning'].includes(task.value.status)
)

const activeStep = computed(() => {
  if (!task.value) return 0
  const map = {
    pending: 0,
    subdomain_collecting: 1,
    httpx_probing: 2,
    xss_scanning: 3,
    completed: 5,
    failed: -1,
  }
  return map[task.value.status] ?? 0
})

async function loadTaskDetail() {
  try {
    const data = await getTask(taskId)
    task.value = data.task
    detail.value = data
  } catch (err) {
    ElMessage.error('获取任务详情失败')
  }
}

async function loadReport() {
  loadingReport.value = true
  try {
    const data = await getReport(taskId)
    reportContent.value = data.report || '暂无报告内容'
  } catch (err) {
    reportContent.value = '暂无报告内容'
  } finally {
    loadingReport.value = false
  }
}

function renderMarkdown(content) {
  if (!content) return ''
  try {
    return marked(content)
  } catch {
    return content
  }
}

function statusType(status) {
  const map = {
    pending: 'info',
    subdomain_collecting: 'warning',
    httpx_probing: 'warning',
    xss_scanning: 'warning',
    completed: 'success',
    failed: 'danger',
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
  }
  return map[status] || status
}

function formatTime(time) {
  if (!time) return '-'
  return new Date(time).toLocaleString('zh-CN')
}

onMounted(() => {
  loading.value = true
  loadTaskDetail().finally(() => { loading.value = false })
  refreshTimer = setInterval(loadTaskDetail, 5000)
})

onUnmounted(() => {
  if (refreshTimer) clearInterval(refreshTimer)
})
</script>

<style scoped>
.task-detail {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.back-btn {
  align-self: flex-start;
}

.task-header {
  margin-bottom: 24px;
}

.task-title {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 8px;
}

.task-title h2 {
  font-size: 24px;
  font-weight: 600;
  color: #303133;
}

.task-meta {
  font-size: 13px;
  color: #909399;
}

.progress-section {
  margin: 24px 0;
}

.current-step {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px 16px;
  background: #f4f4f5;
  border-radius: 8px;
  font-size: 14px;
  color: #606266;
}

.error-alert {
  margin-top: 16px;
}

.stats-row {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 16px;
}

.detail-stat {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 8px;
}

.stat-num {
  font-size: 28px;
  font-weight: 700;
  color: #303133;
}

.stat-label {
  font-size: 14px;
  color: #909399;
}

.url-link {
  color: #409eff;
  text-decoration: none;
  word-break: break-all;
}

.url-link:hover {
  text-decoration: underline;
}

.empty-text {
  text-align: center;
  padding: 40px;
  color: #909399;
}

.xss-title {
  display: flex;
  align-items: center;
  gap: 8px;
}

.report-content {
  padding: 16px;
  line-height: 1.8;
  font-size: 14px;
}

.report-content :deep(pre) {
  background: #f5f7fa;
  padding: 12px;
  border-radius: 4px;
  overflow-x: auto;
}

.report-content :deep(code) {
  background: #f5f7fa;
  padding: 2px 4px;
  border-radius: 3px;
  font-size: 13px;
}

.report-content :deep(h1),
.report-content :deep(h2),
.report-content :deep(h3) {
  margin: 16px 0 8px;
  color: #303133;
}

@media (max-width: 768px) {
  .stats-row {
    grid-template-columns: 1fr;
  }
}
</style>
