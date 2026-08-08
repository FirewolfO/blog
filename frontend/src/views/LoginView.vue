<script setup lang="ts">
import { ref } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage } from 'element-plus'
import { ArrowRight, EditPen } from '@element-plus/icons-vue'
import { apiMessage, blogApi } from '@/api'
const loading = ref(false)
const route = useRoute()
async function login() { loading.value = true; try { sessionStorage.setItem('blog_login_redirect', String(route.query.redirect || '/dashboard')); const result = await blogApi.oauthURL(`${location.origin}/oauth/callback`); location.assign(result.authorizationUrl) } catch (error) { ElMessage.error(apiMessage(error, '无法发起 People 登录')); loading.value = false } }
</script>
<template><main class="login-page"><section class="login-panel"><div class="login-brand"><span class="brand-mark"><el-icon><EditPen /></el-icon></span><span><strong>内部博客</strong><small>Knowledge Studio</small></span></div><div class="login-copy"><p>PEOPLE SSO</p><h1>记录团队的判断与实践</h1><span>使用企业员工身份进入内容工作台。</span></div><el-button type="primary" size="large" :icon="ArrowRight" :loading="loading" @click="login">使用 People 登录</el-button></section></main></template>
