<ul>
	{#if $client.clientPort !== undefined}
		<li class="you" out:fly={{x: -200}} in:fly={{y: 200}}>
			<div class="heading">
			<span class="name">
				{$client.username}
			</span>
				<i class="material-icons">cancel</i>
			</div>
			<div class="list-content">
				<ul>
					<li>
						<span class="list-prefix">Address</span>
						<span>{$client.clientIP}:{$client.clientPort}</span>
					</li>
					<li>
						<span class="list-prefix">
							Platform
						</span>
						<span>{$client.os}</span>
					</li>
				</ul>
				<i class="fa-brands fa-{$client.os.toLowerCase()} fa-2xl"></i>
			</div>
		</li>
	{/if}
	{#each $clients as [clientIP, client]}
		{#if client.username !== undefined}
			<li class:closing={client.closing} out:fly|global={{x: -200, delay: 200}} in:fly|global={{y: 200}} >
				<div class="heading">
				<span class="name">
					{client.username}
				</span>
					<i class="material-icons">cancel</i>
				</div>
				<div class="list-content">
					<ul>
						<li>
							<span class="list-prefix">Address</span>
							<span>{client.clientIP}:{client.clientPort}</span>
						</li>
						<li>
						<span class="list-prefix">
							Platform
						</span>
							<span>{client.os}</span>
						</li>
					</ul>
					<i class="fa-brands fa-{client.os.toLowerCase()} fa-2xl"></i>
				</div>
			</li>
		{/if}
	{/each}
</ul>

<script>
	import { fly, slide } from "svelte/transition";
	import {derived} from "svelte/store";
	import Game from "$lib/game";

	let { client, clients, username } = Game;
</script>

<style lang="scss">
	@use "sass:color";

	ul {
		display: grid;
		gap: 8px;
		margin-top: 4px;
	}
	li {
		&.closing {
			background-color: red;

			.heading {
				background-color: darkred;
			}
		}

		&.you {
			outline: 2px solid blueviolet;


		}

		.subheading {
			color: #333333;
			font-size: 10px;
		}

		.list-content {
			display: flex;
			font-weight: 300;
			padding: 2px 8px;

			div {
				display: flex;
				justify-content: space-between;
			}

			ul {
				gap: 2px;

				li {
					background-color: transparent;
					display: flex;
				}
			}

			i {
				line-height: 46px;
			}
		}

		.list-prefix {
			font-weight: 700;
			margin-right: 4px;

			&+span {
				flex-grow: 1;
			}
		}

		.heading {
			background-color: #006600;
			display: flex;
			font-size: 15px;
			line-height: 25px;



			.name {
				color: #fff;
				flex-grow: 1;
				padding: 0 8px;
			}

			i {
				cursor: pointer;
				line-height: 25px;
			}
		}

		background-color: green;
		border-radius: 4px;
		display: grid;
		font-family: "Roboto", "Ubuntu", "sans-serif";
		overflow: hidden;
	}
</style>