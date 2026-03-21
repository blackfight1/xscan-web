<template>
  <div class="task-detail" v-loading="loading">
    <button class="back-btn" @click="$router.push('/')">
      <el-icon><ArrowLeft /></el-icon>
      Back to Tasks
    </button>

    <template v-if="task">
      <div class="detail-header">
        <div class="detail-header-glow"></div>
        <div class="detail-header-inner">
          <div class="header-title-row">
            <h2>{{ task.root_domain }}</h2>
            <span :class="['mode-badge', task.scan_mode === 'url' ? 'mode-url' : 'mode-domain']">
              {{ task.scan_mode === 'url' ? 'URL' : 'Domain' }}
            </span>
            <span :class="['status-badge', `status-${statusClass(task.status)}`]">
              <span class="status-dot"></span>
              {{ statusText(task.status) }}
            </span>
          </div>
          <div class="header-meta">
            <span>Task #{{ task.id }}</span>
            <span class="sep">|</span>
            <span>{{ formatTime(task.created_at) }}</span>
            <template v-if="duration">
              <span class="sep">|</span>
              <span>{{ duration }}</span>
            </template>
          </div>

          <div class="pipeline" v-if="task.scan_mode !== 'url'">
            <div :class="['ps', { active: activeStep >= 1, current: activeStep === 1 }]">
              <div class="pi">1</div><div class="pl">Subdomain</div><div class="pv">{{ task.subdomain_count }}</div>
            </div>
            <div class="pline" :class="{ filled: activeStep >= 2 }"></div>
            <div :class="['ps', { active: activeStep >= 2, current: activeStep === 2 }]">
              <div class="pi">2</div><div class="pl">Probing</div><div class="pv">{{ task.alive_count }}</div>
            </div>
            <div class="pline" :class="{ filled: activeStep >= 3 }"></div>
            <div :class="['ps', { active: activeStep >= 3, current: activeStep === 3 }]">
              <div class="pi">3</div><div class="pl">XSS Scan</div><div class="pv">{{ task.xss_count }}</div>
            </div>
            <div class="pline" :class="{ filled: activeStep >= 5 }"></div>
            <div :class="['ps', { active: activeStep >= 5 }]">
              <div class="pi">OK</div><div class="pl">Done</div>
            </div>
          </div>
          <div class="pipeline" v-else>
            <div :class="['ps', { active: urlStep >= 1, current: urlStep === 1 }]">
              <div class="pi">1</div><div class="pl">XSS Scan</div><div class="pv">{{ task.xss_count }}</div>
            </div>
            <div class="pline" :class="{ filled: urlStep >= 3 }"></div>
            <div :class="['ps', { active: urlStep >= 3 }]">
              <div class="pi">OK</div><div class="pl">Done</div>
            </div>
          </div>

          <div class="cur-step" v-if="task.current_step && isRunning">
            <div class="pulse-dot"></div>
            <span>{{ task.current_step }}</span>
          </div>
          <div class="err-bar" v-if="task.error_message">
            <el-icon color="#f87171"><WarningFilled /></el-icon>
            <span>{{ task.error_message }}</span>
          </div>
        </div>
      </div>

      <div class="stats-row">
        <div class="sc" v-if="task.scan_mode !== 'url'">
          <div class="si si-sub"><el-icon :size="20"><Connection /></el-icon></div>
          <div><div class="sn">{{ task.subdomain_count }}</div><div class="sl">Subdomains</div></div>
        </div>
        <div class="sc" v-if="task.scan_mode !== 'url'">
          <div class="si si-alive"><el-icon :size="20"><CircleCheck /></el-icon></div>
          <div><div class="sn">{{ task.alive_count }}</div><div class="sl">Alive</div></div>
        </div>
        <div class="sc sc-xss">
          <div class="si si-xss"><el-icon :size="20"><Warning /></el-icon></div>
          <div><div class="sn">{{ task.xss_count }}</div><div class="sl">XSS Vulns</div></div>
        </div>
        <div class="sc">
          <div class="si si-url"><el-icon :size="20"><Link /></el-icon></div>
          <div><div class="sn">{{ urlGroups.length }}</div><div class="sl">Vulnerable URLs</div></div>
        </div>
      </div>

      <div class="content-box">
        <div class="tab-nav">
          <button v-if="task.scan_mode !== 'url'" :class="['tb', { active: tab === 'sub' }]" @click="tab = 'sub'">Subdomains</button>
          <button v-if="task.scan_mode !== 'url'" :class="['tb', { active: tab === 'alive' }]" @click="tab = 'alive'">Alive URLs</button>
          <button :class="['tb', { active: tab === 'xss' }]" @click="tab = 'xss'">
            XSS Vulns<span v-if="xssFindings.length" class="tbadge">{{ xssFindings.length }}</span>
          </button>
          <button :class="['tb', { active: tab === 'report' }]" @click="tab = 'report'">Report</button>
        </div>

        <div class="tab-body">
          <div v-if="tab === 'sub'" class="tp">
            <div class="toolbar">
              <el-input v-model="subFilter" placeholder="Filter..." size="small" clearable class="fi" />
              <span class="rc">{{ filteredSubs.length }} records</span>
            </div>
            <el-table :data="filteredSubs" stripe empty-text="No data" max-height="500">
              <el-table-column type="index" width="60" />
              <el-table-column prop="domain" label="Domain" sortable />
              <el-table-column label="Time" width="180">
                <template #default="{ row }">{{ formatTime(row.created_at) }}</template>
              </el-table-column>
            </el-table>
          </div>

          <div v-if="tab === 'alive'" class="tp">
            <div class="toolbar">
              <el-input v-model="aliveFilter" placeholder="Filter..." size="small" clearable class="fi" />
              <span class="rc">{{ filteredAlive.length }} records</span>
            </div>
            <el-table :data="filteredAlive" stripe empty-text="No data" max-height="500">
              <el-table-column type="index" width="60" />
              <el-table-column prop="url" label="URL" min-width="300">
                <template #default="{ row }">
                  <a :href="row.url" target="_blank" class="ulink">{{ row.url }}</a>
                </template>
              </el-table-column>
              <el-table-column prop="status_code" label="Status" width="90" align="center">
                <template #default="{ row }">
                  <span :class="['hcode', codeClass(row.status_code)]">{{ row.status_code }}</span>
                </template>
              </el-table-column>
              <el-table-column prop="title" label="Title" min-width="200" show-overflow-tooltip />
            </el-table>
          </div>

          <div v-if="tab === 'xss'" class="tp xss-panel">
            <div v-if="!xssFindings.length" class="empty">
              <el-icon :size="48" color="var(--text-muted)"><CircleClose /></el-icon>
              <p>No XSS vulnerabilities found</p>
            </div>

            <div v-else>
              <div class="toolbar xss-toolbar">
                <el-input v-model="findingFilter" placeholder="Filter URL / parameter / payload" size="small" clearable class="fi fi-wide" />
                <div class="toolbar-right">
                  <span class="rc">{{ filteredUrlGroups.length }} URLs | {{ xssFindings.length }} findings</span>
                  <button class="exp-btn" @click="exportCSV"><el-icon><Download /></el-icon> Export CSV</button>
                </div>
              </div>

              <div class="xss-grid">
                <div class="url-list">
                  <button
                    v-for="group in filteredUrlGroups"
                    :key="group.key"
                    :class="['url-item', { active: selectedUrlKey === group.key }]"
                    @click="selectUrl(group.key)"
                  >
                    <div class="url-item-main">{{ group.url }}</div>
                    <div class="url-item-meta">
                      <span>{{ group.host }}</span>
                      <span>|</span>
                      <span>{{ group.count }} finding{{ group.count > 1 ? 's' : '' }}</span>
                    </div>
                    <span class="url-item-count">{{ group.count }}</span>
                  </button>
                </div>

                <div class="finding-view">
                  <template v-if="activeGroup">
                    <div class="finding-view-head">
                      <div>
                        <div class="fv-url">{{ activeGroup.url }}</div>
                        <div class="fv-meta">{{ activeGroup.host }} | {{ activeGroup.count }} finding{{ activeGroup.count > 1 ? 's' : '' }}</div>
                      </div>
                    </div>

                    <div class="vlist">
                      <div v-for="finding in activeGroup.findings" :key="finding.id" class="vitem" :class="{ open: openedFindingId === finding.id }">
                        <div class="vh" @click="toggleFinding(finding.id)">
                          <div class="vhl">
                            <el-icon color="#f87171" :size="18"><WarningFilled /></el-icon>
                            <div>
                              <div class="vu">{{ finding.title }}</div>
                              <div class="vm">
                                <span v-if="finding.type">{{ finding.type }}</span>
                                <span v-if="finding.parameter"> | {{ finding.parameter }}</span>
                                <span v-if="finding.position"> | {{ finding.position }}</span>
                                <span v-if="finding.time"> | {{ finding.time }}</span>
                              </div>
                            </div>
                          </div>
                          <el-icon class="varrow" :class="{ rotated: openedFindingId === finding.id }"><ArrowDown /></el-icon>
                        </div>

                        <div class="vb" v-show="openedFindingId === finding.id">
                          <div class="meta-grid">
                            <div class="meta-cell"><span class="mk">URL</span><span class="mv">{{ finding.vulnUrl }}</span></div>
                            <div class="meta-cell"><span class="mk">Type</span><span class="mv">{{ finding.type || '-' }}</span></div>
                            <div class="meta-cell"><span class="mk">Parameter</span><span class="mv">{{ finding.parameter || '-' }}</span></div>
                            <div class="meta-cell"><span class="mk">Position</span><span class="mv">{{ finding.position || '-' }}</span></div>
                            <div class="meta-cell"><span class="mk">Hidden Param</span><span class="mv">{{ finding.hiddenParameter || '-' }}</span></div>
                            <div class="meta-cell"><span class="mk">Time</span><span class="mv">{{ finding.time || '-' }}</span></div>
                          </div>

                          <div v-if="finding.payload" class="payload-box">
                            <div class="payload-title">Payload</div>
                            <code>{{ finding.payload }}</code>
                          </div>

                          <div class="md-body" v-html="renderMd(finding.report_content)"></div>
                        </div>
                      </div>
                    </div>
                  </template>
                  <div v-else class="empty compact"><p>No URL matched this filter</p></div>
                </div>
              </div>
            </div>
          </div>

          <div v-if="tab === 'report'" class="tp">
            <div v-if="!reportContent" class="empty">
              <el-button @click="loadReport" :loading="loadingReport">Load Full Report</el-button>
            </div>
            <div v-else class="rpt">
              <div class="rpt-hdr"><h3>Scan Report</h3><span>{{ task.root_domain }}</span></div>
              <div class="md-body" v-html="renderMd(reportContent)"></div>
            </div>
          </div>
        </div>
      </div>
    </template>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import { useRoute } from 'vue-router'
