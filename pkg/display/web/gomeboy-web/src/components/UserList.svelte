<ul>
	{#if $client.clientPort !== undefined}
		<li class="you" transition:fly|global={{x : -200}}>
			<div class="heading">
			<span class="name">
				{$client.username}
			</span>
				<span style="margin-right: 8px">
				{Math.trunc($client.ping)}ms
			</span>
				<i class="material-icons">edit</i>
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
			</div>
		</li>
	{/if}
	{#each $clients as [clientIP, client]}
		<li class:closing={client.closing} transition:fly={{x: -200}}>
			<div class="list-content">
				<span>
					{client.username}
				</span>
				<span style="margin-right: 8px">
					{Math.trunc(client.ping)}ms
				</span>
				<i class="fa-brands fa-{client.os.toLowerCase()}"></i>

			</div>
		</li>
	{/each}
</ul>

<script>
	import Game from "$lib/game";

	import { fly } from "svelte/transition";

	let { client, clients } = Game;
</script>

<style lang="scss">
	ul {
		margin-top: 4px;
		display: grid;
		gap: 4px;
		li {
			background-color: green;
			border-radius: 4px;
			height: 24px;

			&.closing {
				background-color: red;
			}

			.list-content {
				display: flex;
				font-weight: 300;
				flex-grow: 1;
				padding: 2px 8px;

				:first-child {
					flex-grow: 1;
				}

				font-family: "Roboto", "Ubuntu", "sans-serif";

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
					line-height: 20px;
				}
			}

			display: flex;
			justify-content: space-between;

			&.you {
				background-color: green;
				border-radius: 4px;
				display: grid;
				height: unset;
				font-family: "Roboto", "Ubuntu", "sans-serif";
				overflow: hidden;
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


		}
	}
</style>