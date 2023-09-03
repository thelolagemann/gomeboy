<i class="material-icons {ButtonClassMap[button]}"
	on:mousedown={() => Game.press(button)}
	on:mouseup={() => Game.release(button)}
	on:touchstart={() => Game.press(button)}
	on:touchend={() => Game.release(button)} >
	{ButtonIconMap[button]}
</i>

<script>
	import { Button, ButtonClassMap, ButtonIconMap } from "$lib/consts";
	import Game from "$lib/game";

	export let button;
</script>