import { getTask, getReport } from '../api'
import { marked } from 'marked'
import { ElMessage } from 'element-plus'

const route = useRoute()
const taskId = route.params.id
const loading = ref(false)
const task = ref(null)
const detail = ref(null)
const tab = ref('xss')
const reportContent = ref('')
const loadingReport = ref(false)
const subFilter = ref('')
const aliveFilter = ref('')
const findingFilter = ref('')
const selectedUrlKey = ref('')
const openedFindingId = ref('')
let timer = null

const isRunning = computed(() => task.value && ['pending', 'subdomain_collecting', 'httpx_probing', 'xss_scanning'].includes(task.value.status))

const filteredSubs = computed(() => {
  const list = detail.value?.subdomains || []
  if (!subFilter.value) return list
  const key = subFilter.value.toLowerCase()
  return list.filter((item) => item.domain?.toLowerCase().includes(key))
})

const filteredAlive = computed(() => {
  const list = detail.value?.alive_urls || []
  if (!aliveFilter.value) return list
  const key = aliveFilter.value.toLowerCase()
  return list.filter((item) => item.url?.toLowerCase().includes(key) || item.title?.toLowerCase().includes(key))
})

const activeStep = computed(() => {
  if (!task.value) return 0
  return { pending: 0, subdomain_collecting: 1, httpx_probing: 2, xss_scanning: 3, completed: 5, failed: -1 }[task.value.status] ?? 0
})

