<script setup lang="ts">
import { ref } from 'vue'

const emit = defineEmits<{ select: [emoji: string] }>()
const visible = ref(false)
const emojis = ['😀', '😄', '😂', '🥳', '👍', '👏', '❤️', '🔥', '🎉', '🤔', '💡', '✅']

function select(emoji: string) {
  emit('select', emoji)
  visible.value = false
}
</script>

<template>
  <el-popover v-model:visible="visible" placement="top-start" :width="232" trigger="click">
    <div class="emoji-grid" role="list" aria-label="常用表情">
      <button v-for="emoji in emojis" :key="emoji" type="button" :aria-label="`插入 ${emoji}`" @click="select(emoji)">{{ emoji }}</button>
    </div>
    <template #reference>
      <el-button class="emoji-trigger" aria-label="添加表情" title="添加表情"><span aria-hidden="true">😀</span></el-button>
    </template>
  </el-popover>
</template>
