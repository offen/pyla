<script setup>
const model = defineModel()
const { label, placeholder, rows = 4} = defineProps(['label', 'placeholder', 'rows'])
</script>

<template>
  <label class="block mb-0 text-neutral-500">
    {{ label }}
    <textarea
      v-bind="$attrs"
      v-model="model"
      class="h-40 block w-full rounded-lg p-3 focus:outline-none focus:ring-2 focus:ring-neutral-400 bg-neutral-50"
      :rows="rows"
      :placeholder="placeholder"
    ></textarea>
  </label>
</template>
