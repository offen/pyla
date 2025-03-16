<script setup>
const model = defineModel()
const { label, placeholder, rows = 4} = defineProps(['label', 'placeholder', 'rows'])
</script>

<template>
  <label class="block mb-2 text-sm font-medium text-gray-900 dark:text-white">
    {{ label }}
    <textarea
      v-bind="$attrs"
      v-model="model"
      class="block border-1"
      :rows="rows"
      :placeholder="placeholder"
    ></textarea>
  </label>
</template>
