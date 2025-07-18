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
      'text-neutral-950',
      'font-semibold',
      'bg-transparent',
      'outline-2',
      'outline-neutral-950',
      'w-64',
      'px-4',
      'py-2',
      'cursor-pointer',
      'rounded-lg'
    ]"
  >
    <slot />
  </button>
</template>
