<nav>
	<h2>GomeBoy</h2>
	<div>
		<svg height="20" width="20">
			<circle cx="10" cy="10" r="10" fill="{$connected ? 'green' : 'red'}"/>
		</svg>
		<span>
			{$connected ? Game.socket.url.substring(5, Game.socket.url.length-1) :'Not connected'}
			<Switch name="admin" bind:value={$adminView} text="Admin"/>
		</span>
	</div>
</nav>

<script>
	import Game, { adminView } from "$lib/game";
	import Switch from "./inputs/Switch.svelte";

	let { connected } = Game.socket;
</script>

<style lang="scss">
	nav {
		background-color: blueviolet;
		color: white;
		display: flex;
		height: 48px;
		line-height: 48px;
		padding: 0 16px;
		width: 100%;
		justify-content: space-between;

		> div {
			font-family:  monospace, mono, monospaced
		}
	}
</style>