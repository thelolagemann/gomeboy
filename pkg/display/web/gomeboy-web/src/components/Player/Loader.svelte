<span>
	<div></div>
</span>

<style lang="scss">
	@use "sass:color";

	span {
		background-color: #2b2b2b;
		margin: 0 auto;

		div {
			animation-duration: 2.5s;
			animation-fill-mode: forwards;
			animation-iteration-count: infinite;
			animation-name: shimmer;
			animation-timing-function: linear;
			height: 480px;
			position: relative;
		}
	}

	@keyframes shimmer {
		0% {
			background-color: color.scale(#2b2b2b, $whiteness: +20%);
		}
		50% {
			background-color: color.scale(#2b2b2b, $whiteness: 10%);
		}
		100% {
			background-color: color.scale(#2b2b2b, $whiteness: +20%);
		}
	}
</style>