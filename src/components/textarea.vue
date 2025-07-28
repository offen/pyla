<script setup>
const model = defineModel()
const { label, placeholder, rows = 4, type = 'light' } = defineProps(['label', 'placeholder', 'rows', 'type'])
</script>

<template>
  <label
    class="block mb-0 text-neutral-500"
    :class="[type === 'dark' ? 'font-mono' : '' ]"
  >
    {{ label }}
    <textarea
      v-bind="$attrs"
      v-model="model"
      class="block w-full rounded-lg p-3 focus:outline-none focus:ring-2 "
      :class="[
        type === 'light' ? 'focus:ring-neutral-400 bg-neutral-50 h-40' : '',
        type === 'dark' ? 'focus:ring-neutral-950 text-neutral-300 bg-neutral-700 mt-2 h-40' : '',
        type === 'lightinput' ? 'focus:ring-neutral-400 font-semibold text-neutral-950 bg-neutral-50 h-[6rem]': ''
      ]"
      :rows="rows"
      :placeholder="placeholder"
    ></textarea>
  </label>
</template>
