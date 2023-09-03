<nav>
	<GameButton button={Button.Up}/>
	<GameButton button={Button.Right}/>
	<GameButton button={Button.Left}/>
	<GameButton button={Button.Down}/>

	<i class="material-icons fill"></i>
</nav>

<script>
	import { Button } from "$lib/consts";
	import GameButton from "../../GameButton.svelte";

</script>

<style lang="scss">
	nav {
		height: 70px;
		transform-origin: top left;
		width: 70px;

		:global(.material-icons) {
			position: absolute;
			cursor: pointer;
			border-radius: 0 0 15px 15px;
		}
		:global(.up) {
			top: 0;
			left: calc(50% - 12px);
			transform: rotate(180deg);
		}
		:global(.down) {
			bottom: 0;
			left: calc(50% - 12px);
		}
		:global(.left) {
			top: calc(50% - 12px);
			left: 0;
			transform: rotate(90deg);
		}
		:global(.right) {
			top: calc(50% - 12px);
			right: 0;
			transform: rotate(270deg);
		}

		.fill {
			position: absolute;
			top: calc(50% - 12px);
			left: calc(50% - 12px);
			background-color: blueviolet;
			width: 24px;
			height: 24px;
			border-radius: 0;
		}
	}
</style>