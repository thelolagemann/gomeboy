<div>
	<span>{heading}<i class="material-icons">{icon}</i></span>
	<hr>
</div>

<script>
	export let heading;
	export let icon;
</script>

<style lang="scss">
	div {
		display: inline;
		padding: 2px 0;

		hr {
			border-color: gray;
		}

		span {
			font-size: 12px;
			display: flex;
			justify-content: space-between;
			line-height: 24px;
			font-family: monospace, mono, monospaced;
			letter-spacing: 2px;
			text-transform: uppercase;
		}

		i {
			background-color: transparent;
			font-size: 20px;
			line-height: 24px;
		}
	}
</style>