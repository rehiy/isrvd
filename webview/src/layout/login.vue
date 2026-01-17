<script setup>
import { inject, reactive } from 'vue'

import api from '@/service/api.js'
import { APP_STATE_KEY, APP_ACTIONS_KEY } from '@/store/state.js'

const state = inject(APP_STATE_KEY)
const actions = inject(APP_ACTIONS_KEY)

const loginForm = reactive({
  username: '',
  password: ''
})

const handleLogin = async () => {
    const data = await api.login(loginForm)
    actions.setAuth(data.payload)

    loginForm.username = ''
    loginForm.password = ''
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
                <input type="text" class="form-control" id="username" v-model="loginForm.username" required>
              </div>
              <div class="mb-3">
                <label for="password" class="form-label">密码</label>
                <input type="password" class="form-control" id="password" v-model="loginForm.password" required>
              </div>
              <button type="submit" class="btn btn-primary w-100" :disabled="state.loading">
                <i class="fas fa-spinner fa-spin" v-if="state.loading"></i>
                {{ state.loading ? '登录中...' : '登录' }}
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
