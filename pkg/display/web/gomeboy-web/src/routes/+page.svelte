<Heading/>
<main>
	{#if $adminView}
		<div transition:fly={{x: -200}} style="display: flex; gap: 8px; flex-direction: row">
			<div style="display: grid; gap: 8px;">
				<Card header="Server settings">
					<ListHeading heading="playback" icon="play_arrow"/>
					<span style="display: flex; justify-content: space-between">
					<label for="speed">Speed {speed}x</label><br/>
					<input type="range" id="speed" name="speed" min="1" max="8" value="1">
				</span>

					<ListHeading heading="streaming" icon="build"/>
					<Switch name="compression" bind:value={$compression} text="Compression">
						{#if $compression}
						<span class="suffix" style="float: right" transition:fade={{duration: 150}}>
							{compressionLevelMap[$compressionLevel]}
						</span>
						{/if}
					</Switch>
					{#if $compression}
						<input
							type="range"
							min="1" max="4"
							transition:fadeSlide={{duration: 150}}
							bind:value={$compressionLevel}
							style="transform: rotateY(180deg)"
						/>
					{/if}
					<Switch name="framePatch" bind:value={$framePatching} text="Frame Patching"/>
					<Switch name="frameSkip" bind:value={$frameSkipping} text="Frame Skipping"/>
					<Switch name="frameCache" bind:value={$frameCaching} text="Frame Caching"/>
					<ListHeading heading="graphics" icon="video_settings"/>

				</Card>
			</div>
			<Card header="Clients">
				<ClientList/>
			</Card>
		</div>
	{/if}
	{#if $connected}
		<Player player="{Game.player1}" controls="{$isPlayer1}"/>
		<Show show="{$p2Running}">
			<Player player="{Game.player2}" controls="{$isPlayer2}"/>
		</Show>
		{#if !$isPlayer1 && !$isPlayer2 && !userView && !$p2Running}
			<Card>
				<div class="content" style="display: grid;">
					Would you like to join as player 2 or watch as a viewer?

					<button on:click={() => Game.send(ControlEvent.Info, InfoControlEvent.Player2Confirmation)}>Join as player 2</button>
					<button on:click={() => userView = true}>Watch as viewer</button>
				</div>
			</Card>
		{/if}


	{/if}
	<div class="settings" style="display: flex; gap: 8px; flex-direction: row">
		<div class="left" style="display: grid; gap: 8px;">
			<Card header="Users">
				<UserList/>
			</Card>
		</div>
		<div class="right" style="display: grid; gap: 8px;">
			<Card header="Settings">
				<ListHeading heading="player" icon="display_settings"/>
				<span style="display: flex; justify-content: space-between">
						<label for="scale">Scale {$scale}</label>
						<input type="range" id="scale" name="scale" min="1" max="3" bind:value={$scale}/>
					</span>
				<button>edit keybindings</button>
			</Card>
			<Card header="Statistics">
				<ul>
					<li class="list-header">
						Frames
						<hr>
					</li>
					<li>
						<i class="material-icons">skip_next</i>
						<span class="label">Skipped</span>
						<span class="list-content">{$framesSkipped}</span>
					</li>
					<li>
						<i class="material-icons">difference</i>
						<span class="label">Patched</span>
						<span class="list-content">{$framesPatched}</span>
					</li>
					<li class="list-header">
						Network
						<hr>
					</li>
					<li>
						<i class="material-icons">import_export</i>
						<span class="label">Transfer Rate</span>
						<span class="list-content">{humanFileSize($transferRate)}/s</span>
					</li>
					<li>
						<i class="material-icons">file_download</i>
						<span class="label">Transferred</span>
						<span class="list-content">{humanFileSize($networkTransferred)}</span>
					</li>
					<li>
						<i class="material-icons">expand</i>
						<span class="label">Uncompressed</span>
						<span class="list-content">{humanFileSize($gameThroughput)}</span>
					</li>
				</ul>
				<SpeedGraph />
			</Card>
			<Card header="Cache">
				<CacheList player={player1} title="frames" />
				<CacheList player={player2} title="patches" />
			</Card>
		</div>
	</div>
</main>

<script>
	import { browser } from "$app/environment";
	import { fadeSlide } from "$lib/animations";
	import { compressionLevelMap } from "$lib/consts";
	import Game, {adminView, ControlEvent, InfoControlEvent} from "$lib/game";
	import { humanFileSize } from "$lib/utils";

	import {onDestroy, onMount} from "svelte";
	import { fade, fly } from "svelte/transition";
	import CacheList from "../components/CacheList.svelte";
	import ClientList from "../components/ClientList.svelte";
	import UserList from "../components/UserList.svelte";
	import Switch from "../components/inputs/Switch.svelte";
	import Card from "../components/Card.svelte";
	import ListHeading from "../components/ListHeading.svelte";
	import SpeedGraph from "../components/SpeedGraph.svelte";
	import {readable} from "svelte/store";
	import { scale } from "../components/Scaler/scaler.js";
	import Player from "../components/Player/Player.svelte";
	import Heading from "../components/Header.svelte";
	import Show from "../components/Show.svelte";

	let {
		compression, compressionLevel,
		frameCaching, framePatching, frameSkipping,
		framesSkipped, framesPatched,
		bgEnabled, windowEnabled, spritesEnabled,
		paused,
		throughput, transferRate,
		player1, player2, isPlayer1, isPlayer2 } = Game,
		{ connected, transferred } = Game.socket;

	let { isRunning: p1Running } = player1;
	let { isRunning: p2Running } = player2;

	let speed = 1;
	let userView = false;

	let networkTransferred;
	let gameThroughput;

	if (browser) {
		onMount(async () => {
			networkTransferred = readable(0, set => {
				let interval = setInterval(() => transferred.subscribe(t => set(t))(), 1000)
				return () => clearInterval(interval)

			})

			gameThroughput = readable(0, set => {
				let interval = setInterval(() => throughput.subscribe(t => set(t))(), 1000)
				return () => clearInterval(interval)
			})

			await Game.init(null, null, null)

		})

		onDestroy(() => {
			Game.close()
		})
	}

	function changeSpeed(event) {
		speed = event.target.value
	}
</script>
