import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import { blogApi, clearToken, getToken, saveToken } from '@/api'
import type { User } from '@/types'

export const useAuthStore = defineStore('auth', () => {
  const user = ref<User | null>(null)
  const initialized = ref(false)
  const authenticated = computed(() => Boolean(user.value))
  async function hydrate() { if (initialized.value) return; if (!getToken()) { initialized.value = true; return }; try { user.value = (await blogApi.me()).user } catch { clearToken() } finally { initialized.value = true } }
  async function completeOAuth(code: string, state: string) { const result = await blogApi.oauthCallback(code, state, `${location.origin}/oauth/callback`); saveToken(result.accessToken); user.value = result.user; initialized.value = true }
  async function logout() { try { await blogApi.logout() } finally { clearToken(); user.value = null } }
  return { user, initialized, authenticated, hydrate, completeOAuth, logout }
})