const urlStep = computed(() => {
  if (!task.value) return 0
  return { pending: 0, xss_scanning: 1, completed: 3, failed: -1 }[task.value.status] ?? 0
})

const duration = computed(() => {
  if (!task.value?.created_at || !task.value?.finished_at) return ''
  const diff = Math.floor((new Date(task.value.finished_at) - new Date(task.value.created_at)) / 1000)
  if (diff < 60) return `${diff}s`
  if (diff < 3600) return `${Math.floor(diff / 60)}m ${diff % 60}s`
  return `${Math.floor(diff / 3600)}h ${Math.floor((diff % 3600) / 60)}m`
})

const xssFindings = computed(() => {
  const list = detail.value?.xss_results || []
  return list.map((item, idx) => buildFinding(item, idx))
})

const urlGroups = computed(() => {
  const map = new Map()
  for (const finding of xssFindings.value) {
    const key = finding.vulnUrl || `finding-${finding.id}`
    if (!map.has(key)) {
      map.set(key, { key, url: finding.vulnUrl, host: finding.host || '-', count: 0, findings: [] })
    }
    const group = map.get(key)
    group.count += 1
    group.findings.push(finding)
  }
  return Array.from(map.values()).sort((a, b) => {
    if (a.count !== b.count) return b.count - a.count
    return a.url.localeCompare(b.url)
  })
})

