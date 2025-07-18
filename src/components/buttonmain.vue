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
      clicked ? 'text-neutral-950' : 'text-white',
      clicked ? 'bg-transparent' : 'bg-neutral-950',
      'outline-neutral-950',
      'disabled:bg-neutral-400',
      'disabled:outline-neutral-400',
      'outline-2',
      'font-semibold',
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
