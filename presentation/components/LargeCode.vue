<template>
  <pre class="large-code"><code v-html="content"></code></pre>
</template>

<script>
export default {
  name: 'LargeCode',
  computed: {
    content() {
      return this.$slots.default[0].text.trim()
    }
  }
}
</script>

<style scoped>
.large-code {
  font-size: 1.2em;
  font-family: monospace;
  line-height: 1.4;
  background-color: #f5f5f5;
  padding: 1rem;
  border-radius: 4px;
  overflow-x: auto;
  white-space: pre;
}

.large-code code {
  background: none;
  padding: 0;
  white-space: pre;
}
</style>
