<div class:hide={!show}>
	<slot/>
</div>

<script>
	export let show = true;
</script>

<style>
	.hide {
		display: none !important;
	}
</style>