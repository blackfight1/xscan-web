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
            <span class="sep">•</span>
            <span>{{ formatTime(task.created_at) }}</span>
            <template v-if="duration">
              <span class="sep">•</span>
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
              <div class="pi">✓</div><div class="pl">Done</div>
            </div>
          </div>
          <div class="pipeline" v-else>
            <div :class="['ps', { active: urlStep >= 1, current: urlStep === 1 }]">
              <div class="pi">1</div><div class="pl">XSS Scan</div><div class="pv">{{ task.xss_count }}</div>
            </div>
            <div class="pline" :class="{ filled: urlStep >= 3 }"></div>
            <div :class="['ps', { active: urlStep >= 3 }]">
              <div class="pi">✓</div><div class="pl">Done</div>
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
      </div>

      <div class="content-box">
        <div class="tab-nav">
          <button v-if="task.scan_mode !== 'url'" :class="['tb', { active: tab === 'sub' }]" @click="tab = 'sub'">Subdomains</button>
          <button v-if="task.scan_mode !== 'url'" :class="['tb', { active: tab === 'alive' }]" @click="tab = 'alive'">Alive URLs</button>
          <button :class="['tb', { active: tab === 'xss' }]" @click="tab = 'xss'">
            XSS Vulns<span v-if="xssResults.length" class="tbadge">{{ xssResults.length }}</span>
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
                  <span :class="['hcode', cc(row.status_code)]">{{ row.status_code }}</span>
                </template>
              </el-table-column>
              <el-table-column prop="title" label="Title" min-width="200" show-overflow-tooltip />
            </el-table>
          </div>

          <div v-if="tab === 'xss'" class="tp">
            <div v-if="!xssResults.length" class="empty">
              <el-icon :size="48" color="var(--text-muted)"><CircleClose /></el-icon>
              <p>No XSS vulnerabilities found</p>
            </div>
            <div v-else>
              <div class="toolbar">
                <span class="rc">{{ xssResults.length }} vulnerabilities</span>
                <button class="exp-btn" @click="exportCSV"><el-icon><Download /></el-icon> Export CSV</button>
              </div>
              <div class="vlist">
                <div v-for="(x, i) in xssResults" :key="x.id" class="vitem" :class="{ open: expanded === i }">
                  <div class="vh" @click="expanded = expanded === i ? -1 : i">
                    <div class="vhl">
                      <el-icon color="#f87171" :size="18"><WarningFilled /></el-icon>
                      <div>
                        <div class="vu">{{ x.url || `Vulnerability #${i + 1}` }}</div>
                        <div class="vm">Finding #{{ i + 1 }}</div>
                      </div>
                    </div>
                    <el-icon class="varrow" :class="{ rotated: expanded === i }"><ArrowDown /></el-icon>
                  </div>
                  <div class="vb" v-show="expanded === i">
                    <div class="md-body" v-html="renderMd(x.report_content)"></div>
                  </div>
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
const tab = ref('xss')
const reportContent = ref('')
const loadingReport = ref(false)
const subFilter = ref('')
const aliveFilter = ref('')
const expanded = ref(0)
let timer = null

const isRunning = computed(() => task.value && ['pending','subdomain_collecting','httpx_probing','xss_scanning'].includes(task.value.status))
const xssResults = computed(() => detail.value?.xss_results || [])
const filteredSubs = computed(() => {
  const l = detail.value?.subdomains || []
  return subFilter.value ? l.filter(s => s.domain?.toLowerCase().includes(subFilter.value.toLowerCase())) : l
})
const filteredAlive = computed(() => {
  const l = detail.value?.alive_urls || []
  if (!aliveFilter.value) return l
  const k = aliveFilter.value.toLowerCase()
  return l.filter(u => u.url?.toLowerCase().includes(k) || u.title?.toLowerCase().includes(k))
})
const activeStep = computed(() => {
  if (!task.value) return 0
  return { pending:0, subdomain_collecting:1, httpx_probing:2, xss_scanning:3, completed:5, failed:-1 }[task.value.status] ?? 0
})
const urlStep = computed(() => {
  if (!task.value) return 0
  return { pending:0, xss_scanning:1, completed:3, failed:-1 }[task.value.status] ?? 0
})
const duration = computed(() => {
  if (!task.value?.created_at || !task.value?.finished_at) return ''
  const d = Math.floor((new Date(task.value.finished_at) - new Date(task.value.created_at)) / 1000)
  if (d < 60) return `${d}s`
  if (d < 3600) return `${Math.floor(d/60)}m ${d%60}s`
  return `${Math.floor(d/3600)}h ${Math.floor((d%3600)/60)}m`
})

