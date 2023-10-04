<div class="player" style="position: relative">
	<div class="card game-player">
		<div class="content">
			<Show show={$running}>
				<Scaler relative height="144" width="160">
					<canvas bind:this={canvasElement} tabindex="1" transition:fade={{duration: 150}}></canvas>
					{#if $paused}
						<div class="pause-overlay" transition:fade={{duration: 150}}>
							<span>Paused</span>
						</div>
					{/if}
				</Scaler>
				{#if controls}
					<Controls/>
					<Scaler height="24" width="160">
						<nav class="pad pad-actions" style="transform-origin: top left; width: 100%">
							<button class="select actions" >SELECT</button>
							<button class="start actions">START</button>
						</nav>
					</Scaler>
					<div class="btn-row">
						<i class="material-icons" on:click={() => player.togglePlayback()}>
							{#if $paused}
								play_arrow
							{:else}
								pause
							{/if}
						</i>
					</div>
				{/if}
			</Show>
		</div>
	</div>
</div>

<script>
	import {EventType, Player} from "$lib/game";

	import { fade } from "svelte/transition";
	import { onDestroy, onMount, tick } from "svelte";
	import { browser } from "$app/environment";
	import Loader from "./Loader.svelte";
	import Scaler from "../Scaler/Scaler.svelte";
	import Controls from "./Controls.svelte";
	import Show from "../Show.svelte";

	let { paused, running } = false;
	let { connected } = false;

	let canvasElement;
	let canvasCtx;
	let canvasImg;

	export let player;
	export let controls;


	if (browser) {
		onMount(async () => {
			await tick() // wait for canvas element to render

			canvasElement.width = 160
			canvasElement.height = 144

			canvasCtx = canvasElement.getContext("2d")
			canvasCtx.imageSmoothingEnabled = false
			canvasImg = canvasCtx.createImageData(160, 144)

			player.init((event, data) => {
				switch (event) {
					case EventType.Frame:
						canvasImg.data.set(data)
						canvasCtx.putImageData(canvasImg, 0, 0)
						break
					case EventType.FramePatch:
						// frame patch
						let changedPixels = 0

						for (let i = 0; i < data.length; i += 4) {
							if (data[i+3] === 255) {
								canvasImg.data.set([data[i], data[i+1], data[i+2], 255], i)
								changedPixels++
							}
						}
						canvasCtx.putImageData(canvasImg, 0, 0)
						break
					default:
						console.log(`unknown event type ${event} ${data}`)
				}
			})



		})
	}
</script>

<style lang="scss">
	.player {
		overflow: hidden;
		position: relative;
	}
	:global(.pause-overlay) {
		position: absolute;
		top: 0;
		left: 0;
		font-size: 32px;
		height: 100%;
		width: 100%;
		background: rgba(0, 0, 0, 0.75);
		text-align: center;
		display: flex;
		align-items: center;
		justify-content: center;
	}
</style>