<nav class="header">
	<h2>GomeBoy</h2>
	<div class="headline">
		<svg height="20" width="20">
			<circle cx="10" cy="10" r="10" fill="{connected ? 'green' : 'red'}"/>
		</svg>
		<span>{connected ? gameSocket.url.substring(5, gameSocket.url.length-1) :'Not connected'}</span>
	</div>
</nav>
<main style="display: flex; justify-content: space-between">
	<div class="card">
		<div class="headline">
			Graph
		</div>
		<div class="content">
			<ul>
				<li>something</li>
			</ul>
		</div>
	</div>
	<div class="player">
		<div class="card game-player">
			<div class="content">

				<div class="scaler player-{scale}">
					{#if connected}
						<canvas class="scale-{scale}" bind:this={canvasElement} tabindex="1" transition:fade={{duration: 150}}></canvas>
						{#if paused}
							<div class="pause-overlay" transition:fade={{duration: 150}}>
								<span>Paused</span>
							</div>
						{/if}
					{:else}
						<div class="placeholder">
							<div class="animated-background"></div>
						</div>
					{/if}
				</div>
				<div class="player-controls" style="display: flex">
					<div class="scaler btn-{scale}">
						<nav class="pad scale-{scale}" style="transform-origin: top left">
							<i class="up material-icons " on:mousedown={dir} on:mouseup={unDir} on:touchstart={dir} on:touchend={unDir}>arrow_drop_down</i>
							<i class="right material-icons" on:mousedown={dir} on:mouseup={unDir} on:touchstart={dir} on:touchend={unDir}>arrow_drop_down</i>
							<i class="left material-icons" on:mousedown={dir} on:mouseup={unDir} on:touchstart={dir} on:touchend={unDir}>arrow_drop_down</i>
							<i class="down material-icons" on:mousedown={dir} on:mouseup={unDir} on:touchstart={dir} on:touchend={unDir}>arrow_drop_down</i>
							<i class="material-icons fill"></i>
						</nav>
					</div>
					<div class="scaler btn-{scale}">
						<nav class="pad scale-{scale}" style="transform-origin: top left;">
							<i class="a material-icons" on:mousedown={dir} on:mouseup={unDir} on:touchstart={dir} on:touchend={unDir}>A</i>
							<i class="b material-icons" on:mousedown={dir} on:mouseup={unDir} on:touchstart={dir} on:touchend={unDir}>B</i>
						</nav>
					</div>
				</div>
				<nav class="pad pad-actions scale-{scale}" style="transform-origin: top left; width: 100%">
					<button class="select actions" on:mousedown={dir} on:mouseup={unDir} on:touchstart={dir} on:touchend={unDir}>SELECT</button>
					<button class="start actions" on:mousedown={dir} on:mouseup={unDir} on:touchstart={dir} on:touchend={unDir}>START</button>
				</nav>
			</div>
		</div>
	</div>
	<div class="settings" style="display: flex; gap: 8px; flex-direction: row">
		<div class="left" style="display: grid; gap: 8px;">
			<div class="card">
				<div class="headline">
					Settings
				</div>
				<div class="content">
					<div class="header">
					<span>
						Player
						<i class="material-icons">display_settings</i>
					</span>
						<hr>
					</div>
					<span style="display: flex; justify-content: space-between">
					<label for="scale">Scale {scale}</label>
					<input type="range" id="scale" name="scale" min="1" max="3" on:change={changeScale} value="3"/>
				</span>
					<span style="display: flex; justify-content: space-between">
					<label for="speed">Speed {speed}x</label>
					<input type="range" id="speed" name="speed" min="1" max="8" on:change={changeSpeed} value="1">
				</span>
					<div class="btn-row">
						<i class="material-icons" on:click={pause}>
							{#if paused}
								play_arrow
							{:else}
								pause
							{/if}
						</i>
					</div>
					<div class="header">
					<span>
						Graphics
						<i class="material-icons">video_settings</i>
					</span>
						<hr>
					</div>
					<div class="switch">
						<input type="checkbox" id="background" name="background" bind:checked={bgEnabled} on:change={toggleBackground}>
						<label for="background">
							BG Layer
						</label>
					</div>
					<div class="switch">
						<input type="checkbox" id="window" name="window" bind:checked={windowEnabled} on:change={toggleWindow}>
						<label for="window">
							Window Layer
						</label>
					</div>
					<div class="switch">
						<input type="checkbox" id="sprites" name="sprites" bind:checked={spritesEnabled} on:change={toggleSprites}>
						<label for="sprites">
							OBJ Layer
						</label>
					</div>
					<div class="header">
					<span>
						Streaming
						<i class="material-icons">build</i>
					</span>
						<hr>
					</div>
					<div class="switch">
						<input type="checkbox" id="compression" bind:checked={compression} on:change={toggleCompression}/>
						<label for="compression">
							Compression
							{#if compression}
						<span class="suffix" style="float: right" transition:fade={{duration: 150}}>
							{compressionLevelMap[compressionLevel]}
						</span>
							{/if}
						</label>
					</div>
					{#if compression}
						<input
								type="range"
								min="1" max="4"
								transition:fadeSlide={{duration: 150}}
								bind:value={compressionLevel}
								on:change={changeCompressionLevel}
								style="transform: rotateY(180deg)"
						/>
					{/if}
					<div class="switch">
						<input type="checkbox" id="framePatch" bind:checked={framePatching} on:change={toggleFramePatching}/>
						<label for="framePatch">
							Frame Patching
							{#if framePatching}
							<span style="float: right" transition:fade={{duration: 150}}>
								{patchRatio*5}%
							</span>
							{/if}
						</label>
					</div>
					{#if framePatching}
						<!--<input type="range" min="1" max="19" transition:fadeSlide={{duration: 150}} bind:value={patchRatio} on:change={changePatchRatio}/>-->
					{/if}
					<div class="switch">
						<input type="checkbox" id="frameSkip" bind:checked={frameSkipping} on:change={toggleFrameSkipping}/>
						<label for="frameSkip">
							Frame Skipping
						</label>
					</div>
					<div class="switch">
						<input type="checkbox" id="frameCache" bind:checked={frameCaching} on:change={toggleFrameCaching}/>
						<label for="frameCache">
							Frame Caching
						</label>
					</div>
				</div>
			</div>
			<div class="card">
				<div class="headline">
					Statistics
				</div>
				<ul class="content">
					<li class="list-header">
						Frames
						<hr>
					</li>
					<li>
						<i class="material-icons">skip_next</i>
						<span class="label">Skipped</span>
						<span class="list-content">{totalFramesSkipped}</span>
					</li>
					<li>
						<i class="material-icons">difference</i>
						<span class="label">Patched</span>
						<span class="list-content">{totalFramesPatched}</span>
					</li>
					<li>
						<i class="material-icons">data_saver_on</i>
						<span class="label">Saved</span>
						<span class="list-content">{humanFileSize(totalPatchSaved)}</span>
					</li>
					<li class="list-header">
						Network
						<hr>
					</li>
					<li>
						<i class="material-icons">import_export</i>
						<span class="label">Transfer Rate</span>
						<span class="list-content">{humanFileSize(avgPacket)}/s</span>
					</li>
					<li>
						<i class="material-icons">file_download</i>
						<span class="label">Transferred</span>
						<span class="list-content">{humanFileSize(totalTransferred)}</span>
					</li>
					<li>
						<i class="material-icons">expand</i>
						<span class="label">Uncompressed</span>
						<span class="list-content">{humanFileSize(totalUncompressed)}</span>
					</li>
				</ul>

			</div>
		</div>
		<div class="right" style="display: grid; gap: 8px;">
			<div class="card">
				<div class="headline">
					Keybindings
				</div>
				<div class="content">
				<span class="keybinding">
					<code>A</code>
					<button>A</button>
				</span>
					<hr>
					<span class="keybinding">
					<code>B</code>
					<button>B</button>
				</span>
					<span class="keybinding">
					<code></code>
				</span>
				</div>
			</div>
			<div class="card">
				<div class="headline">
					Cache
				</div>
				<div class="content">
					<ul>
						<li class="list-header">
							frames
							<hr>
						</li>
						<li>
							<i class="material-icons">save</i>
							<span class="label">Total</span>
							<span class="list-content">{humanFileSize(cacheSize)}</span>
						</li>
						<li>
							<i class="material-icons">save</i>
							<span class="label">Average</span>
							<span class="list-content">{humanFileSize(avgCacheSize)}</span>
						</li>
						<li>
							<i class="material-icons">save</i>
							<span class="label">Items</span>
							<span class="list-content">{cacheLength}</span>
						</li>
						<li>
							<i class="material-icons">save</i>
							<span class="label">Hits</span>
							<span class="list-content">{cacheHits}</span>
						</li>
						<li class="list-header">
							patches
							<hr>
						</li>
						<li>
							<i class="material-icons">save</i>
							<span class="label">Total</span>
							<span class="list-content">{humanFileSize(patchCacheSize)}</span>
						</li>
						<li>
							<i class="material-icons">save</i>
							<span class="label">Average</span>
							<span class="list-content">{humanFileSize(avgPatchCacheSize)}</span>
						</li>
						<li>
							<i class="material-icons">save</i>
							<span class="label">Items</span>
							<span class="list-content">{patchCacheLength}</span>
						</li>
						<li>
							<i class="material-icons">save</i>
							<span class="label">Hits</span>
							<span class="list-content">{patchCacheHits}</span>
						</li>
					</ul>
				</div>
			</div>
		</div>
	</div>
</main>

<script>
	import { browser } from "$app/environment";
	import {onMount, tick} from "svelte";
	import { fade } from "svelte/transition";
	import { fadeSlide } from "$lib/animations";

	import Socket, { Frame, FramePatch, FrameSkip, Info } from "../socket/game.js";

	import { compressionLevelMap, keyMap } from "$lib/consts";

	let framePatching = true, frameSkipping = true, frameCaching = true;
	let patchRatio = 1
	let compression = true;
	let compressionLevel = 2;
	let connected = false;
	let gameSocket = new Socket("ws://192.168.1.22:8090/");

	let speed = 1
	let scale = "3x";
	let paused = false;
	let bgEnabled = true;
	let windowEnabled = true;
	let spritesEnabled = true;

	let avgPacket = 0
	let frames = 0
	let totalTransferred = 0
	let lastTotalTransferred = 0
	let totalUncompressed = 0

	let framesSkipped = 0
	let framesPatched = 0
	let totalFramesSkipped = 0
	let totalFramesPatched = 0
	let patchSaved = 0
	let totalPatchSaved = 0
	let patchCacheSize = 0;
	let avgPatchCacheSize = 0;
	let patchCacheLength = 0;
	let patchCacheHits = 0;
	let cacheSize = 0;
	let avgCacheSize = 0;
	let cacheLength = 0;
	let cacheHits = 0;
	let canvasElement;
	let canvasCtx;
	let canvasImg;

	const dpadMap = {
		"a": 0,
		"b": 1,
		"select": 2,
		"start": 3,
		"right": 4,
		"left": 5,
		"up": 6,
		"down": 7,
	}

	const dpadKeyMap = {
		"ArrowRight": "right",
		"ArrowLeft": "left",
		"ArrowUp": "up",
		"ArrowDown": "down",
		"a": "a",
		"b": "b"
	}

	onMount(async () => {
		if (browser) {
			await gameSocket.init(async (event, data, msg) => {
				switch (event) {
					case Frame:
						canvasImg.data.set(data)
						canvasCtx.putImageData(canvasImg, 0, 0)
						frames++
						break
					case FramePatch:
						// frame patch
						let changedPixels = 0

						for (let i = 0; i < data.length; i += 4) {
							if (data[i+3] === 255) {
								canvasImg.data.set([data[i], data[i+1], data[i+2], 255], i)
								changedPixels++
							}
						}
						canvasCtx.putImageData(canvasImg, 0, 0)
						frames++
						framesPatched++
						patchSaved += (92160 - (changedPixels * 4))
						break
					case FrameSkip:
						let buffer = new ArrayBuffer(4);
						let view = new Uint8Array(buffer)
						let originalView = new Uint8Array(data.buffer)

						for (let i = 0; i < 4; i++) {
							view[i] = originalView[i]
						}
						let inData = new DataView(buffer)

						framesSkipped += inData.getUint32(0, true)
						patchSaved += (92160 * inData.getUint32(0, true))
						break
					case Info:
						// info message
						switch (data[0]) {
							case 0:
								// pause/play status
								paused = data[1] === 0
								break
							case 1:
								// compression
								compression = data[1] === 1
								break
							case 2:
								// compression level
								compressionLevel = data[1]
								break
							case 3:
								// frame patch
								framePatching = data[1] === 1
								break
							case 4:
								// frame skip
								frameSkipping = data[1] === 1
								break
							case 5:
								// status message
								bgEnabled = !testBit(data[1], 0)
								windowEnabled = !testBit(data[1], 1)
								spritesEnabled = !testBit(data[1], 2)
								compression = testBit(data[1], 3)
								gameSocket.compression = compression
								framePatching = testBit(data[1], 4)
								frameSkipping = testBit(data[1], 5)
								paused = !testBit(data[1], 6)
								frameCaching = !testBit([1], 7)

								compressionLevel = data[2]
								patchRatio = data[3]


								break
							case 6:
								bgEnabled = data[1] === 1
								break
							case 7:
								windowEnabled = data[1] === 1
								break
							case 8:
								spritesEnabled = data[1] === 1
								break
							case 9:
								patchRatio = data[1]
								break
							case 10:
								frameCaching = data[1] === 1
								break
							default:
								console.log(data)
						}
						break
					default:
						console.log(`unknown event type ${event} ${data}`)
				}

			}, async () => {
				connected = true
				await tick() // wait for canvas element to render

				canvasElement.width = 160
				canvasElement.height = 144

				canvasCtx = canvasElement.getContext("2d")
				canvasCtx.imageSmoothingEnabled = false
				canvasImg = canvasCtx.createImageData(160, 144)
			}, () => connected = false)
			$: connected = gameSocket.connected
			document.addEventListener("keydown", event => {
				if (event.key in keyMap) {
					gameSocket.send(new Uint8Array([keyMap[event.key], 1]))

					document.querySelector(`.${dpadKeyMap[event.key]}`).classList.add("on")
				}
			})
			document.addEventListener("keyup", event => {
				if (event.key in keyMap) {
					gameSocket.send(new Uint8Array([keyMap[event.key], 0]))

					document.querySelector(`.${dpadKeyMap[event.key]}`).classList.remove("on")
				}
			})


			setInterval(() => {
				totalTransferred = gameSocket.rawTransfer
				totalUncompressed = gameSocket.uncompressedTransfer

				avgPacket = (totalTransferred - lastTotalTransferred)
				lastTotalTransferred = totalTransferred

				patchCacheSize = gameSocket.patchCache.byteSize()
				avgPatchCacheSize = gameSocket.patchCache.averageSize()
				patchCacheLength = gameSocket.patchCache.length()
				patchCacheHits = gameSocket.patchCache.hits

				cacheSize = gameSocket.frameCache.byteSize()
				avgCacheSize = gameSocket.frameCache.averageSize()
				cacheLength = gameSocket.frameCache.length()
				cacheHits = gameSocket.frameCache.hits

				frames = 0

				totalFramesSkipped += framesSkipped
				totalFramesPatched += framesPatched
				totalPatchSaved += patchSaved

				framesSkipped = 0
				framesPatched = 0
				patchSaved = 0

			}, 1000)

		}
	})

	/**
	 * Format bytes as human-readable text.
	 *
	 * @param bytes Number of bytes.
	 * @param si True to use metric (SI) units, aka powers of 1000. False to use
	 *           binary (IEC), aka powers of 1024.
	 * @param dp Number of decimal places to display.
	 *
	 * @return Formatted string.
	 */
	function humanFileSize(bytes, si=false, dp=1) {
		const thresh = si ? 1000 : 1024;

		if (Math.abs(bytes) < thresh) {
			return bytes + ' B';
		}

		const units = si
				? ['kB', 'MB', 'GB', 'TB', 'PB', 'EB', 'ZB', 'YB']
				: ['KiB', 'MiB', 'GiB', 'TiB', 'PiB', 'EiB', 'ZiB', 'YiB'];
		let u = -1;
		const r = 10**dp;

		do {
			bytes /= thresh;
			++u;
		} while (Math.round(Math.abs(bytes) * r) / r >= thresh && u < units.length - 1);


		return bytes.toFixed(dp) + ' ' + units[u];
	}

	function pause() {
		paused = !paused
		gameSocket.send(new Uint8Array([paused ? 0 : 1]))
	}

	function toggleCompression() {
		gameSocket.send(new Uint8Array([10, 1, compression ? 1 : 0]))
	}

	function changeCompressionLevel() {
		gameSocket.send(new Uint8Array([10, 2, compressionLevel]))
	}

	function changePatchRatio() {
		gameSocket.send(new Uint8Array([10, 9, patchRatio]))
	}

	function toggleFramePatching() {
		gameSocket.send(new Uint8Array([10, 3, framePatching ? 1 : 0]))
	}

	function toggleFrameSkipping() {
		gameSocket.send(new Uint8Array([10, 4, frameSkipping ? 1 : 0]))
	}

	function toggleBackground() {
		gameSocket.send(new Uint8Array([9, 0, bgEnabled ? 1 : 0]))
	}

	function toggleWindow() {
		gameSocket.send(new Uint8Array([9, 1, windowEnabled ? 1 : 0]))
	}

	function toggleSprites() {
		gameSocket.send(new Uint8Array([9, 2, spritesEnabled ? 1 : 0]))
	}

	function dir(event) {
		gameSocket.send(new Uint8Array([dpadMap[event.target.classList[0]], 1]))
		event.preventDefault()
	}

	function unDir(event) {
		gameSocket.send(new Uint8Array([dpadMap[event.target.classList[0]], 0]))
		event.preventDefault()
	}

	function changeScale(event) {
		scale = `${event.target.value}x`
	}

	function changeSpeed(event) {
		speed = event.target.value
	}

	function toggleFrameCaching() {
		gameSocket.send(new Uint8Array([10, 10, frameCaching ? 1 : 0]))
	}

	function testBit(num, bit) {
		return ((num>>bit) %2 !== 0)
	}
</script>
