<ul>
	{#if player !== undefined}
		<li class="list-header">
			{title}
			<hr>
		</li>
		<li>
			<i class="material-icons">save</i>
			<span class="label">Total</span>
			<span class="list-content">{humanFileSize($size)}</span>
		</li>
		<li>
			<i class="material-icons">save</i>
			<span class="label">Average</span>
			<span class="list-content">{humanFileSize($averageSize)}</span>
		</li>
		<li>
			<i class="material-icons">save</i>
			<span class="label">Items</span>
			<span class="list-content">{$length}</span>
		</li>
		<li>
			<i class="material-icons">save</i>
			<span class="label">Hits</span>
			<span class="list-content">{$hits}</span>
		</li>
	{/if}
</ul>

<script lang="ts">
	import {humanFileSize} from "$lib/utils";
	import { Player } from "$lib/game";

	export let player: Player;
	export let title: string = 'cache';

	let { frameCache } = player;

	let { size, hits, length, averageSize } = frameCache;
</script>