function cc(c) { return c >= 200 && c < 300 ? 'c2' : c >= 300 && c < 400 ? 'c3' : 'c4' }

async function load() {
  try {
    const data = await getTask(taskId)
    task.value = data.task; detail.value = data
    if (!isRunning.value && timer) { clearInterval(timer); timer = null }
    if (task.value?.scan_mode === 'url' && tab.value === 'sub') tab.value = 'xss'
  } catch { ElMessage.error('Failed to load task') }
}

async function loadReport() {
  loadingReport.value = true
  try { reportContent.value = (await getReport(taskId)).report || 'No report' }
  catch { reportContent.value = 'No report' }
  finally { loadingReport.value = false }
}

function renderMd(c) { try { return c ? marked(c) : '' } catch { return c } }

function exportCSV() {
  const r = xssResults.value; if (!r.length) return
  const rows = r.map((x,i) => [i+1, `"${(x.url||'').replace(/"/g,'""')}"`, `"${(x.report_content||'').replace(/"/g,'""').replace(/\n/g,' ')}"`])
  const csv = ['#,URL,Report', ...rows.map(r => r.join(','))].join('\n')
  const a = document.createElement('a')
  a.href = URL.createObjectURL(new Blob(['\uFEFF'+csv], { type:'text/csv;charset=utf-8;' }))
  a.download = `xss_task_${taskId}.csv`; a.click()
}

function statusClass(s) { return { pending:'pending', subdomain_collecting:'running', httpx_probing:'running', xss_scanning:'running', completed:'completed', failed:'failed' }[s] || 'pending' }
function statusText(s) { return { pending:'Pending', subdomain_collecting:'Collecting', httpx_probing:'Probing', xss_scanning:'Scanning', completed:'Completed', failed:'Failed' }[s] || s }
function formatTime(t) { return t ? new Date(t).toLocaleString('en-US', { hour12:false, month:'short', day:'numeric', hour:'2-digit', minute:'2-digit' }) : '-' }

onMounted(() => { loading.value = true; load().finally(() => loading.value = false); timer = setInterval(load, 5000) })
onUnmounted(() => { if (timer) clearInterval(timer) })
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
.stats-row { display:grid; grid-template-columns:repeat(3,1fr); gap:16px; }
.sc { display:flex; align-items:center; gap:14px; padding:20px; background:var(--bg-card); border:1px solid var(--border-color); border-radius:12px; }
.si { width:40px; height:40px; border-radius:10px; display:flex; align-items:center; justify-content:center; }
.si-sub { background:rgba(99,102,241,.15); color:#818cf8; }
.si-alive { background:rgba(16,185,129,.15); color:#34d399; }
.si-xss { background:rgba(239,68,68,.15); color:#f87171; }
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
.rc { font-size:12px; color:var(--text-muted); }
.ulink { color:var(--accent); text-decoration:none; word-break:break-all; font-size:13px; }
.ulink:hover { text-decoration:underline; }
.hcode { font-size:12px; font-weight:600; padding:2px 8px; border-radius:4px; }
.c2 { background:rgba(16,185,129,.15); color:#34d399; }
.c3 { background:rgba(245,158,11,.15); color:#fbbf24; }
.c4 { background:rgba(239,68,68,.15); color:#f87171; }

.empty { display:flex; flex-direction:column; align-items:center; justify-content:center; padding:60px 20px; gap:12px; }
.empty p { color:var(--text-muted); font-size:14px; }

.exp-btn { display:inline-flex; align-items:center; gap:6px; padding:6px 14px; border-radius:8px; border:1px solid var(--border-color); background:transparent; color:var(--text-secondary); font-size:13px; cursor:pointer; transition:all .2s; }
.exp-btn:hover { border-color:var(--accent); color:var(--accent); }

/* Vuln list */
.vlist { display:flex; flex-direction:column; gap:8px; }
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
  .pipeline { flex-wrap:wrap; gap:8px; }
}
</style>