const filteredUrlGroups = computed(() => {
  if (!findingFilter.value) return urlGroups.value
  const key = findingFilter.value.toLowerCase()
  return urlGroups.value.filter((group) => {
    if (group.url.toLowerCase().includes(key)) return true
    return group.findings.some((finding) => {
      return (
        String(finding.parameter || '').toLowerCase().includes(key) ||
        String(finding.payload || '').toLowerCase().includes(key) ||
        String(finding.position || '').toLowerCase().includes(key)
      )
    })
  })
})

const activeGroup = computed(() => {
  return filteredUrlGroups.value.find((group) => group.key === selectedUrlKey.value) || filteredUrlGroups.value[0] || null
})

watch(
  filteredUrlGroups,
  (groups) => {
    if (!groups.length) {
      selectedUrlKey.value = ''
      openedFindingId.value = ''
      return
    }

    const exists = groups.some((group) => group.key === selectedUrlKey.value)
    if (!exists) {
      selectedUrlKey.value = groups[0].key
      openedFindingId.value = groups[0].findings[0]?.id || ''
    }
  },
  { immediate: true }
)

function codeClass(code) {
  if (code >= 200 && code < 300) return 'c2'
  if (code >= 300 && code < 400) return 'c3'
  return 'c4'
}

function selectUrl(key) {
  selectedUrlKey.value = key
  const current = filteredUrlGroups.value.find((group) => group.key === key)
  openedFindingId.value = current?.findings[0]?.id || ''
}

function toggleFinding(id) {
  openedFindingId.value = openedFindingId.value === id ? '' : id
}

