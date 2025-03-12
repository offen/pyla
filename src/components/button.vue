<script setup>
import { ref } from 'vue'

const clicked = ref(false)

const handleClick = () => {
  clicked.value = true
}
</script>

<template>
  <button
    v-bind="$attrs"
    @click="handleClick"
    :class="[
      clicked ? 'border-2' : 'border-1',
      'p-2',
      'cursor-pointer',
      'rounded-md'
    ]"
  >
    <slot />
  </button>
</template>
