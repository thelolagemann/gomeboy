<div>
	{#if header !== ""}
		<h2>
			{header}
		</h2>
	{/if}
	<span class:full-height={fullHeight}>
		<slot></slot>
	</span>
</div>

<script>
	export let header = "";
	export let fullHeight = false;
</script>

<style lang="scss">
	@use "sass:color";
	div {
		background-color: #2b2b2b;
		border-radius: 4px;
		overflow: hidden;
		box-shadow: 0 1px 2px 0 rgba(0, 0, 0, .35);

		> h2 {
			background-color: color.scale(black, $whiteness: +50%);
			display: flex;
			justify-content: space-between;
			letter-spacing: 1.5px;
			line-height: 24px;
			padding: 4px 8px;
			font-size: 18px;
		}

		span {
			display: grid;
			gap: 4px;
			padding: 4px 8px;

			&.full-height {
				height: calc(100% - 32px);
			}
		}
	}
</style>