<template>
  <div class="app">
    <!-- Login Page -->
    <div v-if="!isAuthenticated" class="login-page">
      <div class="login-bg"></div>
      <div class="login-card">
        <div class="login-logo">
          <div class="logo-icon">
            <svg viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
              <path d="M12 2L2 7V17L12 22L22 17V7L12 2Z" stroke="#00d4ff" stroke-width="1.5" fill="none"/>
              <path d="M12 6L6 9V15L12 18L18 15V9L12 6Z" fill="#00d4ff" opacity="0.3"/>
              <circle cx="12" cy="12" r="2" fill="#00d4ff"/>
            </svg>
          </div>
          <h1>XScan Web</h1>
          <p>XSS Vulnerability Scanner</p>
        </div>
        <el-form @submit.prevent="handleLogin" class="login-form">
          <el-form-item>
            <el-input
              v-model="tokenInput"
              placeholder="Enter access token"
              show-password
              size="large"
              @keyup.enter="handleLogin"
              prefix-icon="Lock"
            />
          </el-form-item>
          <el-button type="primary" size="large" @click="handleLogin" class="login-btn">
            Sign In
          </el-button>
        </el-form>
      </div>
    </div>

    <!-- Main Layout -->
    <el-container v-else class="main-container">
      <el-header class="header">
        <div class="header-left">
          <div class="header-logo">
            <svg viewBox="0 0 24 24" fill="none" width="28" height="28">
              <path d="M12 2L2 7V17L12 22L22 17V7L12 2Z" stroke="#00d4ff" stroke-width="1.5" fill="none"/>
              <path d="M12 6L6 9V15L12 18L18 15V9L12 6Z" fill="#00d4ff" opacity="0.3"/>
              <circle cx="12" cy="12" r="2" fill="#00d4ff"/>
            </svg>
          </div>
          <h1>XScan</h1>
        </div>
        <div class="header-right">
          <el-button text class="logout-btn" @click="handleLogout">
            <el-icon><SwitchButton /></el-icon>
            Logout
          </el-button>
        </div>
      </el-header>
      <el-main class="main-content">
        <router-view />
      </el-main>
    </el-container>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { setToken, getToken } from './api'

const isAuthenticated = ref(false)
const tokenInput = ref('')

onMounted(() => {
  if (getToken()) {
    isAuthenticated.value = true
  }
})

function handleLogin() {
  if (!tokenInput.value.trim()) return
  setToken(tokenInput.value.trim())
  isAuthenticated.value = true
}

function handleLogout() {
  localStorage.removeItem('xscan_token')
  isAuthenticated.value = false
  tokenInput.value = ''
}
</script>

<style>
:root {
  --bg-primary: #0a0e17;
  --bg-secondary: #111827;
  --bg-card: #1a2332;
  --bg-card-hover: #1f2b3d;
  --bg-input: #0d1321;
  --border-color: #1e2d3d;
  --border-light: #263548;
  --text-primary: #e2e8f0;
  --text-secondary: #94a3b8;
  --text-muted: #64748b;
  --accent: #00d4ff;
  --accent-hover: #00b8e6;
  --accent-glow: rgba(0, 212, 255, 0.15);
  --success: #10b981;
  --warning: #f59e0b;
  --danger: #ef4444;
  --info: #6366f1;
}

* {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}

body {
  font-family: 'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
  background-color: var(--bg-primary);
  color: var(--text-primary);
  -webkit-font-smoothing: antialiased;
}

/* Override Element Plus for dark theme */
.el-card {
  background-color: var(--bg-card) !important;
  border: 1px solid var(--border-color) !important;
  color: var(--text-primary) !important;
  border-radius: 12px !important;
}

.el-card:hover {
  border-color: var(--border-light) !important;
}

.el-card__header {
  border-bottom: 1px solid var(--border-color) !important;
  color: var(--text-primary) !important;
}

.el-table {
  background-color: transparent !important;
  color: var(--text-primary) !important;
  --el-table-bg-color: transparent;
  --el-table-tr-bg-color: transparent;
  --el-table-header-bg-color: rgba(0, 212, 255, 0.05);
  --el-table-row-hover-bg-color: rgba(0, 212, 255, 0.08);
  --el-table-border-color: var(--border-color);
  --el-table-text-color: var(--text-primary);
  --el-table-header-text-color: var(--text-secondary);
}

.el-table__body tr.el-table__row--striped td {
  background: rgba(255, 255, 255, 0.02) !important;
}

.el-table th.el-table__cell {
  background-color: rgba(0, 212, 255, 0.05) !important;
}

.el-input__wrapper {
  background-color: var(--bg-input) !important;
  border-color: var(--border-color) !important;
  box-shadow: none !important;
}

.el-input__wrapper:hover,
.el-input__wrapper.is-focus {
  border-color: var(--accent) !important;
  box-shadow: 0 0 0 1px var(--accent) !important;
}

