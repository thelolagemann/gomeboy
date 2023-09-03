<div
	bind:this={scaleContainer}
	style="height: {$adjustedHeight}px; width: {$adjustedWidth}px; {relative ? 'position: relative' : ''}">
	<slot></slot>
</div>

<script>
	import {derived, get} from "svelte/store";
	import { scale } from "./scaler.js";
	import {afterUpdate, beforeUpdate, onMount} from "svelte";

	export let height;
	export let width;

	export let relative = false;

	let adjustedHeight = derived(scale, ($scale) => {
		return $scale * height;
	})
	let adjustedWidth = derived(scale, ($scale) => {
		return $scale * width
	})
	let scaleContainer;

	afterUpdate(() => {
		scaleContainer
			.querySelectorAll(':scope > *')
			.forEach(n => n.style.transform = `scale(${get(scale)}`)
	})

</script>

<style lang="scss">
	div {
		transform-origin: top left;
		image-rendering: pixelated;
		overflow: hidden;

		:global(>*) {
			transform: scale(3);
		}
	}
</style>