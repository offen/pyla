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
      clicked ? 'bg-transparent' : 'bg-neutral-950',
      clicked ? 'outline-3' : 'outline-0',
      'outline-neutral-950',
      'w-48',
      'px-4',
      'py-2',
      'cursor-pointer',
      'rounded-lg'
    ]"
  >
    <slot />
  </button>
</template>
