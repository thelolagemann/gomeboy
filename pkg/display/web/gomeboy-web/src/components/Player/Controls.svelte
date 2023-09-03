<div>
	<Scaler height=70 width=70>
		<DPad/>
	</Scaler>
	<Scaler height=70 width=70>
		<nav class="pad" style="transform-origin: top left;">
			<GameButton button={Button.A}/>
			<GameButton button={Button.B}/>
		</nav>
	</Scaler>
</div>

<script>
	import {Button} from "$lib/consts";
	import DPad from "./Buttons/DPad.svelte";
	import GameButton from "../GameButton.svelte";
	import Scaler from "../Scaler/Scaler.svelte";
</script>

<style lang="scss">
	div {
		display: flex;
		justify-content: space-between;
	}
</style>