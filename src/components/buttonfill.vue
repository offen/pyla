<script setup>
import { ref, useAttrs } from 'vue'

const clicked = ref(false)
const $attrs = useAttrs()

const handleClick = (event) => {
  if ($attrs.disabled) {
    event.preventDefault()
    event.stopPropagation()
    return
  }
  clicked.value = true
}
</script>

<template>
  <button
    v-bind="$attrs"
    @click="handleClick"
    :class="[
      'text-white',
      'bg-neutral-950',
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