<template>
  <div class="app">
    <!-- Login Dialog -->
    <el-dialog v-model="showLogin" title="认证" width="400px" :close-on-click-modal="false" :show-close="false">
      <el-form @submit.prevent="handleLogin">
        <el-form-item label="Token">
          <el-input v-model="tokenInput" placeholder="请输入访问Token" show-password />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="handleLogin" style="width: 100%">登录</el-button>
        </el-form-item>
      </el-form>
    </el-dialog>

    <!-- Main Layout -->
    <el-container v-if="isAuthenticated" class="main-container">
      <el-header class="header">
        <div class="header-left">
          <el-icon :size="24" color="#409eff"><Monitor /></el-icon>
          <h1>XScan Web</h1>
          <el-tag type="info" size="small">XSS扫描管理平台</el-tag>
        </div>
        <div class="header-right">
          <el-button text @click="handleLogout">
            <el-icon><SwitchButton /></el-icon>
            退出
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

const showLogin = ref(false)
const isAuthenticated = ref(false)
const tokenInput = ref('')

onMounted(() => {
  const token = getToken()
  if (token) {
    isAuthenticated.value = true
  } else {
    showLogin.value = true
  }
})

function handleLogin() {
  if (!tokenInput.value.trim()) {
    return
  }
  setToken(tokenInput.value.trim())
  isAuthenticated.value = true
  showLogin.value = false
}

function handleLogout() {
  localStorage.removeItem('xscan_token')
  isAuthenticated.value = false
  showLogin.value = true
  tokenInput.value = ''
}
</script>

<style>
* {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}

body {
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
  background-color: #f0f2f5;
}

.app {
  min-height: 100vh;
}

.main-container {
  min-height: 100vh;
}

.header {
  background: #fff;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 24px;
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.08);
  z-index: 10;
}

.header-left {
  display: flex;
  align-items: center;
  gap: 12px;
}

.header-left h1 {
  font-size: 20px;
  font-weight: 600;
  color: #303133;
}

.main-content {
  padding: 24px;
  max-width: 1400px;
  margin: 0 auto;
  width: 100%;
}
</style>
