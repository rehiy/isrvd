<script setup>
import { inject, reactive, ref } from 'vue'

import api from '@/service/api.js'
import { APP_ACTIONS_KEY } from '@/store/state.js'

const actions = inject(APP_ACTIONS_KEY)

const loginForm = reactive({
  username: 'admin',
  password: 'admin'
})

const loading = ref(false)

const handleLogin = async () => {
  loading.value = true

  try {
    const data = await api.login(loginForm)
    actions.setAuth(data.payload)

   // loginForm.username = ''
   // loginForm.password = ''
  } catch (err) {
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="login-container">
    <div class="row justify-content-center">
      <div class="col-md-6">
        <div class="card">
          <div class="card-header">
            <h5 class="mb-0">
              <i class="fas fa-sign-in-alt"></i> 用户登录
            </h5>
          </div>
          <div class="card-body">
            <form @submit.prevent="handleLogin">
              <div class="mb-3">
                <label for="username" class="form-label">用户名</label>
                <input type="text" class="form-control" id="username" v-model="loginForm.username" :disabled="loading" required>
              </div>
              <div class="mb-3">
                <label for="password" class="form-label">密码</label>
                <input type="password" class="form-control" id="password" v-model="loginForm.password" :disabled="loading" required>
              </div>
              <button type="submit" class="btn btn-primary w-100" :disabled="loading">
                <i class="fas fa-spinner fa-spin" v-if="loading"></i>
                {{ loading ? '登录中...' : '登录' }}
              </button>
            </form>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.login-container {
  margin-top: 25%;
}
</style>