function sanitizeStoredURL(raw) {
  const value = String(raw || '').trim()
  if (!value) return ''
  if (/^https?:\/\//i.test(value)) return value
  if (value.includes('/results/') || value.endsWith('_xss.md') || value.includes('\\')) return ''
  return ''
}

function extractHttpURL(text) {
  const match = String(text || '').match(/https?:\/\/[^\s)"'<>]+/i)
  return match ? match[0].trim() : ''
}

function normalizeMetaKey(key) {
  return String(key || '')
    .toLowerCase()
    .replace(/[\s:_-]/g, '')
}

function parseTableMeta(content) {
  const meta = {}
  const lines = String(content || '').split(/\r?\n/)
  for (const line of lines) {
    if (!line.includes('|')) continue
    const cols = line.split('|').map((item) => item.trim()).filter(Boolean)
    if (cols.length < 2) continue
    if (/^-+$/.test(cols[0]) || /^-+$/.test(cols[1])) continue
    const key = normalizeMetaKey(cols[0])
    if (!key || key === 'key' || key === 'value') continue
    meta[key] = cols[1]
  }
  return meta
}

function pickMeta(meta, keys) {
  for (const key of keys) {
    const normalized = normalizeMetaKey(key)
    if (meta[normalized]) return meta[normalized]
  }
  return ''
}

function hostFromUrl(value) {
  try {
    if (!/^https?:\/\//i.test(value)) return '-'
    return new URL(value).host
  } catch {
    return '-'
  }
}

function buildFinding(item, idx) {
  const report = String(item.report_content || '')
  const meta = parseTableMeta(report)
  const storedURL = sanitizeStoredURL(item.url)
  const detectedURL = extractHttpURL(report)
  const vulnUrl = storedURL || detectedURL || `Finding #${idx + 1}`
  const parameter = pickMeta(meta, ['\u53c2\u6570\u540d', '\u53c2\u6570', 'parameter', 'param'])
  const type = pickMeta(meta, ['xss\u7c7b\u578b', 'type', 'xsstype'])
  const position = pickMeta(meta, ['xss\u4f4d\u7f6e', '\u4f4d\u7f6e', 'position'])
  const hiddenParameter = pickMeta(meta, ['\u9690\u85cf\u53c2\u6570', 'hiddenparameter', 'hidden'])
  const time = pickMeta(meta, ['\u65f6\u95f4', 'time'])
  const payload = pickMeta(meta, ['payload'])

  return {
    id: item.id || `finding-${idx}`,
    vulnUrl,
    host: hostFromUrl(vulnUrl),
    title: parameter ? `${parameter} vulnerability` : `Finding #${idx + 1}`,
    parameter,
    type,
    position,
    hiddenParameter,
    time,
    payload,
    report_content: report,
  }
}

async function load() {
  try {
    const data = await getTask(taskId)
    task.value = data.task
    detail.value = data
    if (!isRunning.value && timer) {
      clearInterval(timer)
      timer = null
    }
    if (task.value?.scan_mode === 'url' && tab.value === 'sub') tab.value = 'xss'
  } catch {
    ElMessage.error('Failed to load task')
  }
}

async function loadReport() {
  loadingReport.value = true
  try {
    reportContent.value = (await getReport(taskId)).report || 'No report'
  } catch {
    reportContent.value = 'No report'
  } finally {
    loadingReport.value = false
  }
}

function renderMd(content) {
  try {
    return content ? marked(content) : ''
  } catch {
    return content
  }
}

function csvEscape(value) {
  return `"${String(value || '').replace(/"/g, '""').replace(/\n/g, ' ')}"`
}

function exportCSV() {
  if (!xssFindings.value.length) return
  const rows = xssFindings.value.map((finding, idx) => {
    return [
      idx + 1,
      csvEscape(finding.vulnUrl),
      csvEscape(finding.type),
      csvEscape(finding.parameter),
      csvEscape(finding.position),
      csvEscape(finding.time),
      csvEscape(finding.payload),
    ].join(',')
  })
  const csv = ['#,URL,Type,Parameter,Position,Time,Payload', ...rows].join('\n')
  const a = document.createElement('a')
  a.href = URL.createObjectURL(new Blob([`\uFEFF${csv}`], { type: 'text/csv;charset=utf-8;' }))
  a.download = `xss_task_${taskId}.csv`
  a.click()
}

function statusClass(status) {
  return { pending: 'pending', subdomain_collecting: 'running', httpx_probing: 'running', xss_scanning: 'running', completed: 'completed', failed: 'failed' }[status] || 'pending'
}

function statusText(status) {
  return { pending: 'Pending', subdomain_collecting: 'Collecting', httpx_probing: 'Probing', xss_scanning: 'Scanning', completed: 'Completed', failed: 'Failed' }[status] || status
}

function formatTime(value) {
  return value
    ? new Date(value).toLocaleString('en-US', { hour12: false, month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' })
    : '-'
}

onMounted(() => {
  loading.value = true
  load().finally(() => {
    loading.value = false
  })
  timer = setInterval(load, 5000)
})

onUnmounted(() => {
  if (timer) clearInterval(timer)
})
</script>

<style scoped>
.task-detail { display:flex; flex-direction:column; gap:24px; }

.back-btn { display:inline-flex; align-items:center; gap:6px; padding:8px 16px; border-radius:8px; border:1px solid var(--border-color); background:transparent; color:var(--text-muted); cursor:pointer; font-size:13px; transition:all .2s; align-self:flex-start; }
.back-btn:hover { color:var(--accent); border-color:var(--accent); }

.detail-header { position:relative; border-radius:16px; overflow:hidden; background:var(--bg-card); border:1px solid var(--border-color); }
.detail-header-glow { position:absolute; inset:0; background:linear-gradient(135deg, rgba(0,212,255,.05), rgba(99,102,241,.05)); pointer-events:none; }
.detail-header-inner { position:relative; padding:28px 32px; }

.header-title-row { display:flex; align-items:center; gap:12px; flex-wrap:wrap; margin-bottom:8px; }
.header-title-row h2 { font-size:22px; font-weight:700; color:var(--text-primary); word-break:break-all; }

.mode-badge { padding:3px 10px; border-radius:6px; font-size:11px; font-weight:600; text-transform:uppercase; letter-spacing:.5px; }
.mode-domain { background:rgba(99,102,241,.15); color:#818cf8; }
.mode-url { background:rgba(16,185,129,.15); color:#34d399; }

.status-badge { display:inline-flex; align-items:center; gap:6px; padding:4px 12px; border-radius:20px; font-size:12px; font-weight:500; }
.status-dot { width:6px; height:6px; border-radius:50%; }
.status-pending { background:rgba(100,116,139,.15); color:#94a3b8; }
.status-pending .status-dot { background:#94a3b8; }
.status-running { background:rgba(245,158,11,.15); color:#fbbf24; }
.status-running .status-dot { background:#fbbf24; animation:pulse 1.5s infinite; }
.status-completed { background:rgba(16,185,129,.15); color:#34d399; }
.status-completed .status-dot { background:#34d399; }
.status-failed { background:rgba(239,68,68,.15); color:#f87171; }
.status-failed .status-dot { background:#f87171; }
@keyframes pulse { 0%,100%{opacity:1} 50%{opacity:.4} }

.header-meta { font-size:13px; color:var(--text-muted); }
.sep { margin:0 8px; opacity:.4; }

/* Pipeline */
.pipeline { display:flex; align-items:center; margin:28px 0 8px; }
.ps { display:flex; flex-direction:column; align-items:center; gap:6px; min-width:80px; }
.pi { width:36px; height:36px; border-radius:50%; display:flex; align-items:center; justify-content:center; font-size:13px; font-weight:700; border:2px solid var(--border-color); color:var(--text-muted); background:var(--bg-input); transition:all .3s; }
.ps.active .pi { border-color:var(--accent); color:var(--accent); background:var(--accent-glow); }
.ps.current .pi { border-color:#fbbf24; color:#fbbf24; background:rgba(245,158,11,.15); box-shadow:0 0 12px rgba(245,158,11,.3); }
.pl { font-size:11px; color:var(--text-muted); text-transform:uppercase; letter-spacing:.5px; }
.pv { font-size:16px; font-weight:700; color:var(--text-primary); }
.pline { flex:1; height:2px; background:var(--border-color); margin-bottom:42px; transition:background .3s; }
.pline.filled { background:var(--accent); }

.cur-step { display:flex; align-items:center; gap:10px; padding:12px 16px; background:rgba(245,158,11,.08); border:1px solid rgba(245,158,11,.2); border-radius:10px; font-size:13px; color:#fbbf24; margin-top:16px; }
.pulse-dot { width:8px; height:8px; border-radius:50%; background:#fbbf24; animation:pulse 1.5s infinite; flex-shrink:0; }
.err-bar { display:flex; align-items:center; gap:10px; padding:12px 16px; background:rgba(239,68,68,.08); border:1px solid rgba(239,68,68,.2); border-radius:10px; font-size:13px; color:#fca5a5; margin-top:16px; }

/* Stats */
.stats-row { display:grid; grid-template-columns:repeat(auto-fit,minmax(180px,1fr)); gap:16px; }
.sc { display:flex; align-items:center; gap:14px; padding:20px; background:var(--bg-card); border:1px solid var(--border-color); border-radius:12px; }
.si { width:40px; height:40px; border-radius:10px; display:flex; align-items:center; justify-content:center; }
.si-sub { background:rgba(99,102,241,.15); color:#818cf8; }
.si-alive { background:rgba(16,185,129,.15); color:#34d399; }
.si-xss { background:rgba(239,68,68,.15); color:#f87171; }
.si-url { background:rgba(14,165,233,.15); color:#38bdf8; }
.sn { font-size:24px; font-weight:700; color:var(--text-primary); }
.sl { font-size:12px; color:var(--text-muted); text-transform:uppercase; letter-spacing:.5px; }
.sc-xss { border-color:rgba(239,68,68,.2); }

/* Tabs */
.content-box { background:var(--bg-card); border:1px solid var(--border-color); border-radius:16px; overflow:hidden; }
.tab-nav { display:flex; border-bottom:1px solid var(--border-color); padding:0 8px; }
.tb { padding:14px 20px; border:none; background:transparent; color:var(--text-muted); font-size:14px; font-weight:500; cursor:pointer; position:relative; transition:color .2s; display:flex; align-items:center; gap:6px; }
.tb:hover { color:var(--text-secondary); }
.tb.active { color:var(--accent); }
.tb.active::after { content:''; position:absolute; bottom:0; left:16px; right:16px; height:2px; background:var(--accent); border-radius:1px; }
.tbadge { background:rgba(239,68,68,.2); color:#f87171; padding:1px 7px; border-radius:10px; font-size:11px; font-weight:700; }

.tab-body { min-height:300px; }
.tp { padding:20px 24px; }
.toolbar { display:flex; justify-content:space-between; align-items:center; margin-bottom:16px; gap:12px; }
.fi { max-width:260px; }
.fi-wide { max-width:360px; }
.toolbar-right { display:flex; align-items:center; gap:14px; }
.rc { font-size:12px; color:var(--text-muted); }
.ulink { color:var(--accent); text-decoration:none; word-break:break-all; font-size:13px; }
.ulink:hover { text-decoration:underline; }
.hcode { font-size:12px; font-weight:600; padding:2px 8px; border-radius:4px; }
.c2 { background:rgba(16,185,129,.15); color:#34d399; }
.c3 { background:rgba(245,158,11,.15); color:#fbbf24; }
.c4 { background:rgba(239,68,68,.15); color:#f87171; }

.empty { display:flex; flex-direction:column; align-items:center; justify-content:center; padding:60px 20px; gap:12px; }
.empty.compact { padding:28px 12px; }
.empty p { color:var(--text-muted); font-size:14px; }

.exp-btn { display:inline-flex; align-items:center; gap:6px; padding:6px 14px; border-radius:8px; border:1px solid var(--border-color); background:transparent; color:var(--text-secondary); font-size:13px; cursor:pointer; transition:all .2s; }
.exp-btn:hover { border-color:var(--accent); color:var(--accent); }

.xss-panel { display:flex; flex-direction:column; }
.xss-grid { display:grid; grid-template-columns:320px minmax(0,1fr); gap:16px; min-height:420px; }
.url-list { border:1px solid var(--border-color); border-radius:12px; overflow:auto; max-height:620px; background:rgba(0,212,255,.02); }
.url-item { width:100%; text-align:left; background:transparent; border:none; border-bottom:1px solid rgba(148,163,184,.12); padding:12px 14px; cursor:pointer; color:var(--text-secondary); position:relative; transition:background .2s; }
.url-item:last-child { border-bottom:none; }
.url-item:hover { background:rgba(0,212,255,.05); }
.url-item.active { background:rgba(0,212,255,.09); box-shadow:inset 2px 0 0 var(--accent); }
.url-item-main { font-size:13px; color:var(--text-primary); white-space:nowrap; overflow:hidden; text-overflow:ellipsis; padding-right:36px; }
.url-item-meta { margin-top:4px; font-size:11px; color:var(--text-muted); }
.url-item-count { position:absolute; right:12px; top:13px; min-width:20px; height:20px; border-radius:999px; background:rgba(239,68,68,.18); color:#f87171; font-size:11px; font-weight:700; display:flex; align-items:center; justify-content:center; }
.finding-view { border:1px solid var(--border-color); border-radius:12px; overflow:hidden; background:rgba(15,23,42,.18); }
.finding-view-head { padding:14px 18px; border-bottom:1px solid var(--border-color); background:rgba(0,212,255,.04); }
.fv-url { font-size:14px; color:#22d3ee; font-weight:600; word-break:break-all; }
.fv-meta { margin-top:4px; font-size:12px; color:var(--text-muted); }

/* Vuln list */
.vlist { display:flex; flex-direction:column; gap:8px; padding:10px; max-height:560px; overflow:auto; }
.vitem { border:1px solid var(--border-color); border-radius:12px; overflow:hidden; transition:border-color .2s; }
.vitem:hover { border-color:rgba(239,68,68,.3); }
.vitem.open { border-color:rgba(239,68,68,.4); }
.vh { display:flex; align-items:center; justify-content:space-between; padding:14px 18px; cursor:pointer; transition:background .2s; }
.vh:hover { background:rgba(239,68,68,.04); }
.vhl { display:flex; align-items:center; gap:12px; flex:1; min-width:0; }
.vu { color:var(--text-primary); font-size:13px; font-weight:500; word-break:break-all; }
.vm { color:var(--text-muted); font-size:11px; margin-top:2px; }
.varrow { color:var(--text-muted); transition:transform .2s; flex-shrink:0; }
.varrow.rotated { transform:rotate(180deg); }
.vb { border-top:1px solid var(--border-color); }
.meta-grid { display:grid; grid-template-columns:repeat(2,minmax(0,1fr)); gap:10px; padding:14px; }
.meta-cell { display:flex; flex-direction:column; gap:4px; background:rgba(15,23,42,.38); border:1px solid rgba(148,163,184,.16); border-radius:8px; padding:8px 10px; }
.mk { font-size:11px; color:var(--text-muted); }
.mv { font-size:12px; color:var(--text-primary); word-break:break-all; }
.payload-box { margin:0 14px 12px; border:1px solid rgba(14,165,233,.3); background:rgba(14,165,233,.07); border-radius:8px; padding:10px; }
.payload-title { font-size:11px; color:#7dd3fc; margin-bottom:6px; text-transform:uppercase; letter-spacing:.5px; }
.payload-box code { color:#bae6fd; font-size:12px; white-space:pre-wrap; word-break:break-all; }

/* Report */
.rpt { padding:0; }
.rpt-hdr { display:flex; align-items:center; justify-content:space-between; padding:16px 20px; border-bottom:1px solid var(--border-color); }
.rpt-hdr h3 { font-size:16px; font-weight:600; color:var(--text-primary); }
.rpt-hdr span { font-size:13px; color:var(--text-muted); }

/* Markdown report styling */
.md-body { padding:20px; line-height:1.8; font-size:14px; color:var(--text-secondary); }
.md-body :deep(h1) { font-size:20px; font-weight:700; color:var(--text-primary); margin:24px 0 12px; padding-bottom:8px; border-bottom:1px solid var(--border-color); }
.md-body :deep(h2) { font-size:17px; font-weight:600; color:var(--text-primary); margin:20px 0 10px; }
.md-body :deep(h3) { font-size:15px; font-weight:600; color:var(--text-primary); margin:16px 0 8px; }
.md-body :deep(p) { margin:8px 0; }
.md-body :deep(a) { color:var(--accent); text-decoration:none; }
.md-body :deep(a:hover) { text-decoration:underline; }
.md-body :deep(strong) { color:var(--text-primary); font-weight:600; }
.md-body :deep(code) { background:rgba(0,212,255,.08); color:#7dd3fc; padding:2px 6px; border-radius:4px; font-size:13px; font-family:'JetBrains Mono',monospace; }
.md-body :deep(pre) { background:var(--bg-input); border:1px solid var(--border-color); border-radius:8px; padding:16px; overflow-x:auto; margin:12px 0; }
.md-body :deep(pre code) { background:none; padding:0; color:var(--text-secondary); }
.md-body :deep(blockquote) { border-left:3px solid var(--accent); padding:8px 16px; margin:12px 0; background:rgba(0,212,255,.04); border-radius:0 8px 8px 0; }
.md-body :deep(ul), .md-body :deep(ol) { padding-left:24px; margin:8px 0; }
.md-body :deep(li) { margin:4px 0; }
.md-body :deep(table) { width:100%; border-collapse:collapse; margin:12px 0; }
.md-body :deep(th) { background:rgba(0,212,255,.06); padding:10px 14px; text-align:left; font-weight:600; color:var(--text-primary); border:1px solid var(--border-color); font-size:13px; }
.md-body :deep(td) { padding:8px 14px; border:1px solid var(--border-color); font-size:13px; }
.md-body :deep(tr:hover td) { background:rgba(0,212,255,.04); }
.md-body :deep(hr) { border:none; border-top:1px solid var(--border-color); margin:16px 0; }
.md-body :deep(img) { max-width:100%; border-radius:8px; margin:8px 0; }

@media (max-width:768px) {
  .stats-row { grid-template-columns:1fr; }
  .toolbar { flex-direction:column; align-items:flex-start; }
  .toolbar-right { width:100%; justify-content:space-between; }
  .pipeline { flex-wrap:wrap; gap:8px; }
  .meta-grid { grid-template-columns:1fr; }
}

@media (max-width:1200px) {
  .xss-grid { grid-template-columns:1fr; }
  .url-list { max-height:260px; }
}
</style>