.el-input__inner {
  color: var(--text-primary) !important;
}

.el-input__inner::placeholder {
  color: var(--text-muted) !important;
}

.el-select .el-input__wrapper {
  background-color: var(--bg-input) !important;
}

.el-dialog {
  background-color: var(--bg-card) !important;
  border: 1px solid var(--border-color) !important;
  border-radius: 16px !important;
}

.el-dialog__title {
  color: var(--text-primary) !important;
}

.el-dialog__header {
  border-bottom: 1px solid var(--border-color) !important;
}

.el-form-item__label {
  color: var(--text-secondary) !important;
}

.el-tabs__nav-wrap::after {
  background-color: var(--border-color) !important;
}

.el-tabs__item {
  color: var(--text-muted) !important;
}

.el-tabs__item.is-active {
  color: var(--accent) !important;
}

.el-tabs__active-bar {
  background-color: var(--accent) !important;
}

.el-steps .el-step__title {
  color: var(--text-muted) !important;
}

.el-steps .el-step__title.is-finish {
  color: var(--success) !important;
}

.el-steps .el-step__title.is-process {
  color: var(--accent) !important;
}

.el-steps .el-step__description {
  color: var(--text-muted) !important;
}

.el-collapse {
  border: none !important;
}

.el-collapse-item__header {
  background: transparent !important;
  color: var(--text-primary) !important;
  border-bottom: 1px solid var(--border-color) !important;
}

.el-collapse-item__wrap {
  background: transparent !important;
  border-bottom: 1px solid var(--border-color) !important;
}

.el-collapse-item__content {
  color: var(--text-secondary) !important;
}

.el-alert--error.is-light {
  background: rgba(239, 68, 68, 0.1) !important;
  color: #fca5a5 !important;
  border: 1px solid rgba(239, 68, 68, 0.3) !important;
}

.el-popconfirm {
  background-color: var(--bg-card) !important;
}

.el-button--primary {
  --el-button-bg-color: var(--accent);
  --el-button-border-color: var(--accent);
  --el-button-hover-bg-color: var(--accent-hover);
  --el-button-hover-border-color: var(--accent-hover);
}

.el-loading-mask {
  background-color: rgba(10, 14, 23, 0.8) !important;
}

/* Scrollbar */
::-webkit-scrollbar {
  width: 6px;
  height: 6px;
}

::-webkit-scrollbar-thumb {
  background: var(--border-light);
  border-radius: 3px;
}

::-webkit-scrollbar-track {
  background: transparent;
}

/* App styles */
.app {
  min-height: 100vh;
}

/* Login Page */
.login-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  position: relative;
  overflow: hidden;
}

.login-bg {
  position: absolute;
  inset: 0;
  background:
    radial-gradient(ellipse at 20% 50%, rgba(0, 212, 255, 0.08) 0%, transparent 50%),
    radial-gradient(ellipse at 80% 20%, rgba(99, 102, 241, 0.08) 0%, transparent 50%),
    radial-gradient(ellipse at 50% 80%, rgba(239, 68, 68, 0.05) 0%, transparent 50%);
}

.login-card {
  position: relative;
  background: var(--bg-card);
  border: 1px solid var(--border-color);
  border-radius: 20px;
  padding: 48px 40px;
  width: 400px;
  backdrop-filter: blur(20px);
}

.login-logo {
  text-align: center;
  margin-bottom: 36px;
}

.logo-icon {
  width: 64px;
  height: 64px;
  margin: 0 auto 16px;
}

.logo-icon svg {
  width: 100%;
  height: 100%;
}

.login-logo h1 {
  font-size: 28px;
  font-weight: 700;
  color: var(--text-primary);
  letter-spacing: -0.5px;
}

.login-logo p {
  font-size: 14px;
  color: var(--text-muted);
  margin-top: 4px;
}

.login-form .el-form-item {
  margin-bottom: 24px;
}

.login-btn {
  width: 100%;
  height: 44px;
  font-size: 15px;
  font-weight: 600;
  border-radius: 10px;
  background: linear-gradient(135deg, #00d4ff, #6366f1) !important;
  border: none !important;
}

.login-btn:hover {
  opacity: 0.9;
}

/* Main container */
.main-container {
  min-height: 100vh;
  display: flex;
  flex-direction: column;
}

.header {
  background: var(--bg-secondary);
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 24px;
  border-bottom: 1px solid var(--border-color);
  height: 56px;
}

.header-left {
  display: flex;
  align-items: center;
  gap: 10px;
}

.header-logo {
  display: flex;
}

.header-left h1 {
  font-size: 18px;
  font-weight: 700;
  color: var(--text-primary);
  letter-spacing: -0.3px;
}

.logout-btn {
  color: var(--text-muted) !important;
}

.logout-btn:hover {
  color: var(--text-primary) !important;
}

.main-content {
  padding: 24px;
  max-width: 1400px;
  margin: 0 auto;
  width: 100%;
  flex: 1;
}
</style>
