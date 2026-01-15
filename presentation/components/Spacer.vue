<template>
  <div class="spacer" :style="{ height: size }"></div>
</template>

<script>
export default {
  name: 'Spacer',
  props: {
    size: {
      type: String,
      default: '1rem'
    }
  }
}
</script>

<style scoped>
.spacer {
  width: 100%;
}
</style>
