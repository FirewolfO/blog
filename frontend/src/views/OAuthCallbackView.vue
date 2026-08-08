<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { Loading } from '@element-plus/icons-vue'
import { apiMessage } from '@/api'
import { useAuthStore } from '@/stores/auth'
const route = useRoute(); const router = useRouter(); const auth = useAuthStore(); const error = ref('')
onMounted(async () => { const code = String(route.query.code || ''); const state = String(route.query.state || ''); if (!code || !state) { error.value = 'OAuth 回调参数不完整'; return }; try { await auth.completeOAuth(code, state); await router.replace(String(sessionStorage.getItem('blog_login_redirect') || '/dashboard')) } catch (reason) { error.value = apiMessage(reason, 'People 登录失败') } })
</script>
<template><main class="callback-page"><el-result v-if="error" icon="error" title="登录失败" :sub-title="error"><template #extra><el-button type="primary" @click="$router.replace('/login')">重新登录</el-button></template></el-result><div v-else class="callback-loading"><el-icon class="is-loading" :size="32"><Loading /></el-icon><strong>正在完成登录</strong></div></main></template>
